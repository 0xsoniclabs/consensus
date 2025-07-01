// Copyright (c) 2025 Fantom Foundation
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
)

type kv struct {
	a, b consensus.EventHash
}

// StronglyReach calculates "sufficient coherence" between the events.
// The A.HighestBefore array remembers the sequence number of the last
// event by each validator that is an ancestor of A. The array for
// B.LowestAfter remembers the sequence number of the earliest
// event by each validator that is a descendant of B. Compare the two arrays,
// and find how many elements in the A.HighestBefore array are greater
// than or equal to the corresponding element of the B.LowestAfter
// array. If there are more than 2n/3 such matches, then the A and B
// have achieved sufficient coherency.
//
// If B1 and B2 are equivocations, then they cannot BOTH strongly-reach any specific event A,
// unless more than 1/3W are Byzantine.
// This great property is the reason why this function exists,
// providing the base for the BFT algorithm.
func (vi *Index) StronglyReach(aID, bID consensus.EventHash) bool {
	vi.InitBranchesInfo()
	// Get events by hash
	aFull := vi.GetHighestBefore(aID)
	if aFull == nil {
		vi.crit(fmt.Errorf("event A=%s not found", aID.String()))
		return false
	}
	a := aFull.VSeq

	// check A doesn't have any reachable equivocations from B
	if vi.AtLeastOneEquivocation() {
		bBranchID := vi.GetEventBranchID(bID)
		if a.Get(bBranchID).IsEquivocationDetected() { // B is reachable as cheater by A
			return false
		}
	}

	// check A has a reachable {QUORUM} of non-cheater-validators that have B as reachable
	b := vi.GetLowestAfter(bID)
	if b == nil {
		vi.crit(fmt.Errorf("event B=%s not found", bID.String()))
		return false
	}

	yes := vi.validators.NewCounter()
	// calculate strongly reaching using the indexes
	branchIDs := vi.BranchesInfo().BranchIDCreatorIdxs
	for branchIDint, creatorIdx := range branchIDs {
		branchID := consensus.ValidatorIndex(branchIDint)

		// bLowestAfter := vi.GetLowestAfterSeq_(bID, branchID)   // lowest event from creator on branchID, which reaches B
		bLowestAfter := b.Get(branchID)   // lowest event from creator on branchID, which reaches B
		aHighestBefore := a.Get(branchID) // highest event from creator, reachable by A

		// if lowest event from branchID which reaches B <= highest from branchID reachable by A
		// then {highest from branchID reachable by A} reaches B
		if bLowestAfter <= aHighestBefore.Seq && bLowestAfter != 0 && !aHighestBefore.IsEquivocationDetected() {
			// we may count the same creator multiple times (on different branches)!
			// so not every call increases the counter
			yes.CountVoteByIndex(creatorIdx)
		}
	}
	return yes.HasQuorum()
}

func (vi *Index) StronglyReachProgress(aID, bID consensus.EventHash, candidateParents, chosenParents consensus.EventHashes) (*consensus.WeightCounter, []*consensus.WeightCounter) {
	// This function is used to determine progress of event bID in strongly reaching aID.
	// It may be used to determine progress toward the strongly reach condition for an event not in vi, but whose parents are in vi.
	// To do so, aID should be the self-parent while chosenParents should be the parents of the not-yet-created event.
	// Further, this function can be used to determine the incremental improvement in progress toward satisfying the strongly
	// reach condition beyond the progress of aId and chosenParents, obtained by inclusion of one additional candidate head at a time.
	// This function is useful in parent selection and event creation timing.

	// The first return is StronglyReach(a + chosenParents, b).
	// The second return argument is a slice containing StronglyReach(a + chosenParents + candidateParent, b) with each element in the
	// slice corresponding to each candidate parent in candidateParents.

	// create the counters that measure the strongly reach progress
	candidateParentsSRProgress := make([]*consensus.WeightCounter, len(candidateParents))
	for i := range candidateParentsSRProgress {
		candidateParentsSRProgress[i] = vi.validators.NewCounter() // initialise the counter for each candidate parent
	}
	chosenParentsSRProgress := vi.validators.NewCounter() // initialise the counter for chosen parents only

	// Get events by hash
	aHBFull := vi.GetHighestBefore(aID)
	if aHBFull == nil {
		vi.crit(fmt.Errorf("event A=%s not found", aID.String()))
		return chosenParentsSRProgress, candidateParentsSRProgress
	}
	aHB := aHBFull.VSeq

	bLA := vi.GetLowestAfter(bID)
	if bLA == nil {
		vi.crit(fmt.Errorf("event B=%s not found", bID.String()))
		return chosenParentsSRProgress, candidateParentsSRProgress
	}

	candidateParentsHB := make([]*HighestBeforeSeq, len(candidateParents))
	for i := range candidateParents {
		hbFull := vi.GetHighestBefore(candidateParents[i])
		if hbFull == nil {
			vi.crit(fmt.Errorf("candidate parent=%s not found", candidateParents[i].String()))
			return chosenParentsSRProgress, candidateParentsSRProgress
		}
		candidateParentsHB[i] = hbFull.VSeq
	}

	chosenParentsHB := make([]*HighestBeforeSeq, len(chosenParents))
	for i := range chosenParents {
		hbFull := vi.GetHighestBefore(chosenParents[i])
		if hbFull == nil {
			vi.crit(fmt.Errorf("chosen parent=%s not found", chosenParents[i].String()))
			return chosenParentsSRProgress, candidateParentsSRProgress
		}
		chosenParentsHB[i] = hbFull.VSeq
	}

	// check A doesn't have any reachable equivocations from B
	if vi.AtLeastOneEquivocation() {
		bBranchID := vi.GetEventBranchID(bID)
		if aHB.Get(bBranchID).IsEquivocationDetected() { // B is reachable as cheater by A
			return chosenParentsSRProgress, candidateParentsSRProgress
		}
	}

	// check chosenParents don't have any reachable equivocations from B
	for i := 0; i < len(chosenParentsHB); i++ {
		if vi.AtLeastOneEquivocation() {
			bBranchID := vi.GetEventBranchID(bID)
			if chosenParentsHB[i].Get(bBranchID).IsEquivocationDetected() { // B is reachable as cheater by a chosen parent
				return chosenParentsSRProgress, candidateParentsSRProgress
			}
		}
	}

	// check candidateParents don't have any reachable equivocations from B
	for i := 0; i < len(candidateParentsHB); i++ {
		if vi.AtLeastOneEquivocation() {
			bBranchID := vi.GetEventBranchID(bID)
			if candidateParentsHB[i].Get(bBranchID).IsEquivocationDetected() { // B is reachable as cheater by a candidate parent
				return chosenParentsSRProgress, candidateParentsSRProgress
			}
		}
	}

	// calculate strongly reaching using the indexes
	branchIDs := vi.BranchesInfo().BranchIDCreatorIdxs
	for branchIDint, creatorIdx := range branchIDs {
		branchID := consensus.ValidatorIndex(branchIDint)

		// bLowestAfter := vi.GetLowestAfterSeq_(bID, branchID)   // lowest event from creator on branchID, which reaches B
		bLowestAfter := bLA.Get(branchID)  // lowest event from creator on branchID, which reaches B
		HighestBefore := aHB.Get(branchID) // highest event from creator, reachable by A

		IsEquivocationDetected := HighestBefore.IsEquivocationDetected()

		for i := range chosenParents {
			chosenParentHighestBefore := chosenParentsHB[i].Get(branchID)                  // highest event from creator, reachable by a chosen parent
			HighestBefore.Seq = maxEvent(HighestBefore.Seq, chosenParentHighestBefore.Seq) // find HighestBefore as reachable by a and all chosen parents
			IsEquivocationDetected = IsEquivocationDetected || chosenParentHighestBefore.IsEquivocationDetected()
		}

		// first do strongly reach for a + chosenParents only
		if bLowestAfter <= HighestBefore.Seq && bLowestAfter != 0 && !IsEquivocationDetected {
			// we may count the same creator multiple times (on different branches)!
			// so not every call increases the counter
			chosenParentsSRProgress.CountVoteByIndex(creatorIdx)
		}
		// now do strongly reach for a + chosenParents + each candidate parent
		for i := range candidateParents {
			candidateParentHighestBefore := candidateParentsHB[i].Get(branchID)
			candidateParentIsEquivocationDetected := IsEquivocationDetected || candidateParentHighestBefore.IsEquivocationDetected()
			candidateParentHighestBefore.Seq = maxEvent(HighestBefore.Seq, candidateParentHighestBefore.Seq)

			if bLowestAfter <= candidateParentHighestBefore.Seq && bLowestAfter != 0 && !candidateParentIsEquivocationDetected {
				// we may count the same creator multiple times (on different branches)!
				// so not every call increases the counter
				candidateParentsSRProgress[i].CountVoteByIndex(creatorIdx)
			}
		}
	}
	// We want SR progress for new candidate events with parents aID + chosenParents + head
	// aID may not contribute to strongly reach without the heads,
	// but may contribute with the heads. HighestBefore and LowestAfter used above do not incorporate
	// these potential new events, so ensure the contribution of aID's creator is checked and made here
	aCreatorID := vi.getEvent(aID).Creator()
	for _, SR := range candidateParentsSRProgress {
		if SR.Sum() > 0 { // if anything in candidate event's subgraph reaches bID, then the candidate must too
			SR.CountVoteByID(aCreatorID)
		}
	}
	return chosenParentsSRProgress, candidateParentsSRProgress
}

func maxEvent(a consensus.Seq, b consensus.Seq) consensus.Seq {
	if a > b {
		return a
	}
	return b
}
