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
	"errors"

	"github.com/0xsoniclabs/consensus/consensus"
)

// NoCheaters excludes events which are reachable by selfParents as cheaters.
// Called by emitter to exclude cheater's events from potential parents list.
func (vi *Index) NoCheaters(selfParent *consensus.EventHash, options consensus.EventHashes) consensus.EventHashes {
	if selfParent == nil {
		return options
	}
	vi.initBranchesInfo()

	if !vi.atLeastOneEquivocation() {
		return options
	}

	// no need to merge, because every branch is marked by IsEquivocationDetected if equivocation is reachable
	highest := vi.getHighestBefore(*selfParent)
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
