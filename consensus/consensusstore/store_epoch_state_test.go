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
	"fmt"
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"
)

func TestStore_ConsistentEpochStatePersistingAndRetrieving(t *testing.T) {
	store := NewMemStore()
	epochState := populateWithEpochState(store)
	if want, got := epochState, store.GetEpochState(); want.Epoch != got.Epoch || want.Validators.TotalWeight() != got.Validators.TotalWeight() {
		t.Fatalf("incorrect epoch state retrieved. expected: %v, got: %v", want, got)
	}
	// force non-cached retrieval
	store.cache.EpochState = nil
	if want, got := epochState, store.GetEpochState(); want.Epoch != got.Epoch || want.Validators.TotalWeight() != got.Validators.TotalWeight() {
		t.Fatalf("incorrect epoch state retrieved. expected: %v, got: %v", want, got)
	}
	if want, got := epochState.Epoch, store.GetEpoch(); want != got {
		t.Fatalf("incorrect epoch retrieved. expected: %d, got: %d", want, got)
	}
	if want, got := epochState.Validators, store.GetValidators(); want.TotalWeight() != got.TotalWeight() {
		t.Fatalf("incorrect validators retrieved. expected: %v, got: %v", want, got)
	}
}

func TestStore_ConsistentEpochStateFormatting(t *testing.T) {
	store := NewMemStore()
	epochState := populateWithEpochState(store)
	if want, got := fmt.Sprintf("%d/%s", epochState.Epoch, epochState.Validators.String()), epochState.String(); want != got {
		t.Fatalf("unexpectedly formatted epochState, expected: %s, got: %s", want, got)
	}
}

func populateWithEpochState(store *Store) *EpochState {
	validatorBuilder := consensus.NewValidatorsBuilder()
	validatorBuilder.Set(1, 10)
	epochState := &EpochState{Epoch: 3, Validators: validatorBuilder.Build()}
	store.SetEpochState(epochState)
	return epochState
}
