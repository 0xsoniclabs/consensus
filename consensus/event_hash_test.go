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
	"crypto/sha256"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestEventHashBytes(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	eh := BytesToEvent(data)
	if !bytes.Equal(eh.Bytes(), (Hash)(eh).Bytes()) {
		t.Errorf("EventHash.Bytes failed")
	}
}

func TestEventHashBig(t *testing.T) {
	eh := HexToEventHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")
	expected := (*Hash)(&eh).Big()
	if eh.Big().Cmp(expected) != 0 {
		t.Errorf("EventHash.Big failed")
	}
}

func TestEventHashSetBytes(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	var eh EventHash
	eh.SetBytes(data)
	var h Hash
	h.SetBytes(data)
	if !bytes.Equal(eh.Bytes(), h.Bytes()) {
		t.Errorf("EventHash.SetBytes failed")
	}
}

func TestBytesToEvent(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	eh := BytesToEvent(data)
	h := FromBytes(data)
	if !bytes.Equal(eh.Bytes(), h.Bytes()) {
		t.Errorf("BytesToEvent failed")
	}
}

func TestFromBytes(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	h := FromBytes(data)
	var expected Hash
	expected.SetBytes(data)
	if !bytes.Equal(h.Bytes(), expected.Bytes()) {
		t.Errorf("FromBytes failed")
	}
}

func TestHexToEventHash(t *testing.T) {
	hexStr := "0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102"
	eh := HexToEventHash(hexStr)
	h := HexToHash(hexStr)
	if !bytes.Equal(eh.Bytes(), h.Bytes()) {
		t.Errorf("HexToEventHash failed")
	}
}

func TestEventHashHex(t *testing.T) {
	eh := HexToEventHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")
	if eh.Hex() != Hash(eh).Hex() {
		t.Errorf("EventHash.Hex failed")
	}
}

func TestEventHashLamport(t *testing.T) {
	lamportBytes := []byte{5, 6, 7, 8}
	var eh EventHash
	copy(eh[4:8], lamportBytes)
	lamport := eh.Lamport()
	expected := BytesToLamport(lamportBytes)
	if lamport != expected {
		t.Errorf("EventHash.Lamport failed")
	}
}

func TestEventHashEpoch(t *testing.T) {
	epochBytes := []byte{1, 2, 3, 4}
	var eh EventHash
	copy(eh[0:4], epochBytes)
	epoch := eh.Epoch()
	expected := BytesToEpoch(epochBytes)
	if epoch != expected {
		t.Errorf("EventHash.Epoch failed")
	}
}

func TestEventHashString(t *testing.T) {
	eh := HexToEventHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")
	if eh.String() != eh.ShortID(3) {
		t.Errorf("EventHash.String failed")
	}
}

func TestEventHashFullID(t *testing.T) {
	eh := HexToEventHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")
	if eh.FullID() != eh.ShortID(32-4-4) {
		t.Errorf("EventHash.FullID failed")
	}
}

func TestEventHashShortID(t *testing.T) {
	epochBytes := []byte{0, 0, 0, 123}
	lamportBytes := []byte{0, 0, 0, 45}
	var eh EventHash
	copy(eh[0:4], epochBytes)
	copy(eh[4:8], lamportBytes)

	for i := 8; i < len(eh); i++ {
		eh[i] = byte(i)
	}

	precision := 3
	expected := "123:45:" + common.Bytes2Hex(eh[8:8+precision])
	if eh.ShortID(precision) != expected {
		t.Errorf("EventHash.ShortID failed, got %s, want %s", eh.ShortID(precision), expected)
	}
}

func TestEventHashIsZero(t *testing.T) {
	var eh EventHash
	if !eh.IsZero() {
		t.Errorf("EventHash.IsZero failed for zero hash")
	}

	eh = HexToEventHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")
	if eh.IsZero() {
		t.Errorf("EventHash.IsZero failed for non-zero hash")
	}
}

func TestEventHashSetCopy(t *testing.T) {
	set := make(EventHashSet)
	eh1 := HexToEventHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")
	eh2 := HexToEventHash("0x0202030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0103")
	set.Add(eh1, eh2)

	copied := set.Copy()
	if len(copied) != len(set) {
		t.Errorf("EventHashSet.Copy failed, lengths differ")
	}

	if !copied.Contains(eh1) || !copied.Contains(eh2) {
		t.Errorf("EventHashSet.Copy failed, content differs")
	}
}

func TestEventHashSetString(t *testing.T) {
	set := make(EventHashSet)
	eh := HexToEventHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")
	set.Add(eh)

	str := set.String()
	expected := "[" + eh.String() + "]"
	if str != expected {
		t.Errorf("EventHashSet.String failed, got %s, want %s", str, expected)
	}
}

func TestEventHashSetSlice(t *testing.T) {
	set := make(EventHashSet)
	eh := HexToEventHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")
	set.Add(eh)

	slice := set.Slice()
	if len(slice) != 1 || slice[0] != eh {
		t.Errorf("EventHashSet.Slice failed")
	}
}

func TestEventHashSetAdd(t *testing.T) {
	set := make(EventHashSet)
	eh1 := HexToEventHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")
	eh2 := HexToEventHash("0x0202030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0103")

	set.Add(eh1, eh2)
	if !set.Contains(eh1) || !set.Contains(eh2) {
		t.Errorf("EventHashSet.Add failed")
	}
}

func TestEventHashSetErase(t *testing.T) {
	set := make(EventHashSet)
	eh1 := HexToEventHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")
	eh2 := HexToEventHash("0x0202030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0103")

	set.Add(eh1, eh2)
	set.Erase(eh1)

	if set.Contains(eh1) || !set.Contains(eh2) {
		t.Errorf("EventHashSet.Erase failed")
	}
}

func TestEventHashSetContains(t *testing.T) {
	set := make(EventHashSet)
	eh := HexToEventHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")

	if set.Contains(eh) {
		t.Errorf("EventHashSet.Contains wrongly returned true for non-existent hash")
	}

	set.Add(eh)
	if !set.Contains(eh) {
		t.Errorf("EventHashSet.Contains wrongly returned false for existent hash")
	}
}

func TestEventHashesCopy(t *testing.T) {
	hashes := EventHashes{
		HexToEventHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102"),
		HexToEventHash("0x0202030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0103"),
	}

	copied := hashes.Copy()
	if len(copied) != len(hashes) {
		t.Errorf("EventHashes.Copy failed, lengths differ")
	}

	for i := range hashes {
		if copied[i] != hashes[i] {
			t.Errorf("EventHashes.Copy failed, content differs at index %d", i)
		}
	}
}

func TestEventHashesString(t *testing.T) {
	hashes := EventHashes{
		HexToEventHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102"),
	}

	str := hashes.String()
	expected := "[" + hashes[0].String() + "]"
	if str != expected {
		t.Errorf("EventHashes.String failed, got %s, want %s", str, expected)
	}
}

func TestEventHashesSet(t *testing.T) {
	hashes := EventHashes{
		HexToEventHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102"),
		HexToEventHash("0x0202030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0103"),
	}

	set := hashes.Set()
	if len(set) != len(hashes) {
		t.Errorf("EventHashes.Set failed, lengths differ")
	}

	for _, h := range hashes {
		if !set.Contains(h) {
			t.Errorf("EventHashes.Set failed, missing hash %s", h.String())
		}
	}
}

func TestEventHashesAdd(t *testing.T) {
	var hashes EventHashes
	eh1 := HexToEventHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")
	eh2 := HexToEventHash("0x0202030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0103")

	hashes.Add(eh1, eh2)
	if len(hashes) != 2 || hashes[0] != eh1 || hashes[1] != eh2 {
		t.Errorf("EventHashes.Add failed")
	}
}

func TestEventHashStackPush(t *testing.T) {
	var stack EventHashStack
	eh := HexToEventHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")

	stack.Push(eh)
	if len(stack) != 1 || stack[0] != eh {
		t.Errorf("EventHashStack.Push failed")
	}
}

func TestEventHashStackPushAll(t *testing.T) {
	var stack EventHashStack
	hashes := EventHashes{
		HexToEventHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102"),
		HexToEventHash("0x0202030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0103"),
	}

	stack.PushAll(hashes)
	if len(stack) != len(hashes) {
		t.Errorf("EventHashStack.PushAll failed, lengths differ")
	}

	for i := range hashes {
		if stack[i] != hashes[i] {
			t.Errorf("EventHashStack.PushAll failed, content differs at index %d", i)
		}
	}
}

func TestEventHashStackPop(t *testing.T) {
	var stack EventHashStack
	eh1 := HexToEventHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")
	eh2 := HexToEventHash("0x0202030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0103")

	stack.Push(eh1)
	stack.Push(eh2)

	popped := stack.Pop()
	if *popped != eh2 {
		t.Errorf("EventHashStack.Pop failed, returned wrong element")
	}

	if len(stack) != 1 || stack[0] != eh1 {
		t.Errorf("EventHashStack.Pop failed, stack modified incorrectly")
	}

	stack.Pop()
	if stack.Pop() != nil {
		t.Errorf("EventHashStack.Pop failed, should return nil for empty stack")
	}
}

func TestEventHashFromBytes(t *testing.T) {
	data1 := []byte{1, 2, 3, 4}
	data2 := []byte{5, 6, 7, 8}

	hash := EventHashFromBytes(data1, data2)

	d := sha256.New()
	d.Write(data1)
	d.Write(data2)
	expected := make([]byte, 32)
	d.Sum(expected[:0])

	if !bytes.Equal(hash[:], expected) {
		t.Errorf("EventHashFromBytes failed")
	}
}

func TestZeroEventHash(t *testing.T) {
	var expected EventHash
	if ZeroEventHash != expected {
		t.Errorf("ZeroEventHash not initialized correctly")
	}
}
