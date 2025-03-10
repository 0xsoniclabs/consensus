package abft

import (
	"math/rand"
	"slices"
	"testing"

	"github.com/0xsoniclabs/consensus/abft/election"
	"github.com/0xsoniclabs/consensus/inter/dag/tdag"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/inter/pos"
)

func TestStore_StatePersisting(t *testing.T) {
	store := NewMemStore()
	epochState, lastDecidedState := populateWithEpochStates(store)
	if want, got := epochState, store.GetEpochState(); want.Epoch != got.Epoch || want.Validators.TotalWeight() != got.Validators.TotalWeight() {
		t.Fatalf("incorrect epoch state retrieved. expected: %v, got: %v", want, got)
	}
	if want, got := lastDecidedState, store.GetLastDecidedState(); want.LastDecidedFrame != got.LastDecidedFrame {
		t.Fatalf("incorrect last decided state retrieved. expected: %v, got: %v", want, got)
	}
}

func TestStore_FrameRootPersisting(t *testing.T) {
	store := NewMemStore()
	roots := populateWithRoots(store)
	retrievalOrder := make([]int, len(roots))
	for i := range len(roots) {
		retrievalOrder[i] = i
	}
	rand.Shuffle(len(retrievalOrder), func(i, j int) { retrievalOrder[i], retrievalOrder[j] = retrievalOrder[j], retrievalOrder[i] })
	for _, frame := range retrievalOrder {
		frameRoots := store.GetFrameRoots(idx.Frame(frame))
		if want, got := len(roots[frame]), len(frameRoots); want != got {
			t.Fatalf("incorrect number of roots retrieved for frame %d, expected: %d, got: %d", frame, want, got)
		}
		slices.SortFunc(frameRoots, func(r1, r2 election.RootContext) int { return int(r1.ValidatorID) - int(r2.ValidatorID) })
		for validatorID := range len(frameRoots) {
			if want, got := roots[frame][validatorID].ID(), frameRoots[validatorID].RootHash; want != got {
				t.Fatalf("incorrect root retrieved for [frame, validator]: [%d, %d], expected: %s, got: %s", frame, validatorID, want, got)
			}
		}
	}
}

func TestStore_Close(t *testing.T) {
	store := NewMemStore()
	populateWithEpochStates(store)
	populateWithRoots(store)
	store.Close()
	if store.table.EpochState != nil {
		t.Fatalf("expected EpochState table to be nil")
	}
	if store.table.LastDecidedState != nil {
		t.Fatalf("expected LastDecidedState table to be nil")
	}
	if store.epochTable.Roots != nil {
		t.Fatalf("expected Roots table to be nil")
	}
}

func populateWithRoots(store *Store) [][]*tdag.TestEvent {
	store.openEpochDB(1)
	roots := make([][]*tdag.TestEvent, 100)
	for frame := range idx.Frame(100) {
		roots[frame] = make([]*tdag.TestEvent, 100)
		for validatorID := range idx.ValidatorID(100) {
			root := &tdag.TestEvent{}
			root.SetID([24]byte{byte(frame), byte(validatorID)})
			store.addRoot(root, frame)
			roots[frame][validatorID] = root
		}
	}
	return roots
}

func populateWithEpochStates(store *Store) (*EpochState, *LastDecidedState) {
	validatorBuilder := pos.NewBuilder()
	validatorBuilder.Set(1, 10)
	epochState := &EpochState{Epoch: 3, Validators: validatorBuilder.Build()}
	lastDecidedState := &LastDecidedState{LastDecidedFrame: 5}
	store.SetEpochState(epochState)
	store.SetLastDecidedState(lastDecidedState)
	return epochState, lastDecidedState
}
