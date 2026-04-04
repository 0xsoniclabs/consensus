package dagvec

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/0xsoniclabs/consensus/consensus"
)

func TestLowestAfterSeq_NewAndSize(t *testing.T) {
	v := NewLowestAfterSeq(5)
	assert.Equal(t, consensus.ValidatorIndex(5), v.Size())
}

func TestLowestAfterSeq_GetSet(t *testing.T) {
	v := NewLowestAfterSeq(3)
	v.Set(0, 10)
	v.Set(2, 20)
	assert.Equal(t, consensus.Seq(10), v.Get(0))
	assert.Equal(t, consensus.Seq(0), v.Get(1))
	assert.Equal(t, consensus.Seq(20), v.Get(2))
}

func TestLowestAfterSeq_GetOutOfRange(t *testing.T) {
	v := NewLowestAfterSeq(2)
	// Get beyond size returns 0
	assert.Equal(t, consensus.Seq(0), v.Get(5))
}

func TestLowestAfterSeq_SetExpands(t *testing.T) {
	v := NewLowestAfterSeq(2)
	// Set beyond current size expands the vector
	v.Set(5, 42)
	assert.Equal(t, consensus.Seq(42), v.Get(5))
}

func TestHighestBeforeSeq_GetSet(t *testing.T) {
	hb := NewHighestBefore(3)
	hb.VSeq.Set(0, BranchSeq{Seq: 10, MinSeq: 5})
	hb.VSeq.Set(2, BranchSeq{Seq: 20, MinSeq: 15})

	got := hb.VSeq.Get(0)
	assert.Equal(t, consensus.Seq(10), got.Seq)
	assert.Equal(t, consensus.Seq(5), got.MinSeq)

	got2 := hb.VSeq.Get(2)
	assert.Equal(t, consensus.Seq(20), got2.Seq)
	assert.Equal(t, consensus.Seq(15), got2.MinSeq)

	// Unset position
	got1 := hb.VSeq.Get(1)
	assert.Equal(t, consensus.Seq(0), got1.Seq)
	assert.Equal(t, consensus.Seq(0), got1.MinSeq)
}

func TestHighestBeforeSeq_Size(t *testing.T) {
	hb := NewHighestBefore(4)
	assert.Equal(t, 4, hb.VSeq.Size())
}

func TestHighestBeforeSeq_GetOutOfRange(t *testing.T) {
	hb := NewHighestBefore(2)
	got := hb.VSeq.Get(10)
	assert.Equal(t, BranchSeq{}, got)
}

func TestHighestBeforeSeq_SetExpands(t *testing.T) {
	hb := NewHighestBefore(2)
	hb.VSeq.Set(5, BranchSeq{Seq: 99, MinSeq: 1})
	got := hb.VSeq.Get(5)
	assert.Equal(t, consensus.Seq(99), got.Seq)
}

func TestHighestBeforeTime_GetSet(t *testing.T) {
	hb := NewHighestBefore(3)
	hb.VTime.Set(0, 1000)
	hb.VTime.Set(2, 2000)

	assert.Equal(t, Timestamp(1000), hb.VTime.Get(0))
	assert.Equal(t, Timestamp(0), hb.VTime.Get(1))
	assert.Equal(t, Timestamp(2000), hb.VTime.Get(2))
}

func TestHighestBeforeTime_GetOutOfRange(t *testing.T) {
	hb := NewHighestBefore(2)
	assert.Equal(t, Timestamp(0), hb.VTime.Get(10))
}

func TestHighestBeforeTime_SetExpands(t *testing.T) {
	hb := NewHighestBefore(2)
	hb.VTime.Set(5, 42)
	assert.Equal(t, Timestamp(42), hb.VTime.Get(5))
}

func TestHighestBeforeTime_Size(t *testing.T) {
	hb := NewHighestBefore(4)
	assert.Equal(t, consensus.ValidatorIndex(4), hb.VTime.Size())
}

func TestEquivocationDetectedSeq(t *testing.T) {
	assert.True(t, EquivocationDetectedSeq.IsEquivocationDetected())

	normal := BranchSeq{Seq: 1, MinSeq: 1}
	assert.False(t, normal.IsEquivocationDetected())
}
