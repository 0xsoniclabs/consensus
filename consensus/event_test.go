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
	"reflect"
	"testing"
)

func TestBaseEvent_GettersReturnsCorrectValues(t *testing.T) {
	epoch := Epoch(1)
	seq := Seq(2)
	frame := Frame(3)
	creator := ValidatorID(4)
	parents := EventHashes{EventHash{1, 2, 3}}
	lamport := Lamport(5)
	id := EventHash{6, 7, 8}

	event := BaseEvent{
		epoch:   epoch,
		seq:     seq,
		frame:   frame,
		creator: creator,
		parents: parents,
		lamport: lamport,
		id:      id,
	}

	if event.Epoch() != epoch {
		t.Errorf("Expected epoch %v, got %v", epoch, event.Epoch())
	}

	if event.Seq() != seq {
		t.Errorf("Expected seq %v, got %v", seq, event.Seq())
	}

	if event.Frame() != frame {
		t.Errorf("Expected frame %v, got %v", frame, event.Frame())
	}

	if event.Creator() != creator {
		t.Errorf("Expected creator %v, got %v", creator, event.Creator())
	}

	if !reflect.DeepEqual(event.Parents(), parents) {
		t.Errorf("Expected parents %v, got %v", parents, event.Parents())
	}

	if event.Lamport() != lamport {
		t.Errorf("Expected lamport %v, got %v", lamport, event.Lamport())
	}

	if event.ID() != id {
		t.Errorf("Expected id %v, got %v", id, event.ID())
	}
}

func TestBaseEvent_SizeCalculatesCorrectly(t *testing.T) {
	// Single parent
	event1 := BaseEvent{
		parents: EventHashes{EventHash{1, 2, 3}},
	}
	expectedSize1 := 4 + 4 + 4 + 4 + 1*32 + 4 + 32 // epoch, seq, frame, creator, parents, lamport, id
	if event1.Size() != expectedSize1 {
		t.Errorf("Expected size %v, got %v", expectedSize1, event1.Size())
	}

	// Multiple parents
	event2 := BaseEvent{
		parents: EventHashes{EventHash{1, 2, 3}, EventHash{4, 5, 6}},
	}
	expectedSize2 := 4 + 4 + 4 + 4 + 2*32 + 4 + 32
	if event2.Size() != expectedSize2 {
		t.Errorf("Expected size %v, got %v", expectedSize2, event2.Size())
	}
}

func TestBaseEvent_StringFormatsCorrectly(t *testing.T) {
	event := BaseEvent{
		id:      EventHash{1, 2, 3},
		parents: EventHashes{EventHash{4, 5, 6}},
		creator: ValidatorID(7),
		frame:   Frame(8),
	}

	expected := "{id=16909056:0:000000, p=[67438080:0:000000], by=7, frame=8}"
	if event.String() != expected {
		t.Errorf("Expected string %v, got %v", expected, event.String())
	}
}

func TestBaseEvent_SelfParentWithValidParentReturnsParent(t *testing.T) {
	parentHash := EventHash{1, 2, 3}
	event := BaseEvent{
		seq:     2, // Greater than 1
		parents: EventHashes{parentHash},
	}

	selfParent := event.SelfParent() // return &e.parents[0]
	if selfParent == nil {
		t.Fatal("Expected self-parent to not be nil")
	}
	if *selfParent != parentHash {
		t.Errorf("Expected self-parent %v, got %v", parentHash, *selfParent)
	}
}

func TestBaseEvent_SelfParentWithSeqOneReturnsNil(t *testing.T) {
	event := BaseEvent{
		seq:     1,
		parents: EventHashes{EventHash{1, 2, 3}},
	}

	if event.SelfParent() != nil {
		t.Error("Expected self-parent to be nil when seq is 1")
	}
}

func TestBaseEvent_SelfParentWithNoParentsReturnsNil(t *testing.T) {
	event := BaseEvent{
		seq:     2,
		parents: EventHashes{},
	}

	if event.SelfParent() != nil {
		t.Error("Expected self-parent to be nil when there are no parents")
	}
}

func TestBaseEvent_IsSelfParentWithMatchingParentReturnsTrue(t *testing.T) {
	parentHash := EventHash{1, 2, 3}
	event := BaseEvent{
		seq:     2,
		parents: EventHashes{parentHash},
	}

	if !event.IsSelfParent(parentHash) {
		t.Error("Expected IsSelfParent to return true for matching parent")
	}
}

func TestBaseEvent_IsSelfParentWithNonMatchingParentReturnsFalse(t *testing.T) {
	parentHash := EventHash{1, 2, 3}
	otherHash := EventHash{4, 5, 6}
	event := BaseEvent{
		seq:     2,
		parents: EventHashes{parentHash},
	}

	if event.IsSelfParent(otherHash) {
		t.Error("Expected IsSelfParent to return false for non-matching parent")
	}
}

func TestBaseEvent_IsSelfParentWithNoSelfParentReturnsFalse(t *testing.T) {
	event := BaseEvent{
		seq: 1,
	}

	if event.IsSelfParent(EventHash{1, 2, 3}) {
		t.Error("Expected IsSelfParent to return false when there is no self-parent")
	}
}

func TestMutableBaseEvent_SettersUpdateValues(t *testing.T) {
	event := MutableBaseEvent{}

	epoch := Epoch(1)
	seq := Seq(2)
	frame := Frame(3)
	creator := ValidatorID(4)
	parents := EventHashes{EventHash{1, 2, 3}}
	lamport := Lamport(5)

	event.SetEpoch(epoch)
	if event.epoch != epoch {
		t.Errorf("SetEpoch failed: expected %v, got %v", epoch, event.epoch)
	}

	event.SetSeq(seq)
	if event.seq != seq {
		t.Errorf("SetSeq failed: expected %v, got %v", seq, event.seq)
	}

	event.SetFrame(frame)
	if event.frame != frame {
		t.Errorf("SetFrame failed: expected %v, got %v", frame, event.frame)
	}

	event.SetCreator(creator)
	if event.creator != creator {
		t.Errorf("SetCreator failed: expected %v, got %v", creator, event.creator)
	}

	event.SetParents(parents)
	if !reflect.DeepEqual(event.parents, parents) {
		t.Errorf("SetParents failed: expected %v, got %v", parents, event.parents)
	}

	event.SetLamport(lamport)
	if event.lamport != lamport {
		t.Errorf("SetLamport failed: expected %v, got %v", lamport, event.lamport)
	}
}

func TestMutableBaseEvent_SetIDAssignsCorrectly(t *testing.T) {
	event := MutableBaseEvent{
		BaseEvent: BaseEvent{
			epoch:   Epoch(1),
			lamport: Lamport(2),
		},
	}

	rID := [24]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24}

	event.SetID(rID)

	// Verify epoch bytes were copied to first 4 bytes
	expectedEpochBytes := Epoch(1).Bytes()
	if !bytes.Equal(event.id[0:4], expectedEpochBytes) {
		t.Errorf("Expected epoch bytes %v, got %v", expectedEpochBytes, event.id[0:4])
	}

	// Verify lamport bytes were copied to next 4 bytes
	expectedLamportBytes := Lamport(2).Bytes()
	if !bytes.Equal(event.id[4:8], expectedLamportBytes) {
		t.Errorf("Expected lamport bytes %v, got %v", expectedLamportBytes, event.id[4:8])
	}

	// Verify rest of event ID
	if !bytes.Equal(event.id[8:], rID[:]) {
		t.Errorf("Expected rID bytes %v, got %v", rID[:], event.id[8:])
	}
}

func TestMutableBaseEvent_BuildCreatesImmutableEvent(t *testing.T) {
	mEvent := MutableBaseEvent{
		BaseEvent: BaseEvent{
			epoch:   Epoch(1),
			seq:     Seq(2),
			frame:   Frame(3),
			creator: ValidatorID(4),
			parents: EventHashes{EventHash{1, 2, 3}},
			lamport: Lamport(5),
		},
	}

	rID := [24]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24}

	event := mEvent.Build(rID)

	if event.epoch != mEvent.epoch {
		t.Errorf("Expected epoch %v, got %v", mEvent.epoch, event.epoch)
	}

	if event.seq != mEvent.seq {
		t.Errorf("Expected seq %v, got %v", mEvent.seq, event.seq)
	}

	if event.frame != mEvent.frame {
		t.Errorf("Expected frame %v, got %v", mEvent.frame, event.frame)
	}

	if event.creator != mEvent.creator {
		t.Errorf("Expected creator %v, got %v", mEvent.creator, event.creator)
	}

	if !reflect.DeepEqual(event.parents, mEvent.parents) {
		t.Errorf("Expected parents %v, got %v", mEvent.parents, event.parents)
	}

	if event.lamport != mEvent.lamport {
		t.Errorf("Expected lamport %v, got %v", mEvent.lamport, event.lamport)
	}

	expectedEpochBytes := Epoch(1).Bytes()
	if !bytes.Equal(event.id[0:4], expectedEpochBytes) {
		t.Errorf("Expected epoch bytes %v, got %v", expectedEpochBytes, event.id[0:4])
	}

	expectedLamportBytes := Lamport(5).Bytes()
	if !bytes.Equal(event.id[4:8], expectedLamportBytes) {
		t.Errorf("Expected lamport bytes %v, got %v", expectedLamportBytes, event.id[4:8])
	}

	if !bytes.Equal(event.id[8:], rID[:]) {
		t.Errorf("Expected rID bytes %v, got %v", rID[:], event.id[8:])
	}
}
