package consensusstore

import (
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"
)

func TestStore_StatesPersisting(t *testing.T) {
	store := NewMemStore()
	lastDecidedState := populateWithLastDecidedState(store)
	if want, got := lastDecidedState, store.GetLastDecidedState(); want.LastDecidedFrame != got.LastDecidedFrame {
		t.Fatalf("incorrect last decided state retrieved. expected: %v, got: %v", want, got)
	}
	// force non-cached retrieval
	store.cache.LastDecidedState = nil
	if want, got := lastDecidedState, store.GetLastDecidedState(); want.LastDecidedFrame != got.LastDecidedFrame {
		t.Fatalf("incorrect last decided state retrieved. expected: %v, got: %v", want, got)
	}
	if want, got := lastDecidedState.LastDecidedFrame, store.GetLastDecidedFrame(); want != got {
		t.Fatalf("incorrect last decided frame retrieved. expected: %d, got: %d", want, got)
	}
}

func populateWithLastDecidedState(store *Store) *LastDecidedState {
	validatorBuilder := consensus.NewValidatorsBuilder()
	validatorBuilder.Set(1, 10)
	lastDecidedState := &LastDecidedState{LastDecidedFrame: 5}
	store.SetLastDecidedState(lastDecidedState)
	return lastDecidedState
}
