// Copyright (c) 2026 Sonic Operations Ltd
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensusstore

import (
	"testing"
)

func TestStore_Close(t *testing.T) {
	store := NewMemStore()
	populateWithEpochState(store)
	populateWithLastCertifiedState(store)
	populateWithBases(t, store, 10, 10)
	err := store.Close()
	if err != nil {
		t.Fatalf("store.Close() failed: %v", err)
	}
	if store.table.EpochState != nil {
		t.Fatalf("expected EpochState table to be nil")
	}
	if store.table.LastCertifiedState != nil {
		t.Fatalf("expected LastCertifiedState table to be nil")
	}
	if store.epochTable.Bases != nil {
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
