// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package idx

import (
	"github.com/0xsoniclabs/consensus/common/bigendian"
)

type (
	// Epoch numeration.
	Epoch uint32

	// Event numeration.
	Event uint32

	// Block numeration.
	Block uint64

	// Lamport numeration.
	Lamport uint32

	// Frame numeration.
	Frame uint32

	// Pack numeration.
	Pack uint32

	// ValidatorID numeration.
	ValidatorID uint32
)

// Bytes gets the byte representation of the index.
func (e Epoch) Bytes() []byte {
	return bigendian.Uint32ToBytes(uint32(e))
}

// Bytes gets the byte representation of the index.
func (e Event) Bytes() []byte {
	return bigendian.Uint32ToBytes(uint32(e))
}

// Bytes gets the byte representation of the index.
func (b Block) Bytes() []byte {
	return bigendian.Uint64ToBytes(uint64(b))
}

// Bytes gets the byte representation of the index.
func (l Lamport) Bytes() []byte {
	return bigendian.Uint32ToBytes(uint32(l))
}

// Bytes gets the byte representation of the index.
func (p Pack) Bytes() []byte {
	return bigendian.Uint32ToBytes(uint32(p))
}

// Bytes gets the byte representation of the index.
func (s ValidatorID) Bytes() []byte {
	return bigendian.Uint32ToBytes(uint32(s))
}

// Bytes gets the byte representation of the index.
func (f Frame) Bytes() []byte {
	return bigendian.Uint32ToBytes(uint32(f))
}

// BytesToEpoch converts bytes to epoch index.
func BytesToEpoch(b []byte) Epoch {
	return Epoch(bigendian.BytesToUint32(b))
}

// BytesToEvent converts bytes to event index.
func BytesToEvent(b []byte) Event {
	return Event(bigendian.BytesToUint32(b))
}

// BytesToBlock converts bytes to block index.
func BytesToBlock(b []byte) Block {
	return Block(bigendian.BytesToUint64(b))
}

// BytesToLamport converts bytes to block index.
func BytesToLamport(b []byte) Lamport {
	return Lamport(bigendian.BytesToUint32(b))
}

// BytesToFrame converts bytes to block index.
func BytesToFrame(b []byte) Frame {
	return Frame(bigendian.BytesToUint32(b))
}

// BytesToPack converts bytes to block index.
func BytesToPack(b []byte) Pack {
	return Pack(bigendian.BytesToUint32(b))
}

// BytesToValidatorID converts bytes to validator index.
func BytesToValidatorID(b []byte) ValidatorID {
	return ValidatorID(bigendian.BytesToUint32(b))
}

// MaxLamport return max value
func MaxLamport(x, y Lamport) Lamport {
	if x > y {
		return x
	}
	return y
}
