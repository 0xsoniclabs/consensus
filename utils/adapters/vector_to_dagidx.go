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
	"github.com/0xsoniclabs/consensus/abft/dagidx"
	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/vecengine"
)

type VectorSeqToDagIndexSeq struct {
	*vecengine.HighestBeforeSeq
}

type BranchSeq struct {
	vecengine.BranchSeq
}

// Seq is a maximum observed e.Seq in the branch
func (b *BranchSeq) Seq() idx.Seq {
	return b.BranchSeq.Seq
}

// MinSeq is a minimum observed e.Seq in the branch
func (b *BranchSeq) MinSeq() idx.Seq {
	return b.BranchSeq.MinSeq
}

// Get i's position in the byte-encoded vector clock
func (b VectorSeqToDagIndexSeq) Get(i idx.ValidatorIdx) dagidx.Seq {
	seq := b.HighestBeforeSeq.Get(i)
	return &BranchSeq{seq}
}

type VectorToDagIndexer struct {
	*vecengine.Engine
}

func (v *VectorToDagIndexer) GetMergedHighestBefore(id hash.EventHash) dagidx.HighestBeforeSeq {
	return VectorSeqToDagIndexSeq{v.Engine.GetMergedHighestBefore(id).(*vecengine.HighestBeforeSeq)}
}
