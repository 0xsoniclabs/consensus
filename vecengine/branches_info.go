// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package vecengine

import (
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/inter/pos"
)

// BranchesInfo contains information about global branches of each validator
type BranchesInfo struct {
	BranchIDLastSeq     []idx.Event       // branchID -> highest e.Seq in the branch
	BranchIDCreatorIdxs []idx.Validator   // branchID -> validator idx
	BranchIDByCreators  [][]idx.Validator // validator idx -> list of branch IDs
}

// InitBranchesInfo loads BranchesInfo from store
func (vi *Engine) InitBranchesInfo() {
	if vi.bi == nil {
		// if not cached
		vi.bi = vi.getBranchesInfo()
		if vi.bi == nil {
			// first run
			vi.bi = newInitialBranchesInfo(vi.validators)
		}
	}
}

func newInitialBranchesInfo(validators *pos.Validators) *BranchesInfo {
	branchIDCreators := validators.SortedIDs()
	branchIDCreatorIdxs := make([]idx.Validator, len(branchIDCreators))
	for i := range branchIDCreators {
		branchIDCreatorIdxs[i] = idx.Validator(i)
	}

	branchIDLastSeq := make([]idx.Event, len(branchIDCreatorIdxs))
	branchIDByCreators := make([][]idx.Validator, validators.Len())
	for i := range branchIDByCreators {
		branchIDByCreators[i] = make([]idx.Validator, 1, validators.Len()/2+1)
		branchIDByCreators[i][0] = idx.Validator(i)
	}
	return &BranchesInfo{
		BranchIDLastSeq:     branchIDLastSeq,
		BranchIDCreatorIdxs: branchIDCreatorIdxs,
		BranchIDByCreators:  branchIDByCreators,
	}
}

func (vi *Engine) AtLeastOneFork() bool {
	return idx.Validator(len(vi.bi.BranchIDCreatorIdxs)) > vi.validators.Len()
}

func (vi *Engine) BranchesInfo() *BranchesInfo {
	return vi.bi
}
