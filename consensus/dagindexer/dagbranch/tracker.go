// Copyright (c) 2026 Sonic Operations Ltd
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package dagbranch

import (
	"errors"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/dagindexer/dagvec"
)

// BranchStore is the persistence interface the Tracker needs.
type BranchStore interface {
	GetEventBranchID(id consensus.EventHash) consensus.ValidatorIndex
	SetEventBranchID(id consensus.EventHash, branchID consensus.ValidatorIndex)
	GetBranchesInfoBytes() []byte
	SetBranchesInfoBytes(data []byte)
}

// Tracker manages branch IDs and equivocation detection.
// It owns the in-memory BranchesInfo and delegates persistence to a BranchStore.
type Tracker struct {
	crit          func(error)
	validators    *consensus.Validators
	validatorIdxs map[consensus.ValidatorID]consensus.ValidatorIndex
	info          *BranchesInfo
	store         BranchStore
}

// NewTracker creates a Tracker bound to the given store and validator set.
func NewTracker(crit func(error), validators *consensus.Validators, store BranchStore) *Tracker {
	return &Tracker{
		crit:          crit,
		validators:    validators,
		validatorIdxs: validators.Idxs(),
		store:         store,
	}
}

// Init loads BranchesInfo from the store if not already cached.
func (t *Tracker) Init() {
	if t.info == nil {
		buf := t.store.GetBranchesInfoBytes()
		if buf != nil {
			t.info = &BranchesInfo{}
			if err := rlp.DecodeBytes(buf, t.info); err != nil {
				t.crit(err)
			}
		}
		if t.info == nil {
			t.info = NewInitialBranchesInfo(t.validators)
		}
	}
}

// Flush persists the cached BranchesInfo to the store.
func (t *Tracker) Flush() {
	if t.info != nil {
		buf, err := rlp.EncodeToBytes(t.info)
		if err != nil {
			t.crit(err)
		}
		t.store.SetBranchesInfoBytes(buf)
	}
}

// DropNotFlushed discards the in-memory BranchesInfo.
func (t *Tracker) DropNotFlushed() {
	t.info = nil
}

// Info returns the current BranchesInfo.
func (t *Tracker) Info() *BranchesInfo {
	return t.info
}

// AtLeastOneEquivocation reports whether any equivocation has been detected.
func (t *Tracker) AtLeastOneEquivocation() bool {
	return t.info.AtLeastOneEquivocation(t.validators.Len())
}

// BranchCount returns the total number of branch IDs (validators + equivocal branches).
func (t *Tracker) BranchCount() consensus.ValidatorIndex {
	return consensus.ValidatorIndex(len(t.info.BranchIDCreatorIdxs))
}

// GetEventBranchID delegates to the store.
func (t *Tracker) GetEventBranchID(id consensus.EventHash) consensus.ValidatorIndex {
	return t.store.GetEventBranchID(id)
}

// SetEventBranchID delegates to the store.
func (t *Tracker) SetEventBranchID(id consensus.EventHash, branchID consensus.ValidatorIndex) {
	t.store.SetEventBranchID(id, branchID)
}

// AssignBranchID determines the branch ID for a new event.
// If the event represents a new equivocation, a new branch ID is allocated.
func (t *Tracker) AssignBranchID(e consensus.Event, meIdx consensus.ValidatorIndex) (consensus.ValidatorIndex, error) {
	if len(t.info.BranchIDCreatorIdxs) != len(t.info.BranchIDLastSeq) {
		return 0, errors.New("inconsistent BranchIDCreators len (inconsistent DB)")
	}
	if consensus.ValidatorIndex(len(t.info.BranchIDCreatorIdxs)) < t.validators.Len() {
		return 0, errors.New("inconsistent BranchIDCreators len (inconsistent DB)")
	}

	if e.SelfParent() == nil {
		// is it first event indeed?
		if t.info.BranchIDLastSeq[meIdx] == 0 {
			// OK, not a new equivocation
			t.info.BranchIDLastSeq[meIdx] = e.Seq()
			return meIdx, nil
		}
	} else {
		selfParentBranchID := t.store.GetEventBranchID(*e.SelfParent())
		// sanity checks
		if len(t.info.BranchIDCreatorIdxs) != len(t.info.BranchIDLastSeq) {
			return 0, errors.New("inconsistent BranchIDCreators len (inconsistent DB)")
		}
		if t.info.BranchIDLastSeq[selfParentBranchID]+1 == e.Seq() {
			t.info.BranchIDLastSeq[selfParentBranchID] = e.Seq()
			// OK, not a new equivocation
			return selfParentBranchID, nil
		}
	}

	// if we're here, then new equivocation is reachable (only globally), create new branchID due to a new equivocation
	t.info.BranchIDLastSeq = append(t.info.BranchIDLastSeq, e.Seq())
	t.info.BranchIDCreatorIdxs = append(t.info.BranchIDCreatorIdxs, meIdx)
	newBranchID := consensus.ValidatorIndex(len(t.info.BranchIDLastSeq) - 1)
	t.info.BranchIDByCreators[meIdx] = append(t.info.BranchIDByCreators[meIdx], newBranchID)
	return newBranchID, nil
}

// SetEquivocationDetected marks all branches of the creator identified by branchID
// as having detected an equivocation in the given HighestBefore vector.
func (t *Tracker) SetEquivocationDetected(before *dagvec.HighestBefore, branchID consensus.ValidatorIndex) {
	creatorIdx := t.info.BranchIDCreatorIdxs[branchID]
	for _, bid := range t.info.BranchIDByCreators[creatorIdx] {
		before.SetEquivocationDetected(bid)
	}
}

// DetectEquivocations updates the HighestBefore vector for a newly added event
// by propagating equivocation markers from parents and detecting new equivocations
// via cross-branch sequence comparison.
func (t *Tracker) DetectEquivocations(before *dagvec.HighestBefore) {
	if !t.AtLeastOneEquivocation() {
		return
	}
	for n := consensus.ValidatorIndex(0); n < t.validators.Len(); n++ {
		if len(t.info.BranchIDByCreators[n]) <= 1 {
			continue
		}
		for _, branchID := range t.info.BranchIDByCreators[n] {
			if before.IsEquivocationDetected(branchID) {
				// if one branch reaches an equivocation, mark all the branches as observing the equivocation
				t.SetEquivocationDetected(before, n)
				break
			}
		}
	}
nextCreator:
	for n := consensus.ValidatorIndex(0); n < t.validators.Len(); n++ {
		if before.IsEquivocationDetected(n) {
			continue
		}
		for _, branchID1 := range t.info.BranchIDByCreators[n] {
			for _, branchID2 := range t.info.BranchIDByCreators[n] {
				a := branchID1
				b := branchID2
				if a == b {
					continue
				}
				if before.IsEmpty(a) || before.IsEmpty(b) {
					continue
				}
				if before.MinSeq(a) <= before.Seq(b) && before.MinSeq(b) <= before.Seq(a) {
					t.SetEquivocationDetected(before, n)
					continue nextCreator
				}
			}
		}
	}
}

// GetMergedHighestBefore returns a HighestBefore vector with branches merged
// into one entry per validator. If no equivocations exist, returns the original.
func (t *Tracker) GetMergedHighestBefore(id consensus.EventHash, getHighestBefore func(consensus.EventHash) *dagvec.HighestBefore) *dagvec.HighestBefore {
	t.Init()
	if t.AtLeastOneEquivocation() {
		scatteredBefore := getHighestBefore(id)
		mergedBefore := dagvec.NewHighestBefore(t.validators.Len())
		for creatorIdx, branches := range t.info.BranchIDByCreators {
			mergedBefore.GatherFrom(consensus.ValidatorIndex(creatorIdx), scatteredBefore, branches)
		}
		return mergedBefore
	}
	return getHighestBefore(id)
}
