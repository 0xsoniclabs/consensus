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
func (b *BranchSeq) Seq() idx.Event {
	return b.BranchSeq.Seq
}

// MinSeq is a minimum observed e.Seq in the branch
func (b *BranchSeq) MinSeq() idx.Event {
	return b.BranchSeq.MinSeq
}

// Get i's position in the byte-encoded vector clock
func (b VectorSeqToDagIndexSeq) Get(i idx.Validator) dagidx.Seq {
	seq := b.HighestBeforeSeq.Get(i)
	return &BranchSeq{seq}
}

type VectorToDagIndexer struct {
	*vecengine.Engine
}

func (v *VectorToDagIndexer) GetMergedHighestBefore(id hash.Event) dagidx.HighestBeforeSeq {
	return VectorSeqToDagIndexSeq{v.Engine.GetMergedHighestBefore(id).(*vecengine.HighestBeforeSeq)}
}
