// Copyright (c) 2026 Sonic Operations Ltd
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package dagvec

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/0xsoniclabs/consensus/consensus"
)

// mockEvent implements consensus.Event for testing vector operations.
type mockEvent struct {
	consensus.Event
	seq     consensus.Seq
	parents consensus.EventHashes
}

func (e *mockEvent) Seq() consensus.Seq             { return e.seq }
func (e *mockEvent) Parents() consensus.EventHashes { return e.parents }

// mockCreationTimerEvent also implements creationTimer.
type mockCreationTimerEvent struct {
	*mockEvent
	creationTime Timestamp
}

func (e *mockCreationTimerEvent) CreationTimePortable() Timestamp {
	return e.creationTime
}

func TestHighestBefore_InitWithEvent(t *testing.T) {
	hb := NewHighestBefore(3)
	e := &mockEvent{seq: 5}
	hb.InitWithEvent(1, e)

	got := hb.VSeq.Get(1)
	assert.Equal(t, consensus.Seq(5), got.Seq)
	assert.Equal(t, consensus.Seq(5), got.MinSeq)
	// No creationTimer => VTime stays 0
	assert.Equal(t, Timestamp(0), hb.VTime.Get(1))
}

func TestHighestBefore_InitWithEventCreationTimer(t *testing.T) {
	hb := NewHighestBefore(3)
	e := &mockCreationTimerEvent{
		mockEvent:    &mockEvent{seq: 5},
		creationTime: 12345,
	}
	hb.InitWithEvent(1, e)

	assert.Equal(t, Timestamp(12345), hb.VTime.Get(1))
}

func TestLowestAfter_InitWithEvent(t *testing.T) {
	la := NewLowestAfterSeq(3)
	e := &mockEvent{seq: 7}
	la.InitWithEvent(2, e)

	assert.Equal(t, consensus.Seq(7), la.Get(2))
}

func TestLowestAfter_Visit(t *testing.T) {
	la := NewLowestAfterSeq(3)
	e := &mockEvent{seq: 7}

	// First visit should succeed
	ok := la.Visit(2, e)
	assert.True(t, ok)
	assert.Equal(t, consensus.Seq(7), la.Get(2))

	// Second visit should be no-op
	e2 := &mockEvent{seq: 10}
	ok = la.Visit(2, e2)
	assert.False(t, ok)
	assert.Equal(t, consensus.Seq(7), la.Get(2))
}

func TestHighestBefore_IsEmpty(t *testing.T) {
	hb := NewHighestBefore(3)
	assert.True(t, hb.IsEmpty(0))

	hb.VSeq.Set(0, BranchSeq{Seq: 1, MinSeq: 1})
	assert.False(t, hb.IsEmpty(0))

	hb.SetEquivocationDetected(1)
	assert.False(t, hb.IsEmpty(1))
}

func TestHighestBefore_IsEquivocationDetected(t *testing.T) {
	hb := NewHighestBefore(3)
	assert.False(t, hb.IsEquivocationDetected(0))

	hb.SetEquivocationDetected(0)
	assert.True(t, hb.IsEquivocationDetected(0))
}

func TestHighestBefore_SeqMinSeq(t *testing.T) {
	hb := NewHighestBefore(3)
	hb.VSeq.Set(0, BranchSeq{Seq: 10, MinSeq: 3})
	assert.Equal(t, consensus.Seq(10), hb.Seq(0))
	assert.Equal(t, consensus.Seq(3), hb.MinSeq(0))
}

func TestHighestBefore_CollectFrom(t *testing.T) {
	hb1 := NewHighestBefore(3)
	hb2 := NewHighestBefore(3)

	// hb2 has some values
	hb2.VSeq.Set(0, BranchSeq{Seq: 5, MinSeq: 3})
	hb2.VTime.Set(0, 100)
	hb2.VSeq.Set(1, BranchSeq{Seq: 10, MinSeq: 7})
	hb2.VTime.Set(1, 200)

	hb1.CollectFrom(hb2, 3)
	assert.Equal(t, consensus.Seq(5), hb1.Seq(0))
	assert.Equal(t, consensus.Seq(3), hb1.MinSeq(0))
	assert.Equal(t, Timestamp(100), hb1.VTime.Get(0))
	assert.Equal(t, consensus.Seq(10), hb1.Seq(1))
}

func TestHighestBefore_CollectFrom_EquivocationDetected(t *testing.T) {
	hb1 := NewHighestBefore(3)
	hb2 := NewHighestBefore(3)

	hb2.SetEquivocationDetected(0)
	hb1.CollectFrom(hb2, 3)
	assert.True(t, hb1.IsEquivocationDetected(0))
}

func TestHighestBefore_CollectFrom_SkipsMaxed(t *testing.T) {
	hb1 := NewHighestBefore(3)
	hb2 := NewHighestBefore(3)

	// hb1 already at equivocation detected
	hb1.SetEquivocationDetected(0)
	hb2.VSeq.Set(0, BranchSeq{Seq: 5, MinSeq: 3})
	hb1.CollectFrom(hb2, 3)
	// Should still be equivocation detected
	assert.True(t, hb1.IsEquivocationDetected(0))
}

func TestHighestBefore_CollectFrom_TakesMinSeq(t *testing.T) {
	hb1 := NewHighestBefore(3)
	hb2 := NewHighestBefore(3)

	hb1.VSeq.Set(0, BranchSeq{Seq: 10, MinSeq: 5})
	hb2.VSeq.Set(0, BranchSeq{Seq: 8, MinSeq: 2})
	hb2.VTime.Set(0, 200)

	hb1.CollectFrom(hb2, 3)
	// MinSeq should be the lower one
	assert.Equal(t, consensus.Seq(2), hb1.MinSeq(0))
	// Seq should remain 10 (higher)
	assert.Equal(t, consensus.Seq(10), hb1.Seq(0))
}

func TestHighestBefore_CollectFrom_TakesHigherSeq(t *testing.T) {
	hb1 := NewHighestBefore(3)
	hb2 := NewHighestBefore(3)

	hb1.VSeq.Set(0, BranchSeq{Seq: 5, MinSeq: 3})
	hb1.VTime.Set(0, 100)
	hb2.VSeq.Set(0, BranchSeq{Seq: 10, MinSeq: 7})
	hb2.VTime.Set(0, 200)

	hb1.CollectFrom(hb2, 3)
	assert.Equal(t, consensus.Seq(10), hb1.Seq(0))
	assert.Equal(t, Timestamp(200), hb1.VTime.Get(0))
}

func TestHighestBefore_GatherFrom(t *testing.T) {
	hb := NewHighestBefore(3)
	other := NewHighestBefore(5)

	other.VSeq.Set(3, BranchSeq{Seq: 5, MinSeq: 3})
	other.VTime.Set(3, 100)
	other.VSeq.Set(4, BranchSeq{Seq: 10, MinSeq: 7})
	other.VTime.Set(4, 200)

	hb.GatherFrom(0, other, []consensus.ValidatorIndex{3, 4})
	assert.Equal(t, consensus.Seq(10), hb.Seq(0))
	assert.Equal(t, Timestamp(200), hb.VTime.Get(0))
}

func TestHighestBefore_GatherFrom_EquivocationDetected(t *testing.T) {
	hb := NewHighestBefore(3)
	other := NewHighestBefore(5)

	other.VSeq.Set(3, BranchSeq{Seq: 5, MinSeq: 3})
	other.SetEquivocationDetected(4)

	hb.GatherFrom(0, other, []consensus.ValidatorIndex{3, 4})
	assert.True(t, hb.IsEquivocationDetected(0))
}
