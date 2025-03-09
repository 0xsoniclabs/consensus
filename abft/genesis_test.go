package abft

import (
	"testing"

	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/inter/pos"
)

func TestGenesis_Sucess(t *testing.T) {
	store := NewMemStore()
	validatorBuilder := pos.NewBuilder()
	validatorBuilder.Set(1, 10)
	validators := validatorBuilder.Build()
	epoch := idx.Epoch(3)
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
	lastDecidedState, exists := store.get(store.table.LastDecidedState, []byte(dsKey), &LastDecidedState{}).(*LastDecidedState)
	if !exists {
		t.Fatal("last decided state not set")
	}
	if want, got := lastDecidedState.LastDecidedFrame, FirstFrame-1; want != got {
		t.Fatalf("expected frame for last state: %d, got: %d", want, got)
	}
}
func TestGenesis_Fail(t *testing.T) {
	store := NewMemStore()
	if err := store.ApplyGenesis(nil); err == nil {
		t.Fatal("error expected but not received")
	}
	if err := store.ApplyGenesis(&Genesis{Epoch: 1, Validators: &pos.Validators{}}); err == nil {
		t.Fatal("error expected but not received")
	}
	validatorBuilder := pos.NewBuilder()
	validatorBuilder.Set(1, 10)
	store.table.LastDecidedState.Put([]byte(dsKey), []byte{})
	if err := store.ApplyGenesis(&Genesis{Epoch: 1, Validators: validatorBuilder.Build()}); err == nil {
		t.Fatal("error expected but not received")
	}
}
