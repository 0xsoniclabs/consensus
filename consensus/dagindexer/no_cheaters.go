package dagindexer

import (
	"errors"

	"github.com/0xsoniclabs/consensus/consensus"
)

// NoCheaters excludes events which are reachable by selfParents as cheaters.
// Called by emitter to exclude cheater's events from potential parents list.
func (vi *Index) NoCheaters(selfParent *consensus.EventHash, options consensus.EventHashes) consensus.EventHashes {
	if selfParent == nil {
		return options
	}
	vi.InitBranchesInfo()

	if !vi.AtLeastOneEquivocation() {
		return options
	}

	// no need to merge, because every branch is marked by IsEquivocationDetected if equivocation is reachable
	highest := vi.GetHighestBefore(*selfParent)
	filtered := make(consensus.EventHashes, 0, len(options))
	for _, id := range options {
		e := vi.getEvent(id)
		if e == nil {
			vi.crit(errors.New("event not found"))
		}
		if !highest.VSeq.Get(vi.validatorIdxs[e.Creator()]).IsEquivocationDetected() {
			filtered.Add(id)
		}
	}
	return filtered
}
