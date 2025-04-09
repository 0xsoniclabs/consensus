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

import (
	"bytes"
	"testing"

	"github.com/0xsoniclabs/consensus/utils/byteutils"
)

func TestEpochBytes(t *testing.T) {
	epoch := Epoch(123)
	expected := byteutils.Uint32ToBigEndian(uint32(epoch))
	if !bytes.Equal(epoch.Bytes(), expected) {
		t.Errorf("Epoch.Bytes failed, got %x, want %x", epoch.Bytes(), expected)
	}
}

func TestSeqBytes(t *testing.T) {
	seq := Seq(456)
	expected := byteutils.Uint32ToBigEndian(uint32(seq))
	if !bytes.Equal(seq.Bytes(), expected) {
		t.Errorf("Seq.Bytes failed, got %x, want %x", seq.Bytes(), expected)
	}
}

func TestLamportBytes(t *testing.T) {
	lamport := Lamport(789)
	expected := byteutils.Uint32ToBigEndian(uint32(lamport))
	if !bytes.Equal(lamport.Bytes(), expected) {
		t.Errorf("Lamport.Bytes failed, got %x, want %x", lamport.Bytes(), expected)
	}
}

func TestValidatorIDBytes(t *testing.T) {
	validatorID := ValidatorID(101)
	expected := byteutils.Uint32ToBigEndian(uint32(validatorID))
	if !bytes.Equal(validatorID.Bytes(), expected) {
		t.Errorf("ValidatorID.Bytes failed, got %x, want %x", validatorID.Bytes(), expected)
	}
}

func TestFrameBytes(t *testing.T) {
	frame := Frame(202)
	expected := byteutils.Uint32ToBigEndian(uint32(frame))
	if !bytes.Equal(frame.Bytes(), expected) {
		t.Errorf("Frame.Bytes failed, got %x, want %x", frame.Bytes(), expected)
	}
}

func TestValidatorIndexBytes(t *testing.T) {
	validatorIndex := ValidatorIndex(303)
	expected := byteutils.Uint32ToBigEndian(uint32(validatorIndex))
	if !bytes.Equal(validatorIndex.Bytes(), expected) {
		t.Errorf("ValidatorIndex.Bytes failed, got %x, want %x", validatorIndex.Bytes(), expected)
	}
}

func TestBytesToEpoch(t *testing.T) {
	value := uint32(123)
	bytes := byteutils.Uint32ToBigEndian(value)
	epoch := BytesToEpoch(bytes)
	if uint32(epoch) != value {
		t.Errorf("BytesToEpoch failed, got %d, want %d", epoch, value)
	}
}

func TestBytesToBlock(t *testing.T) {
	value := uint64(1234567890)
	bytes := byteutils.Uint64ToBigEndian(value)
	blockID := BytesToBlock(bytes)
	if uint64(blockID) != value {
		t.Errorf("BytesToBlock failed, got %d, want %d", blockID, value)
	}
}

func TestBytesToLamport(t *testing.T) {
	value := uint32(456)
	bytes := byteutils.Uint32ToBigEndian(value)
	lamport := BytesToLamport(bytes)
	if uint32(lamport) != value {
		t.Errorf("BytesToLamport failed, got %d, want %d", lamport, value)
	}
}

func TestBytesToFrame(t *testing.T) {
	value := uint32(789)
	bytes := byteutils.Uint32ToBigEndian(value)
	frame := BytesToFrame(bytes)
	if uint32(frame) != value {
		t.Errorf("BytesToFrame failed, got %d, want %d", frame, value)
	}
}

func TestBytesToValidatorID(t *testing.T) {
	value := uint32(101)
	bytes := byteutils.Uint32ToBigEndian(value)
	validatorID := BytesToValidatorID(bytes)
	if uint32(validatorID) != value {
		t.Errorf("BytesToValidatorID failed, got %d, want %d", validatorID, value)
	}
}

func TestBytesToValidator(t *testing.T) {
	value := uint32(202)
	bytes := byteutils.Uint32ToBigEndian(value)
	validatorIndex := BytesToValidator(bytes)
	if uint32(validatorIndex) != value {
		t.Errorf("BytesToValidator failed, got %d, want %d", validatorIndex, value)
	}
}

func TestMaxLamport(t *testing.T) {
	tests := []struct {
		x        Lamport
		y        Lamport
		expected Lamport
	}{
		{Lamport(10), Lamport(20), Lamport(20)},
		{Lamport(30), Lamport(20), Lamport(30)},
		{Lamport(0), Lamport(0), Lamport(0)},
	}

	for _, test := range tests {
		result := MaxLamport(test.x, test.y)
		if result != test.expected {
			t.Errorf("MaxLamport(%d, %d) = %d, want %d", test.x, test.y, result, test.expected)
		}
	}
}

func TestFirstFrameConstant(t *testing.T) {
	if FirstFrame != Frame(1) {
		t.Errorf("FirstFrame constant incorrect, got %d, want %d", FirstFrame, Frame(1))
	}
}

func TestFirstEpochConstant(t *testing.T) {
	if FirstEpoch != Epoch(1) {
		t.Errorf("FirstEpoch constant incorrect, got %d, want %d", FirstEpoch, Epoch(1))
	}
}
