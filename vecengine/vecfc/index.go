package vecfc

import (
	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/dag"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/inter/pos"
	"github.com/0xsoniclabs/consensus/kvdb"
	"github.com/0xsoniclabs/consensus/kvdb/table"
	"github.com/0xsoniclabs/consensus/utils/cachescale"
	"github.com/0xsoniclabs/consensus/utils/simplewlru"
	"github.com/0xsoniclabs/consensus/vecengine"
)

// IndexCacheConfig - config for cache sizes of Engine
type IndexCacheConfig struct {
	ForklessCausePairs   int
	HighestBeforeSeqSize uint
	LowestAfterSeqSize   uint
}

// IndexConfig - Engine config (cache sizes)
type IndexConfig struct {
	Caches IndexCacheConfig
}

// Index is a data structure to detect forkless-cause condition, calculate median timestamp, detect forks.
type Index struct {
	*vecengine.Engine

	TableAllVecs struct {
		HighestBeforeSeq kvdb.Store `table:"S"`
		LowestAfterSeq   kvdb.Store `table:"s"`
	}

	Cache struct {
		HighestBeforeSeq *simplewlru.Cache
		LowestAfterSeq   *simplewlru.Cache
		ForklessCause    *simplewlru.Cache
	}

	Config IndexConfig
}

// DefaultConfig returns default index config
func DefaultConfig(scale cachescale.Func) IndexConfig {
	return IndexConfig{
		Caches: IndexCacheConfig{
			ForklessCausePairs:   scale.I(20000),
			HighestBeforeSeqSize: scale.U(160 * 1024),
			LowestAfterSeqSize:   scale.U(160 * 1024),
		},
	}
}

// LiteConfig returns default index config for tests
func LiteConfig() IndexConfig {
	return DefaultConfig(cachescale.Ratio{Base: 100, Target: 1})
}

// NewIndex creates Index instance.
func NewIndex(errorHandler func(error), config IndexConfig) *Index {
	vi := &Index{
		Config: config,
	}
	vi.Engine = vecengine.NewIndex(errorHandler, vi.CreateEngineCallbacks())
	vi.initCaches()

	return vi
}

func (vi *Index) initCaches() {
	vi.Cache.ForklessCause, _ = simplewlru.New(uint(vi.Config.Caches.ForklessCausePairs), vi.Config.Caches.ForklessCausePairs)
	vi.Cache.HighestBeforeSeq, _ = simplewlru.New(vi.Config.Caches.HighestBeforeSeqSize, int(vi.Config.Caches.HighestBeforeSeqSize))
	vi.Cache.LowestAfterSeq, _ = simplewlru.New(vi.Config.Caches.LowestAfterSeqSize, int(vi.Config.Caches.HighestBeforeSeqSize))
}

// Reset resets buffers.
func (vi *Index) Reset(validators *pos.Validators, db kvdb.FlushableKVStore, getEvent func(hash.Event) dag.Event) {
	vi.Engine.Reset(validators, db, getEvent)
	vi.VecDb = db
	table.MigrateTables(&vi.TableAllVecs, vi.VecDb)
	vi.GetEvent = getEvent
	vi.Validators = validators
	vi.ValidatorIdxs = validators.Idxs()
	vi.Cache.ForklessCause.Purge()
	vi.onDropNotFlushed()
}

func (vi *Index) CreateEngineCallbacks() vecengine.Callbacks {
	return vecengine.Callbacks{
		GetHighestBefore: func(event hash.Event) vecengine.HighestBeforeI {
			return vi.GetHighestBefore(event)
		},
		GetLowestAfter: func(event hash.Event) vecengine.LowestAfterI {
			return vi.GetLowestAfter(event)
		},
		SetHighestBefore: func(event hash.Event, b vecengine.HighestBeforeI) {
			vi.SetHighestBefore(event, b.(*HighestBeforeSeq))
		},
		SetLowestAfter: func(event hash.Event, b vecengine.LowestAfterI) {
			vi.SetLowestAfter(event, b.(*LowestAfterSeq))
		},
		NewHighestBefore: func(size idx.Validator) vecengine.HighestBeforeI {
			return NewHighestBeforeSeq(size)
		},
		NewLowestAfter: func(size idx.Validator) vecengine.LowestAfterI {
			return NewLowestAfterSeq(size)
		},
		OnDropNotFlushed: vi.onDropNotFlushed,
	}
}

func (vi *Index) onDropNotFlushed() {
	vi.Cache.HighestBeforeSeq.Purge()
	vi.Cache.LowestAfterSeq.Purge()
}

// GetMergedHighestBefore returns HighestBefore vector clock without branches, where branches are merged into one
func (vi *Index) GetMergedHighestBefore(id hash.Event) *HighestBeforeSeq {
	return vi.Engine.GetMergedHighestBefore(id).(*HighestBeforeSeq)
}
