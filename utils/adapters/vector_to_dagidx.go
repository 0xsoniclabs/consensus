// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package adapters

import (
	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/dagidx"
	"github.com/0xsoniclabs/consensus/dagindexer"
)

type VectorSeqToDagIndexSeq struct {
	*dagindexer.HighestBeforeSeq
}

type BranchSeq struct {
	dagindexer.BranchSeq
}

// Seq is a maximum observed e.Seq in the branch
func (b *BranchSeq) Seq() consensus.Seq {
	return b.BranchSeq.Seq
}

// MinSeq is a minimum observed e.Seq in the branch
func (b *BranchSeq) MinSeq() consensus.Seq {
	return b.BranchSeq.MinSeq
}

// Get i's position in the byte-encoded vector clock
func (b VectorSeqToDagIndexSeq) Get(i consensus.ValidatorIndex) dagidx.Seq {
	seq := b.HighestBeforeSeq.Get(i)
	return &BranchSeq{seq}
}

type VectorToDagIndexer struct {
	*dagindexer.Index
}

func (v *VectorToDagIndexer) GetMergedHighestBefore(id consensus.EventHash) dagidx.HighestBeforeSeq {
	return VectorSeqToDagIndexSeq{v.Index.GetMergedHighestBefore(id).VSeq}
}
