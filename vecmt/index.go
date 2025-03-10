package vecmt

import (
	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/dag"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/inter/pos"
	"github.com/0xsoniclabs/consensus/kvdb"
	"github.com/0xsoniclabs/consensus/kvdb/table"
	"github.com/0xsoniclabs/consensus/utils/cachescale"
	"github.com/0xsoniclabs/consensus/utils/simplewlru"
	"github.com/0xsoniclabs/consensus/utils/wlru"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// UNIX nanoseconds timestamp
type Timestamp = uint64

// IndexCacheConfig - config for cache sizes of Engine
type IndexCacheConfig struct {
	HighestBeforeTimeSize uint
	DBCache               int
	ForklessCausePairs    int
	HighestBeforeSeqSize  uint
	LowestAfterSeqSize    uint
}

// IndexConfig - Engine config (cache sizes)
type IndexConfig struct {
	Caches IndexCacheConfig
}

// Index is a data to detect forkless-cause condition, calculate median timestamp, detect forks.
type Index struct {
	//*vecengine.Engine

	crit          func(error)
	validators    *pos.Validators
	validatorIdxs map[idx.ValidatorID]idx.Validator

	branchesInfo *BranchesInfo

	getEvent func(hash.Event) dag.Event

	vecDb *VecFlushable
	table struct {
		HighestBeforeTime kvdb.Store `table:"T"`
		EventBranch       kvdb.Store `table:"b"`
		BranchesInfo      kvdb.Store `table:"B"`
		HighestBeforeSeq  kvdb.Store `table:"S"`
		LowestAfterSeq    kvdb.Store `table:"s"`
	}

	cache struct {
		HighestBeforeTime *wlru.Cache
		ForklessCause     *simplewlru.Cache
		HighestBeforeSeq  *simplewlru.Cache
		LowestAfterSeq    *simplewlru.Cache
	}

	cfg IndexConfig
}

// DefaultConfig returns default index config
func DefaultConfig(scale cachescale.Func) IndexConfig {
	return IndexConfig{
		Caches: IndexCacheConfig{
			HighestBeforeTimeSize: scale.U(160 * 1024),
			DBCache:               scale.I(10 * opt.MiB),
			ForklessCausePairs:    scale.I(20000),
			HighestBeforeSeqSize:  scale.U(160 * 1024),
			LowestAfterSeqSize:    scale.U(160 * 1024),
		},
	}
}

// LiteConfig returns default index config for tests
func LiteConfig() IndexConfig {
	scale := cachescale.Ratio{Base: 100, Target: 1}
	return IndexConfig{
		Caches: IndexCacheConfig{
			HighestBeforeTimeSize: 4 * 1024,
			ForklessCausePairs:    scale.I(20000),
			HighestBeforeSeqSize:  scale.U(160 * 1024),
			LowestAfterSeqSize:    scale.U(160 * 1024),
		},
	}
}

// NewIndex creates Index instance.
func NewIndex(crit func(error), config IndexConfig) *Index {
	vi := &Index{
		cfg:  config,
		crit: crit,
	}

	vi.initCaches()

	return vi
}

// Add calculates vector clocks for the event and saves into DB.
func (vi *Index) Add(e dag.Event) error {
	vi.InitBranchesInfo()
	_, err := vi.fillEventVectors(e)
	return err
}

// Flush writes vector clocks to persistent store.
func (vi *Index) Flush() {
	if vi.branchesInfo != nil {
		vi.setBranchesInfo(vi.branchesInfo)
	}
	if err := vi.vecDb.Flush(); err != nil {
		vi.crit(err)
	}
}

func (vi *Index) initCaches() {
	vi.cache.HighestBeforeTime, _ = wlru.New(vi.cfg.Caches.HighestBeforeTimeSize, int(vi.cfg.Caches.HighestBeforeTimeSize))
	vi.cache.ForklessCause, _ = simplewlru.New(uint(vi.cfg.Caches.ForklessCausePairs), vi.cfg.Caches.ForklessCausePairs)
	vi.cache.HighestBeforeSeq, _ = simplewlru.New(vi.cfg.Caches.HighestBeforeSeqSize, int(vi.cfg.Caches.HighestBeforeSeqSize))
	vi.cache.LowestAfterSeq, _ = simplewlru.New(vi.cfg.Caches.LowestAfterSeqSize, int(vi.cfg.Caches.HighestBeforeSeqSize))
}

// DropNotFlushed not connected clocks. Call it if event has failed.
func (vi *Index) DropNotFlushed() {
	vi.branchesInfo = nil
	if vi.vecDb.NotFlushedPairs() != 0 {
		vi.vecDb.DropNotFlushed()
		vi.OnDropNotFlushed()
	}
}

// Reset resets buffers.
func (vi *Index) Reset(validators *pos.Validators, db kvdb.Store, getEvent func(hash.Event) dag.Event) {
	fdb := WrapByVecFlushable(db, vi.cfg.Caches.DBCache)
	vi.vecDb = fdb
	vi.getEvent = getEvent
	vi.validators = validators
	vi.validatorIdxs = validators.Idxs()
	vi.DropNotFlushed()
	table.MigrateTables(&vi.table, vi.vecDb)
	vi.cache.ForklessCause.Purge()
	vi.OnDropNotFlushed()
}

func (vi *Index) Close() error {
	return vi.vecDb.Close()
}

// GetMergedHighestBefore returns HighestBefore vector clock without branches, where branches are merged into one
func (vi *Index) GetMergedHighestBefore(id hash.Event) *HighestBefore {
	vi.InitBranchesInfo()

	if vi.AtLeastOneFork() {
		scatteredBefore := vi.GetHighestBefore(id)

		mergedBefore := NewHighestBefore(vi.validators.Len())

		for creatorIdx, branches := range vi.branchesInfo.BranchIDByCreators {
			mergedBefore.GatherFrom(idx.Validator(creatorIdx), scatteredBefore, branches)
		}

		return mergedBefore
	}
	return vi.GetHighestBefore(id)
}
