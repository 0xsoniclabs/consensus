// Copyright (c) 2026 Sonic Operations Ltd
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package dagindexer

import "github.com/0xsoniclabs/consensus/consensus"

// Forwarding methods that delegate to vi.store for convenience.
// These keep the public API of Index unchanged and allow algorithm
// files and tests to call vi.GetHighestBefore etc. directly.

func (vi *Index) GetHighestBefore(id consensus.EventHash) *HighestBefore {
	return vi.store.GetHighestBefore(id)
}

func (vi *Index) GetLowestAfter(id consensus.EventHash) *LowestAfter {
	return vi.store.GetLowestAfter(id)
}

func (vi *Index) SetHighestBefore(id consensus.EventHash, vec *HighestBefore) {
	vi.store.SetHighestBefore(id, vec)
}

func (vi *Index) SetLowestAfter(id consensus.EventHash, seq *LowestAfterSeq) {
	vi.store.SetLowestAfter(id, seq)
}

func (vi *Index) GetEventBranchID(id consensus.EventHash) consensus.ValidatorIndex {
	return vi.store.GetEventBranchID(id)
}

func (vi *Index) SetEventBranchID(id consensus.EventHash, branchID consensus.ValidatorIndex) {
	vi.store.SetEventBranchID(id, branchID)
}
