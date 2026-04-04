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
