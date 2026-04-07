// Copyright (c) 2026 Sonic Operations Ltd
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package dagindexer

import (
	"fmt"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/dagindexer/dagbranch"
	"github.com/0xsoniclabs/consensus/consensus/dagindexer/dagstore"
	"github.com/0xsoniclabs/consensus/consensus/vecflushable"
	"github.com/0xsoniclabs/kvdb"
)

// Index is a data to detect strongly-reach condition, calculate median timestamp, detect equivocations.
type Index struct {
	crit          func(error)
	validators    *consensus.Validators
	validatorIdxs map[consensus.ValidatorID]consensus.ValidatorIndex

	branches *dagbranch.Tracker
	getEvent func(consensus.EventHash) consensus.Event
	store    *dagstore.Store
	cfg      IndexConfig
}

// NewIndex creates Index instance.
func NewIndex(crit func(error), config IndexConfig) *Index {
	vi := &Index{
		cfg:  config,
		crit: crit,
		store: dagstore.NewStore(crit, dagstore.CacheConfig{
			HighestBeforeTimeSize: config.Caches.HighestBeforeTimeSize,
			HighestBeforeSeqSize:  config.Caches.HighestBeforeSeqSize,
			LowestAfterSeqSize:    config.Caches.LowestAfterSeqSize,
		}),
	}
	return vi
}

// Add calculates vector clocks for the event and saves into DB.
func (vi *Index) Add(e consensus.Event) error {
	vi.InitBranchesInfo()
	_, err := vi.fillEventVectors(e)
	return err
}

// Flush writes vector clocks to persistent store.
func (vi *Index) Flush() {
	vi.branches.Flush()
	if err := vi.store.Db.Flush(); err != nil {
		vi.crit(err)
	}
}

// DropNotFlushed not connected clocks. Call it if event has failed.
func (vi *Index) DropNotFlushed() {
	if vi.branches != nil {
		vi.branches.DropNotFlushed()
	}
	if vi.store.Db.NotFlushedPairs() != 0 {
		vi.store.Db.DropNotFlushed()
		vi.store.OnDropNotFlushed()
	}
}

func (vi *Index) WrapWithFlushable(db kvdb.Store) kvdb.FlushableKVStore {
	return vecflushable.Wrap(db, vi.cfg.Caches.DBCache)
}

// Reset resets buffers.
func (vi *Index) Reset(validators *consensus.Validators, db kvdb.FlushableKVStore, getEvent func(consensus.EventHash) consensus.Event) {
	vi.getEvent = getEvent
	vi.validators = validators
	vi.validatorIdxs = validators.Idxs()
	vi.store.InitTables(db)
	vi.branches = dagbranch.NewTracker(vi.crit, validators, vi.store)
	vi.DropNotFlushed()
	vi.store.OnDropNotFlushed()
}

func (vi *Index) Close() error {
	return vi.store.Db.Close()
}

// fillEventVectors calculates (and stores) event's vectors, and updates LowestAfter of newly-reachable events.
func (vi *Index) fillEventVectors(e consensus.Event) (AllVecs, error) {
	meIdx := vi.validatorIdxs[e.Creator()]
	branchCount := vi.branches.BranchCount()

	myVecs := AllVecs{
		Before: NewHighestBefore(branchCount),
		After:  NewLowestAfterSeq(branchCount),
	}

	meBranchID, err := vi.branches.AssignBranchID(e, meIdx)
	if err != nil {
		return myVecs, err
	}

	// pre-load parents into RAM for quick access
	parentsVecs := make([]*HighestBefore, len(e.Parents()))
	parentsBranchIDs := make([]consensus.ValidatorIndex, len(e.Parents()))
	for i, p := range e.Parents() {
		parentsBranchIDs[i] = vi.GetEventBranchID(p)
		parentsVecs[i] = vi.GetHighestBefore(p)
		if parentsVecs[i] == nil {
			return myVecs, fmt.Errorf("processed out of order, parent not found (inconsistent DB), parent=%s", p.String())
		}
	}

	// reachable by himself
	myVecs.After.InitWithEvent(meBranchID, e)
	myVecs.Before.InitWithEvent(meBranchID, e)
	for _, pVec := range parentsVecs {
		// calculate HighestBefore  Detect equivocations for a case when parent reaches an equivocation
		myVecs.Before.CollectFrom(pVec, branchCount)
	}

	// Detect equivocations, which were not reachable by parents
	vi.branches.DetectEquivocations(myVecs.Before)

	// graph traversal starting from e, but excluding e
	onWalk := func(walk consensus.EventHash) (godeeper bool) {
		wLowestAfterSeq := vi.GetLowestAfter(walk)
		// update LowestAfter vector of the old event, because newly-connected event reaches it
		if wLowestAfterSeq.Visit(meBranchID, e) {
			vi.SetLowestAfter(walk, wLowestAfterSeq)
			return true
		}
		return false
	}
	err = vi.DfsSubgraph(e, onWalk)
	if err != nil {
		vi.crit(err)
	}

	// store calculated vectors
	vi.SetHighestBefore(e.ID(), myVecs.Before)
	vi.SetLowestAfter(e.ID(), myVecs.After)
	vi.branches.SetEventBranchID(e.ID(), meBranchID)

	return myVecs, nil
}

// GetMergedHighestBefore returns HighestBefore vector clock without branches, where branches are merged into one
func (vi *Index) GetMergedHighestBefore(id consensus.EventHash) *HighestBefore {
	return vi.branches.GetMergedHighestBefore(id, vi.GetHighestBefore)
}

// InitBranchesInfo loads BranchesInfo from store
func (vi *Index) InitBranchesInfo() {
	vi.branches.Init()
}

func (vi *Index) AtLeastOneEquivocation() bool {
	return vi.branches.AtLeastOneEquivocation()
}

func (vi *Index) BranchesInfo() *BranchesInfo {
	return vi.branches.Info()
}
