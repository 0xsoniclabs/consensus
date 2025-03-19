// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package vecengine

import (
	"errors"
	"fmt"

	"github.com/0xsoniclabs/cacheutils/cachescale"
	"github.com/0xsoniclabs/cacheutils/simplewlru"

	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/dag"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/inter/pos"
	"github.com/0xsoniclabs/kvdb"
	"github.com/0xsoniclabs/kvdb/table"
)

type Callbacks struct {
	GetHighestBefore func(hash.Event) HighestBeforeI
	GetLowestAfter   func(hash.Event) LowestAfterI
	SetHighestBefore func(hash.Event, HighestBeforeI)
	SetLowestAfter   func(hash.Event, LowestAfterI)
	NewHighestBefore func(idx.Validator) HighestBeforeI
	NewLowestAfter   func(idx.Validator) LowestAfterI
	OnDropNotFlushed func()
}

// Index is a data to detect forkless-cause condition, calculate median timestamp, detect forks.
type Engine struct {
	crit          func(error)
	validators    *pos.Validators
	validatorIdxs map[idx.ValidatorID]idx.Validator

	bi *BranchesInfo

	getEvent func(hash.Event) dag.Event

	Callbacks Callbacks

	vecDb kvdb.FlushableKVStore

	table struct {
		EventBranch      kvdb.Store `table:"b"`
		BranchesInfo     kvdb.Store `table:"B"`
		HighestBeforeSeq kvdb.Store `table:"S"`
		LowestAfterSeq   kvdb.Store `table:"s"`
	}

	cache struct {
		HighestBeforeSeq *simplewlru.Cache
		LowestAfterSeq   *simplewlru.Cache
		ForklessCause    *simplewlru.Cache
	}

	cfg IndexConfig
}

// NewIndex creates Index instance.
func NewIndex(crit func(error), config IndexConfig, getCallbacks func(vi *Engine) Callbacks) *Engine {
	vi := &Engine{
		cfg:  config,
		crit: crit,
	}
	vi.Callbacks = getCallbacks(vi)
	vi.initCaches()

	return vi
}

// Add calculates vector clocks for the event and saves into DB.
func (vi *Engine) Add(e dag.Event) error {
	vi.InitBranchesInfo()
	_, err := vi.fillEventVectors(e)
	return err
}

// Flush writes vector clocks to persistent store.
func (vi *Engine) Flush() {
	if vi.bi != nil {
		vi.setBranchesInfo(vi.bi)
	}
	if err := vi.vecDb.Flush(); err != nil {
		vi.crit(err)
	}
}

// DropNotFlushed not connected clocks. Call it if event has failed.
func (vi *Engine) DropNotFlushed() {
	vi.bi = nil
	if vi.vecDb.NotFlushedPairs() != 0 {
		vi.vecDb.DropNotFlushed()
		if vi.Callbacks.OnDropNotFlushed != nil {
			vi.Callbacks.OnDropNotFlushed()
		}
	}
}

func (vi *Engine) setForkDetected(before HighestBeforeI, branchID idx.Validator) {
	creatorIdx := vi.bi.BranchIDCreatorIdxs[branchID]
	for _, branchID := range vi.bi.BranchIDByCreators[creatorIdx] {
		before.SetForkDetected(branchID)
	}
}

func (vi *Engine) fillGlobalBranchID(e dag.Event, meIdx idx.Validator) (idx.Validator, error) {
	// sanity checks
	if len(vi.bi.BranchIDCreatorIdxs) != len(vi.bi.BranchIDLastSeq) {
		return 0, errors.New("inconsistent BranchIDCreators len (inconsistent DB)")
	}
	if idx.Validator(len(vi.bi.BranchIDCreatorIdxs)) < vi.validators.Len() {
		return 0, errors.New("inconsistent BranchIDCreators len (inconsistent DB)")
	}

	if e.SelfParent() == nil {
		// is it first event indeed?
		if vi.bi.BranchIDLastSeq[meIdx] == 0 {
			// OK, not a new fork
			vi.bi.BranchIDLastSeq[meIdx] = e.Seq()
			return meIdx, nil
		}
	} else {
		selfParentBranchID := vi.GetEventBranchID(*e.SelfParent())
		// sanity checks
		if len(vi.bi.BranchIDCreatorIdxs) != len(vi.bi.BranchIDLastSeq) {
			return 0, errors.New("inconsistent BranchIDCreators len (inconsistent DB)")
		}

		if vi.bi.BranchIDLastSeq[selfParentBranchID]+1 == e.Seq() {
			vi.bi.BranchIDLastSeq[selfParentBranchID] = e.Seq()
			// OK, not a new fork
			return selfParentBranchID, nil
		}
	}

	// if we're here, then new fork is observed (only globally), create new branchID due to a new fork
	vi.bi.BranchIDLastSeq = append(vi.bi.BranchIDLastSeq, e.Seq())
	vi.bi.BranchIDCreatorIdxs = append(vi.bi.BranchIDCreatorIdxs, meIdx)
	newBranchID := idx.Validator(len(vi.bi.BranchIDLastSeq) - 1)
	vi.bi.BranchIDByCreators[meIdx] = append(vi.bi.BranchIDByCreators[meIdx], newBranchID)
	return newBranchID, nil
}

// fillEventVectors calculates (and stores) event's vectors, and updates LowestAfter of newly-observed events.
func (vi *Engine) fillEventVectors(e dag.Event) (allVecs, error) {
	meIdx := vi.validatorIdxs[e.Creator()]
	myVecs := allVecs{
		before: vi.Callbacks.NewHighestBefore(idx.Validator(len(vi.bi.BranchIDCreatorIdxs))),
		after:  vi.Callbacks.NewLowestAfter(idx.Validator(len(vi.bi.BranchIDCreatorIdxs))),
	}

	meBranchID, err := vi.fillGlobalBranchID(e, meIdx)
	if err != nil {
		return myVecs, err
	}

	// pre-load parents into RAM for quick access
	parentsVecs := make([]HighestBeforeI, len(e.Parents()))
	parentsBranchIDs := make([]idx.Validator, len(e.Parents()))
	for i, p := range e.Parents() {
		parentsBranchIDs[i] = vi.GetEventBranchID(p)
		parentsVecs[i] = vi.Callbacks.GetHighestBefore(p)
		if parentsVecs[i] == nil {
			return myVecs, fmt.Errorf("processed out of order, parent not found (inconsistent DB), parent=%s", p.String())
		}
	}

	// observed by himself
	myVecs.after.InitWithEvent(meBranchID, e)
	myVecs.before.InitWithEvent(meBranchID, e)

	for _, pVec := range parentsVecs {
		// calculate HighestBefore  Detect forks for a case when parent observes a fork
		myVecs.before.CollectFrom(pVec, idx.Validator(len(vi.bi.BranchIDCreatorIdxs)))
	}
	// Detect forks, which were not observed by parents
	if vi.AtLeastOneFork() {
		for n := idx.Validator(0); n < vi.validators.Len(); n++ {
			if len(vi.bi.BranchIDByCreators[n]) <= 1 {
				continue
			}
			for _, branchID := range vi.bi.BranchIDByCreators[n] {
				if myVecs.before.IsForkDetected(branchID) {
					// if one branch observes a fork, mark all the branches as observing the fork
					vi.setForkDetected(myVecs.before, n)
					break
				}
			}
		}

	nextCreator:
		for n := idx.Validator(0); n < vi.validators.Len(); n++ {
			if myVecs.before.IsForkDetected(n) {
				continue
			}
			for _, branchID1 := range vi.bi.BranchIDByCreators[n] {
				for _, branchID2 := range vi.bi.BranchIDByCreators[n] {
					a := branchID1
					b := branchID2
					if a == b {
						continue
					}

					if myVecs.before.IsEmpty(a) || myVecs.before.IsEmpty(b) {
						continue
					}
					if myVecs.before.MinSeq(a) <= myVecs.before.Seq(b) && myVecs.before.MinSeq(b) <= myVecs.before.Seq(a) {
						vi.setForkDetected(myVecs.before, n)
						continue nextCreator
					}
				}
			}
		}
	}

	// graph traversal starting from e, but excluding e
	onWalk := func(walk hash.Event) (godeeper bool) {
		wLowestAfterSeq := vi.Callbacks.GetLowestAfter(walk)

		// update LowestAfter vector of the old event, because newly-connected event observes it
		if wLowestAfterSeq.Visit(meBranchID, e) {
			vi.Callbacks.SetLowestAfter(walk, wLowestAfterSeq)
			return true
		}
		return false
	}
	err = vi.DfsSubgraph(e, onWalk)
	if err != nil {
		vi.crit(err)
	}

	// store calculated vectors
	vi.Callbacks.SetHighestBefore(e.ID(), myVecs.before)
	vi.Callbacks.SetLowestAfter(e.ID(), myVecs.after)
	vi.SetEventBranchID(e.ID(), meBranchID)

	return myVecs, nil
}

func (vi *Engine) GetMergedHighestBefore(id hash.Event) HighestBeforeI {
	vi.InitBranchesInfo()

	if vi.AtLeastOneFork() {
		scatteredBefore := vi.Callbacks.GetHighestBefore(id)

		mergedBefore := vi.Callbacks.NewHighestBefore(vi.validators.Len())

		for creatorIdx, branches := range vi.bi.BranchIDByCreators {
			mergedBefore.GatherFrom(idx.Validator(creatorIdx), scatteredBefore, branches)
		}

		return mergedBefore
	}
	return vi.Callbacks.GetHighestBefore(id)
}

func (vi *Engine) initCaches() {
	vi.cache.ForklessCause, _ = simplewlru.New(uint(vi.cfg.Caches.ForklessCausePairs), vi.cfg.Caches.ForklessCausePairs)
	vi.cache.HighestBeforeSeq, _ = simplewlru.New(vi.cfg.Caches.HighestBeforeSeqSize, int(vi.cfg.Caches.HighestBeforeSeqSize))
	vi.cache.LowestAfterSeq, _ = simplewlru.New(vi.cfg.Caches.LowestAfterSeqSize, int(vi.cfg.Caches.HighestBeforeSeqSize))
}

func GetEngineCallbacks(vi *Engine) Callbacks {
	return Callbacks{
		GetHighestBefore: func(event hash.Event) HighestBeforeI {
			return vi.GetHighestBefore(event)
		},
		GetLowestAfter: func(event hash.Event) LowestAfterI {
			return vi.GetLowestAfter(event)
		},
		SetHighestBefore: func(event hash.Event, b HighestBeforeI) {
			vi.SetHighestBefore(event, b.(*HighestBeforeSeq))
		},
		SetLowestAfter: func(event hash.Event, b LowestAfterI) {
			vi.SetLowestAfter(event, b.(*LowestAfterSeq))
		},
		NewHighestBefore: func(size idx.Validator) HighestBeforeI {
			return NewHighestBeforeSeq(size)
		},
		NewLowestAfter: func(size idx.Validator) LowestAfterI {
			return NewLowestAfterSeq(size)
		},
		OnDropNotFlushed: func() { vi.onDropNotFlushed() },
	}
}

// Reset resets buffers.
func (vi *Engine) Reset(validators *pos.Validators, db kvdb.FlushableKVStore, getEvent func(hash.Event) dag.Event) {
	vi.getEvent = getEvent
	vi.vecDb = db
	vi.validators = validators
	vi.validatorIdxs = validators.Idxs()
	vi.DropNotFlushed()
	table.MigrateTables(&vi.table, vi.vecDb)
	vi.getEvent = getEvent
	vi.cache.ForklessCause.Purge()
	vi.onDropNotFlushed()
}

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

func (vi *Engine) onDropNotFlushed() {
	vi.cache.HighestBeforeSeq.Purge()
	vi.cache.LowestAfterSeq.Purge()
}
