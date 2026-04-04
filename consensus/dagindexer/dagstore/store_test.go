package dagstore

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/dagindexer/dagvec"
	"github.com/0xsoniclabs/kvdb"
	"github.com/0xsoniclabs/kvdb/flushable"
	"github.com/0xsoniclabs/kvdb/memorydb"
	"github.com/0xsoniclabs/kvdb/table"
)

func newTestStore() *Store {
	s := NewStore(func(err error) { panic(err) }, CacheConfig{
		HighestBeforeTimeSize: 1024,
		HighestBeforeSeqSize:  1024,
		LowestAfterSeqSize:    1024,
	})
	s.InitTables(flushable.Wrap(memorydb.New()))
	return s
}

func TestStore_HighestBefore_RoundTrip(t *testing.T) {
	s := newTestStore()
	id := consensus.EventHash{1}
	hb := dagvec.NewHighestBefore(3)
	hb.VSeq.Set(0, dagvec.BranchSeq{Seq: 10, MinSeq: 5})
	hb.VTime.Set(0, 1000)

	s.SetHighestBefore(id, hb)
	got := s.GetHighestBefore(id)
	assert.NotNil(t, got)
	assert.Equal(t, dagvec.BranchSeq{Seq: 10, MinSeq: 5}, got.VSeq.Get(0))
	assert.Equal(t, dagvec.Timestamp(1000), got.VTime.Get(0))
}

func TestStore_HighestBefore_NotFound(t *testing.T) {
	s := newTestStore()
	id := consensus.EventHash{99}
	got := s.GetHighestBefore(id)
	assert.Nil(t, got)
}

func TestStore_HighestBefore_CacheHit(t *testing.T) {
	s := newTestStore()
	id := consensus.EventHash{1}
	hb := dagvec.NewHighestBefore(3)
	hb.VSeq.Set(0, dagvec.BranchSeq{Seq: 5, MinSeq: 2})
	hb.VTime.Set(0, 500)

	s.SetHighestBefore(id, hb)
	// Second read should come from cache
	got := s.GetHighestBefore(id)
	assert.NotNil(t, got)
	assert.Equal(t, dagvec.Timestamp(500), got.VTime.Get(0))
}

func TestStore_LowestAfter_RoundTrip(t *testing.T) {
	s := newTestStore()
	id := consensus.EventHash{2}
	la := dagvec.NewLowestAfterSeq(3)
	la.Set(0, 7)
	la.Set(2, 14)

	s.SetLowestAfter(id, la)
	got := s.GetLowestAfter(id)
	assert.NotNil(t, got)
	assert.Equal(t, consensus.Seq(7), got.Get(0))
	assert.Equal(t, consensus.Seq(14), got.Get(2))
}

func TestStore_LowestAfter_NotFound(t *testing.T) {
	s := newTestStore()
	id := consensus.EventHash{99}
	got := s.GetLowestAfter(id)
	assert.Nil(t, got)
}

func TestStore_LowestAfter_CacheHit(t *testing.T) {
	s := newTestStore()
	id := consensus.EventHash{2}
	la := dagvec.NewLowestAfterSeq(3)
	la.Set(0, 7)

	s.SetLowestAfter(id, la)
	// Second read should come from cache
	got := s.GetLowestAfter(id)
	assert.NotNil(t, got)
	assert.Equal(t, consensus.Seq(7), got.Get(0))
}

func TestStore_LowestAfter_DBHit(t *testing.T) {
	s := newTestStore()
	id := consensus.EventHash{2}
	la := dagvec.NewLowestAfterSeq(3)
	la.Set(0, 7)
	la.Set(1, 12)

	s.SetLowestAfter(id, la)
	// Purge cache so the next read goes to DB
	s.cache.LowestAfterSeq.Purge()

	got := s.GetLowestAfter(id)
	assert.NotNil(t, got)
	assert.Equal(t, consensus.Seq(7), got.Get(0))
	assert.Equal(t, consensus.Seq(12), got.Get(1))
}

func TestStore_EventBranchID_RoundTrip(t *testing.T) {
	s := newTestStore()
	id := consensus.EventHash{3}
	s.SetEventBranchID(id, 42)
	got := s.GetEventBranchID(id)
	assert.Equal(t, consensus.ValidatorIndex(42), got)
}

func TestStore_EventBranchID_NotFound(t *testing.T) {
	s := newTestStore()
	id := consensus.EventHash{99}
	assert.Panics(t, func() {
		s.GetEventBranchID(id)
	})
}

func TestStore_BranchesInfoBytes_RoundTrip(t *testing.T) {
	s := newTestStore()
	data := []byte{0xde, 0xad, 0xbe, 0xef}
	s.SetBranchesInfoBytes(data)
	got := s.GetBranchesInfoBytes()
	assert.Equal(t, data, got)
}

func TestStore_BranchesInfoBytes_NotFound(t *testing.T) {
	s := newTestStore()
	got := s.GetBranchesInfoBytes()
	assert.Nil(t, got)
}

func TestStore_OnDropNotFlushed(t *testing.T) {
	s := newTestStore()
	id := consensus.EventHash{1}
	hb := dagvec.NewHighestBefore(3)
	hb.VSeq.Set(0, dagvec.BranchSeq{Seq: 5, MinSeq: 2})
	hb.VTime.Set(0, 500)
	s.SetHighestBefore(id, hb)

	la := dagvec.NewLowestAfterSeq(3)
	la.Set(0, 7)
	s.SetLowestAfter(id, la)

	s.OnDropNotFlushed()
	// After purging caches, reads go to DB (which is flushed).
	// Data should still be readable from DB.
	got := s.GetHighestBefore(id)
	assert.NotNil(t, got)
}

func TestStore_InitTables(t *testing.T) {
	s := NewStore(func(err error) { panic(err) }, CacheConfig{
		HighestBeforeTimeSize: 1024,
		HighestBeforeSeqSize:  1024,
		LowestAfterSeqSize:    1024,
	})
	db := flushable.Wrap(memorydb.New())
	s.InitTables(db)
	assert.Equal(t, db, s.Db)
}

// errKVStore is a minimal kvdb.Store that returns errors from Get and Put.
// Used to test error paths in dagstore without hanging.
type errKVStore struct {
	kvdb.Store
	err error
}

func (e *errKVStore) Get(key []byte) ([]byte, error) { return nil, e.err }
func (e *errKVStore) Put(key, value []byte) error    { return e.err }
func (e *errKVStore) Has(key []byte) (bool, error)   { return false, e.err }

// newStoreWithErrTable creates a Store where a single table is replaced
// with one backed by errKVStore, so Get/Put return the given error.
func newStoreWithErrTable(errVal error) *Store {
	s := newTestStore()
	errDB := &errKVStore{Store: s.Db, err: errVal}
	// Replace all tables with error-returning versions.
	s.Table.HighestBeforeSeq = table.New(errDB, []byte("S"))
	s.Table.HighestBeforeTime = table.New(errDB, []byte("T"))
	s.Table.LowestAfterSeq = table.New(errDB, []byte("s"))
	s.Table.EventBranch = table.New(errDB, []byte("b"))
	s.Table.BranchesInfo = table.New(errDB, []byte("B"))
	return s
}

func TestStore_getBytes_Error(t *testing.T) {
	s := newStoreWithErrTable(errors.New("get failed"))
	s.crit = func(err error) { panic(err) }
	id := consensus.EventHash{1}
	assert.Panics(t, func() {
		s.getBytes(s.Table.HighestBeforeSeq, id)
	})
}

func TestStore_setBytes_Error(t *testing.T) {
	s := newStoreWithErrTable(errors.New("put failed"))
	s.crit = func(err error) { panic(err) }
	id := consensus.EventHash{1}
	assert.Panics(t, func() {
		s.setBytes(s.Table.HighestBeforeSeq, id, []byte{1, 2, 3})
	})
}

func TestStore_SetBranchesInfoBytes_Error(t *testing.T) {
	s := newStoreWithErrTable(errors.New("put failed"))
	s.crit = func(err error) { panic(err) }
	assert.Panics(t, func() {
		s.SetBranchesInfoBytes([]byte{1, 2, 3})
	})
}

func TestStore_GetBranchesInfoBytes_Error(t *testing.T) {
	s := newStoreWithErrTable(errors.New("get failed"))
	s.crit = func(err error) { panic(err) }
	assert.Panics(t, func() {
		s.GetBranchesInfoBytes()
	})
}

func TestStore_GetEventBranchID_NilReturn(t *testing.T) {
	s := newTestStore()
	// EventBranch not set for this ID => getBytes returns nil => crit
	id := consensus.EventHash{99}
	assert.Panics(t, func() {
		s.GetEventBranchID(id)
	})
}
