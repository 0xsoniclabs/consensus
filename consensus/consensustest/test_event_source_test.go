package consensustest

import (
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"
)

func TestNewTestEventSource_CreatesEmptyStore(t *testing.T) {
	s := NewTestEventSource()

	if s == nil {
		t.Fatal("NewTestEventSource returned nil")
	}

	if s.db == nil {
		t.Fatal("EventSource db map is nil")
	}

	if len(s.db) != 0 {
		t.Fatal("EventSource db should be empty on creation")
	}
}

func TestSetEvent_StoresEvent(t *testing.T) {
	s := NewTestEventSource()

	event := createTestEvent()
	eventID := event.ID()

	s.SetEvent(event)

	if !s.HasEvent(eventID) {
		t.Fatal("Event not found after being set")
	}

	if s.db[eventID] != event {
		t.Fatal("stored Event doesn't match the original event")
	}
}

func TestGetEvent_RetrievesEvent(t *testing.T) {
	s := NewTestEventSource()

	event := createTestEvent()
	eventID := event.ID()

	s.SetEvent(event)

	retrievedEvent := s.GetEvent(eventID)

	if retrievedEvent == nil {
		t.Fatal("retrieved Event is nil")
	}

	if retrievedEvent != event {
		t.Fatal("retrieved Event doesn't match the original event")
	}
}

func TestGetEvent_ReturnsNilForNonexistentEvent(t *testing.T) {
	s := NewTestEventSource()

	var nonExistentHash consensus.EventHash

	retrievedEvent := s.GetEvent(nonExistentHash)

	if retrievedEvent != nil {
		t.Fatal("GetEvent should return nil for nonexistent event")
	}
}

func TestHasEvent_ReturnsTrueForExistingEvent(t *testing.T) {
	s := NewTestEventSource()

	event := createTestEvent()
	eventID := event.ID()

	s.SetEvent(event)

	if !s.HasEvent(eventID) {
		t.Fatal("HasEvent should return true for existing event")
	}
}

func TestHasEvent_ReturnsFalseForNonexistentEvent(t *testing.T) {
	s := NewTestEventSource()

	var nonExistentHash consensus.EventHash

	if s.HasEvent(nonExistentHash) {
		t.Fatal("HasEvent should return false for nonexistent event")
	}
}

func TestClose_CleansUpResources(t *testing.T) {
	s := NewTestEventSource()

	event := createTestEvent()
	s.SetEvent(event)

	s.Close()

	if s.db != nil {
		t.Fatal("Close should set db to nil")
	}
}

func createTestEvent() consensus.Event {
	mutableEvent := &consensus.MutableBaseEvent{}

	mutableEvent.SetEpoch(1)
	mutableEvent.SetSeq(2)
	mutableEvent.SetFrame(3)
	mutableEvent.SetCreator(4)
	mutableEvent.SetLamport(5)

	parents := make(consensus.EventHashes, 1)
	var parentHash consensus.EventHash
	for i := 0; i < consensus.HashLength && i < 5; i++ {
		parentHash[i] = byte(i + 1)
	}
	parents[0] = parentHash
	mutableEvent.SetParents(parents)

	var randomID [24]byte
	for i := 0; i < 24; i++ {
		randomID[i] = byte(i + 10)
	}

	return mutableEvent.Build(randomID)
}
