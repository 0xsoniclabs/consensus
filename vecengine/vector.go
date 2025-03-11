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

	"github.com/0xsoniclabs/consensus/ctype"
)

type LowestAfterI interface {
	InitWithEvent(i ctype.ValidatorIdx, e ctype.Event)
	Visit(i ctype.ValidatorIdx, e ctype.Event) bool
}

type HighestBeforeI interface {
	InitWithEvent(i ctype.ValidatorIdx, e ctype.Event)
	IsEmpty(i ctype.ValidatorIdx) bool
	IsForkDetected(i ctype.ValidatorIdx) bool
	Seq(i ctype.ValidatorIdx) ctype.Seq
	MinSeq(i ctype.ValidatorIdx) ctype.Seq
	SetForkDetected(i ctype.ValidatorIdx)
	CollectFrom(other HighestBeforeI, branches ctype.ValidatorIdx)
	GatherFrom(to ctype.ValidatorIdx, other HighestBeforeI, from []ctype.ValidatorIdx)
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
		Seq    ctype.Seq
		MinSeq ctype.Seq
	}
)

// NewLowestAfterSeq creates new LowestAfterSeq vector.
func NewLowestAfterSeq(size ctype.ValidatorIdx) *LowestAfterSeq {
	b := make(LowestAfterSeq, size*4)
	return &b
}

// NewHighestBeforeSeq creates new HighestBeforeSeq vector.
func NewHighestBeforeSeq(size ctype.ValidatorIdx) *HighestBeforeSeq {
	b := make(HighestBeforeSeq, size*8)
	return &b
}

// Get i's position in the byte-encoded vector clock
func (b LowestAfterSeq) Get(i ctype.ValidatorIdx) ctype.Seq {
	for i >= b.Size() {
		return 0
	}
	return ctype.Seq(binary.LittleEndian.Uint32(b[i*4 : (i+1)*4]))
}

// Size of the vector clock
func (b LowestAfterSeq) Size() ctype.ValidatorIdx {
	return ctype.ValidatorIdx(len(b)) / 4
}

// Set i's position in the byte-encoded vector clock
func (b *LowestAfterSeq) Set(i ctype.ValidatorIdx, seq ctype.Seq) {
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
func (b HighestBeforeSeq) Get(i ctype.ValidatorIdx) BranchSeq {
	for int(i) >= b.Size() {
		return BranchSeq{}
	}
	seq1 := binary.LittleEndian.Uint32(b[i*8 : i*8+4])
	seq2 := binary.LittleEndian.Uint32(b[i*8+4 : i*8+8])

	return BranchSeq{
		Seq:    ctype.Seq(seq1),
		MinSeq: ctype.Seq(seq2),
	}
}

// Set i's position in the byte-encoded vector clock
func (b *HighestBeforeSeq) Set(i ctype.ValidatorIdx, seq BranchSeq) {
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
		MinSeq: ctype.Seq(math.MaxInt32),
	}
)

// IsForkDetected returns true if observed fork by a creator (in combination of branches)
func (seq BranchSeq) IsForkDetected() bool {
	return seq == forkDetectedSeq
}
