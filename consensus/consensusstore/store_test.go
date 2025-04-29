package consensusstore

import (
	"testing"
)

func TestStore_Close(t *testing.T) {
	store := NewMemStore()
	populateWithEpochState(store)
	populateWithLastDecidedState(store)
	populateWithBases(t, store, 10, 10)
	err := store.Close()
	if err != nil {
		t.Fatalf("store.Close() failed: %v", err)
	}
	if store.table.EpochState != nil {
		t.Fatalf("expected EpochState table to be nil")
	}
	if store.table.LastDecidedState != nil {
		t.Fatalf("expected LastDecidedState table to be nil")
	}
	if store.EpochTable.Bases != nil {
		t.Fatalf("expected Bases table to be nil")
	}
}

func TestStore_Drop(t *testing.T) {
	store := NewMemStore()
	basesExpected := populateWithBases(t, store, 10, 10)
	// silence the panic
	store.crit = func(err error) {}
	if err := store.DropEpochDB(); err != nil {
		t.Fatalf("store drop failed unexpectedly")
	}
	for frame := range basesExpected {
		bases := store.GetFrameBases(frame)
		if len(bases) > 0 {
			t.Fatalf("retrieved non-empty frame bases after dropping the epoch DB")
		}
	}
}
