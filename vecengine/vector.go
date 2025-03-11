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
	"encoding/binary"
	"math"

	"github.com/0xsoniclabs/consensus/inter/dag"
	"github.com/0xsoniclabs/consensus/inter/idx"
)

type LowestAfterI interface {
	InitWithEvent(i idx.ValidatorIdx, e dag.Event)
	Visit(i idx.ValidatorIdx, e dag.Event) bool
}

type HighestBeforeI interface {
	InitWithEvent(i idx.ValidatorIdx, e dag.Event)
	IsEmpty(i idx.ValidatorIdx) bool
	IsForkDetected(i idx.ValidatorIdx) bool
	Seq(i idx.ValidatorIdx) idx.Seq
	MinSeq(i idx.ValidatorIdx) idx.Seq
	SetForkDetected(i idx.ValidatorIdx)
	CollectFrom(other HighestBeforeI, branches idx.ValidatorIdx)
	GatherFrom(to idx.ValidatorIdx, other HighestBeforeI, from []idx.ValidatorIdx)
}

type allVecs struct {
	after  LowestAfterI
	before HighestBeforeI
}

/*
 * Use binary form for optimization, to avoid serialization. As a result, DB cache works as elements cache.
 */

type (
	// LowestAfterSeq is a vector of lowest events (their Seq) which do observe the source event
	LowestAfterSeq []byte
	// HighestBeforeSeq is a vector of highest events (their Seq + IsForkDetected) which are observed by source event
	HighestBeforeSeq []byte

	// BranchSeq encodes Seq and MinSeq into 8 bytes
	BranchSeq struct {
		Seq    idx.Seq
		MinSeq idx.Seq
	}
)

// NewLowestAfterSeq creates new LowestAfterSeq vector.
func NewLowestAfterSeq(size idx.ValidatorIdx) *LowestAfterSeq {
	b := make(LowestAfterSeq, size*4)
	return &b
}

// NewHighestBeforeSeq creates new HighestBeforeSeq vector.
func NewHighestBeforeSeq(size idx.ValidatorIdx) *HighestBeforeSeq {
	b := make(HighestBeforeSeq, size*8)
	return &b
}

// Get i's position in the byte-encoded vector clock
func (b LowestAfterSeq) Get(i idx.ValidatorIdx) idx.Seq {
	for i >= b.Size() {
		return 0
	}
	return idx.Seq(binary.LittleEndian.Uint32(b[i*4 : (i+1)*4]))
}

// Size of the vector clock
func (b LowestAfterSeq) Size() idx.ValidatorIdx {
	return idx.ValidatorIdx(len(b)) / 4
}

// Set i's position in the byte-encoded vector clock
func (b *LowestAfterSeq) Set(i idx.ValidatorIdx, seq idx.Seq) {
	for i >= b.Size() {
		// append zeros if exceeds size
		*b = append(*b, []byte{0, 0, 0, 0}...)
	}

	binary.LittleEndian.PutUint32((*b)[i*4:(i+1)*4], uint32(seq))
}

// Size of the vector clock
func (b HighestBeforeSeq) Size() int {
	return len(b) / 8
}

// Get i's position in the byte-encoded vector clock
func (b HighestBeforeSeq) Get(i idx.ValidatorIdx) BranchSeq {
	for int(i) >= b.Size() {
		return BranchSeq{}
	}
	seq1 := binary.LittleEndian.Uint32(b[i*8 : i*8+4])
	seq2 := binary.LittleEndian.Uint32(b[i*8+4 : i*8+8])

	return BranchSeq{
		Seq:    idx.Seq(seq1),
		MinSeq: idx.Seq(seq2),
	}
}

// Set i's position in the byte-encoded vector clock
func (b *HighestBeforeSeq) Set(i idx.ValidatorIdx, seq BranchSeq) {
	for int(i) >= b.Size() {
		// append zeros if exceeds size
		*b = append(*b, []byte{0, 0, 0, 0, 0, 0, 0, 0}...)
	}
	binary.LittleEndian.PutUint32((*b)[i*8:i*8+4], uint32(seq.Seq))
	binary.LittleEndian.PutUint32((*b)[i*8+4:i*8+8], uint32(seq.MinSeq))
}

var (
	// forkDetectedSeq is a special marker of observed fork by a creator
	forkDetectedSeq = BranchSeq{
		Seq:    0,
		MinSeq: idx.Seq(math.MaxInt32),
	}
)

// IsForkDetected returns true if observed fork by a creator (in combination of branches)
func (seq BranchSeq) IsForkDetected() bool {
	return seq == forkDetectedSeq
}
