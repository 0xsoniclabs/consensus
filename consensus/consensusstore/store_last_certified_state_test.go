package consensusstore

import (
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"
)

func TestStore_StatesPersisting(t *testing.T) {
	store := NewMemStore()
	lastCertifiedState := populateWithLastCertifiedState(store)
	if want, got := lastCertifiedState, store.GetLastCertifiedState(); want.LastCertifiedFrame != got.LastCertifiedFrame {
		t.Fatalf("incorrect last certified state retrieved. expected: %v, got: %v", want, got)
	}
	// force non-cached retrieval
	store.cache.LastCertifiedState = nil
	if want, got := lastCertifiedState, store.GetLastCertifiedState(); want.LastCertifiedFrame != got.LastCertifiedFrame {
		t.Fatalf("incorrect last certified state retrieved. expected: %v, got: %v", want, got)
	}
	if want, got := lastCertifiedState.LastCertifiedFrame, store.GetLastCertifiedFrame(); want != got {
		t.Fatalf("incorrect last certified frame retrieved. expected: %d, got: %d", want, got)
	}
}

func populateWithLastCertifiedState(store *Store) *LastCertifiedState {
	validatorBuilder := consensus.NewValidatorsBuilder()
	validatorBuilder.Set(1, 10)
	lastCertifiedState := &LastCertifiedState{LastCertifiedFrame: 5}
	store.SetLastCertifiedState(lastCertifiedState)
	return lastCertifiedState
}
