package consensusstore

import (
	"encoding/binary"
	"math/rand"
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"
)

func TestEventConfirmedOn_NonExistingEvent(t *testing.T) {
	store := NewMemStore()
	if err := store.OpenEpochDB(1); err != nil {
		t.Fatal(err)
	}

	if want, got := consensus.Frame(0), store.GetEventConfirmedOn(consensus.EventHash{}); want != got {
		t.Fatalf("unexpected frame retrieved for non-existing event hash, expected: %d, got: %d", want, got)
	}
}

func TestEventConfirmedOn_ConsistentPersistingAndRetrieval(t *testing.T) {
	store := NewMemStore()
	if err := store.OpenEpochDB(1); err != nil {
		t.Fatal(err)
	}

	expectedFrames := make(map[consensus.EventHash]consensus.Frame)
	numFrames, meanEventPerFrame := 100, 1000
	for i := range meanEventPerFrame * numFrames {
		frame := consensus.Frame(rand.Intn(numFrames))
		eventHash := consensus.EventHash{}
		binary.LittleEndian.PutUint16(eventHash[:16], uint16(frame))
		binary.LittleEndian.PutUint16(eventHash[16:], uint16(i))
		store.SetEventConfirmedOn(eventHash, frame)
		expectedFrames[eventHash] = frame
	}

	for eventHash, want := range expectedFrames {
		if got := store.GetEventConfirmedOn(eventHash); want != got {
			t.Fatalf("unexpected frame retrieved for event: %s, expected: %d, got: %d", eventHash.String(), want, got)
		}
	}
}
