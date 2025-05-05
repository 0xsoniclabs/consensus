package consensusstore

import (
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"
)

func TestGenesis_Sucess(t *testing.T) {
	store := NewMemStore()
	validatorBuilder := consensus.NewValidatorsBuilder()
	validatorBuilder.Set(1, 10)
	validators := validatorBuilder.Build()
	epoch := consensus.Epoch(3)
	if err := store.ApplyGenesis(&Genesis{Epoch: epoch, Validators: validators}); err != nil {
		t.Fatal(err)
	}
	epochState, exists := store.get(store.table.EpochState, []byte(esKey), &EpochState{}).(*EpochState)
	if !exists {
		t.Fatal("epoch state not set")
	}
	if want, got := epochState.Epoch, epoch; want != got {
		t.Fatalf("expected set epoch: %d, got: %d", want, got)
	}
	if want, got := epochState.Validators.Get(1), validators.Get(1); want != got {
		t.Fatalf("expected set validator weight: %d, got: %d", want, got)
	}
	lastCertifiedState, exists := store.get(store.table.LastCertifiedState, []byte(dsKey), &LastCertifiedState{}).(*LastCertifiedState)
	if !exists {
		t.Fatal("last certified state not set")
	}
	if want, got := lastCertifiedState.LastCertifiedFrame, consensus.FirstFrame-1; want != got {
		t.Fatalf("expected frame for last state: %d, got: %d", want, got)
	}
}
func TestGenesis_Fail(t *testing.T) {
	store := NewMemStore()
	if err := store.ApplyGenesis(nil); err == nil {
		t.Fatal("error expected but not received")
	}
	if err := store.ApplyGenesis(&Genesis{Epoch: 1, Validators: &consensus.Validators{}}); err == nil {
		t.Fatal("error expected but not received")
	}
	validatorBuilder := consensus.NewValidatorsBuilder()
	validatorBuilder.Set(1, 10)
	err := store.table.LastCertifiedState.Put([]byte(dsKey), []byte{})
	if err != nil {
		t.Fatalf("failed to set up prerequisite state (Put LastCertifiedState): %v", err)
	}
	if err := store.ApplyGenesis(&Genesis{Epoch: 1, Validators: validatorBuilder.Build()}); err == nil {
		t.Fatal("error expected but not received")
	}
}
