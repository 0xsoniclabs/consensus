// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensus

import "github.com/0xsoniclabs/consensus/utils/byteutils"

type (
	// Epoch numeration.
	Epoch uint32

	// Seq numeration.
	Seq uint32

	// BlockID numeration.
	BlockID uint64

	// Lamport numeration.
	Lamport uint32

	// Frame numeration.
	Frame uint32

	// Pack numeration.
	Pack uint32

	// ValidatorID numeration.
	ValidatorID uint32

	// ValidatorIndex represents a normalized value of ValidatorID for slice/array packing purposes
	ValidatorIndex uint32
)

const (
	FirstFrame = Frame(1)
	FirstEpoch = Epoch(1)
)

// Bytes gets the byte representation of the index.
func (e Epoch) Bytes() []byte {
	return byteutils.Uint32ToBigEndian(uint32(e))
}

// Bytes gets the byte representation of the index.
func (e Seq) Bytes() []byte {
	return byteutils.Uint32ToBigEndian(uint32(e))
}

// Bytes gets the byte representation of the index.
func (b BlockID) Bytes() []byte {
	return byteutils.Uint64ToBigEndian(uint64(b))
}

// Bytes gets the byte representation of the index.
func (l Lamport) Bytes() []byte {
	return byteutils.Uint32ToBigEndian(uint32(l))
}

// Bytes gets the byte representation of the index.
func (s ValidatorID) Bytes() []byte {
	return byteutils.Uint32ToBigEndian(uint32(s))
}

// Bytes gets the byte representation of the index.
func (f Frame) Bytes() []byte {
	return byteutils.Uint32ToBigEndian(uint32(f))
}

// BytesToEpoch converts bytes to epoch index.
func BytesToEpoch(b []byte) Epoch {
	return Epoch(byteutils.BigEndianToUint32(b))
}

// BytesToBlock converts bytes to block index.
func BytesToBlock(b []byte) BlockID {
	return BlockID(byteutils.BigEndianToUint64(b))
}

// BytesToLamport converts bytes to block index.
func BytesToLamport(b []byte) Lamport {
	return Lamport(byteutils.BigEndianToUint32(b))
}

// BytesToFrame converts bytes to block index.
func BytesToFrame(b []byte) Frame {
	return Frame(byteutils.BigEndianToUint32(b))
}

// BytesToValidatorID converts bytes to validator index.
func BytesToValidatorID(b []byte) ValidatorID {
	return ValidatorID(byteutils.BigEndianToUint32(b))
}

// MaxLamport return max value
func MaxLamport(x, y Lamport) Lamport {
	if x > y {
		return x
	}
	return y
}

// Bytes gets the byte representation of the index.
func (v ValidatorIndex) Bytes() []byte {
	return byteutils.Uint32ToBigEndian(uint32(v))
}

// BytesToValidator converts bytes to validator index.
func BytesToValidator(b []byte) ValidatorIndex {
	return ValidatorIndex(byteutils.BigEndianToUint32(b))
}
