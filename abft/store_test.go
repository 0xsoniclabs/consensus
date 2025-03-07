package abft

import (
	"testing"

	"github.com/0xsoniclabs/consensus/inter/pos"
)

func TestStore_StateSetting(t *testing.T) {
	store, epochState, lastDecidedState := newPopulatedStore()
	if want, got := epochState, store.GetEpochState(); want.Epoch != got.Epoch || want.Validators.TotalWeight() != got.Validators.TotalWeight() {
		t.Fatalf("incorrect epoch state retrieved. expected: %v, got: %v", want, got)
	}
	if want, got := lastDecidedState, store.GetLastDecidedState(); want.LastDecidedFrame != got.LastDecidedFrame {
		t.Fatalf("incorrect last decided state retrieved. expected: %v, got: %v", want, got)
	}
}

func TestStore_Close(t *testing.T) {
	store, _, _ := newPopulatedStore()
	store.Close()
	if store.table.EpochState != nil {
		t.Fatalf("expected EpochState table to be nil")
	}
	if store.table.LastDecidedState != nil {
		t.Fatalf("expected LastDecidedState table to be nil")
	}
}

func newPopulatedStore() (*Store, *EpochState, *LastDecidedState) {
	store := NewMemStore()
	validatorBuilder := pos.NewBuilder()
	validatorBuilder.Set(1, 10)
	epochState := &EpochState{Epoch: 3, Validators: validatorBuilder.Build()}
	lastDecidedState := &LastDecidedState{LastDecidedFrame: 5}
	store.SetEpochState(epochState)
	store.SetLastDecidedState(lastDecidedState)

	return store, epochState, lastDecidedState
}
