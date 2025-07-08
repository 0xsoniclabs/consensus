// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package dagindexer

import (
	"errors"
	"fmt"
	"math"

	"github.com/0xsoniclabs/cacheutils/simplelru"
	"github.com/0xsoniclabs/cacheutils/wlru"
	"github.com/0xsoniclabs/consensus/consensus/vecflushable"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/0xsoniclabs/cacheutils/cachescale"
	"github.com/0xsoniclabs/cacheutils/simplewlru"
	"github.com/0xsoniclabs/consensus/consensus"

	"github.com/0xsoniclabs/kvdb"
	"github.com/0xsoniclabs/kvdb/table"
)

// UNIX nanoseconds timestamp
type Timestamp = uint64

const NilLastEventUid = consensus.Seq(math.MaxUint32)

// IndexCacheConfig - config for cache sizes of Engine
type IndexCacheConfig struct {
	HighestBeforeTimeSize uint
	DBCache               int
	StronglyReachPairs    int
	HighestBeforeSeqSize  uint
	LowestAfterSeqSize    uint
	EventUIdSize          uint
}

// IndexConfig - Engine config (cache sizes)
type IndexConfig struct {
	Caches IndexCacheConfig
}

// Index is a data to detect strongly-reach condition, calculate median timestamp, detect equivocations.
type Index struct {
	crit          func(error)
	validators    *consensus.Validators
	validatorIdxs map[consensus.ValidatorID]consensus.ValidatorIndex

	branchesInfo *BranchesInfo

	getEvent func(consensus.EventHash) consensus.Event

	vecDb kvdb.FlushableKVStore
	table struct {
		HighestBeforeTime kvdb.Store `table:"T"`
		EventBranch       kvdb.Store `table:"b"`
		BranchesInfo      kvdb.Store `table:"B"`
		HighestBeforeSeq  kvdb.Store `table:"S"`
		LowestAfterSeq    kvdb.Store `table:"s"`
		EventUidTable     kvdb.Store `table:"l"`
	}

	cache struct {
		HighestBeforeTime *wlru.Cache
		HighestBeforeSeq  *simplewlru.Cache
		LowestAfterSeq    *simplewlru.Cache
		EventId           *simplelru.Cache
		LastEventId       consensus.Seq
	}

	cfg IndexConfig
}

// DefaultConfig returns default index config
func DefaultConfig(scale cachescale.Func) IndexConfig {
	return IndexConfig{
		Caches: IndexCacheConfig{
			HighestBeforeTimeSize: scale.U(160 * 1024),
			DBCache:               scale.I(10 * opt.MiB),
			HighestBeforeSeqSize:  scale.U(320 * 1024),
			LowestAfterSeqSize:    scale.U(160 * 1024),
			EventUIdSize:          scale.U(100),
		},
	}
}

// LiteConfig returns default index config for tests
func LiteConfig() IndexConfig {
	scale := cachescale.Ratio{Base: 6, Target: 10}
	return IndexConfig{
		Caches: IndexCacheConfig{
			HighestBeforeTimeSize: 4 * 1024,
			HighestBeforeSeqSize:  scale.U(320 * 1024),
			LowestAfterSeqSize:    scale.U(160 * 1024),
			EventUIdSize:          scale.U(100),
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
func (vi *Index) Add(e consensus.Event) error {
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
	vi.cache.HighestBeforeSeq, _ = simplewlru.New(vi.cfg.Caches.HighestBeforeSeqSize, int(vi.cfg.Caches.HighestBeforeSeqSize))
	vi.cache.LowestAfterSeq, _ = simplewlru.New(vi.cfg.Caches.LowestAfterSeqSize, int(vi.cfg.Caches.HighestBeforeSeqSize))
	vi.cache.EventId, _ = simplelru.New(int(vi.cfg.Caches.EventUIdSize))
	vi.cache.LastEventId = NilLastEventUid
}

// DropNotFlushed not connected clocks. Call it if event has failed.
func (vi *Index) DropNotFlushed() {
	vi.branchesInfo = nil
	if vi.vecDb.NotFlushedPairs() != 0 {
		vi.vecDb.DropNotFlushed()
		vi.OnDropNotFlushed()
	}
}

func (vi *Index) WrapWithFlushable(db kvdb.Store) kvdb.FlushableKVStore {
	return vecflushable.Wrap(db, vi.cfg.Caches.DBCache)
}

// Reset resets buffers.
func (vi *Index) Reset(validators *consensus.Validators, db kvdb.FlushableKVStore, getEvent func(consensus.EventHash) consensus.Event) {
	vi.vecDb = db
	vi.getEvent = getEvent
	vi.validators = validators
	vi.validatorIdxs = validators.Idxs()
	vi.DropNotFlushed()
	table.MigrateTables(&vi.table, vi.vecDb)
	vi.OnDropNotFlushed()
	// vi.setLastUId(0)
}

func (vi *Index) Close() error {
	return vi.vecDb.Close()
}

func (vi *Index) setEquivocationDetected(before *HighestBefore, branchID consensus.ValidatorIndex) {
	creatorIdx := vi.branchesInfo.BranchIDCreatorIdxs[branchID]
	for _, branchID := range vi.branchesInfo.BranchIDByCreators[creatorIdx] {
		before.SetEquivocationDetected(branchID)
	}
}

func (vi *Index) fillGlobalBranchID(e consensus.Event, meIdx consensus.ValidatorIndex) (consensus.ValidatorIndex, error) {
	// sanity checks
	if len(vi.branchesInfo.BranchIDCreatorIdxs) != len(vi.branchesInfo.BranchIDLastSeq) {
		return 0, errors.New("inconsistent BranchIDCreators len (inconsistent DB)")
	}
	if consensus.ValidatorIndex(len(vi.branchesInfo.BranchIDCreatorIdxs)) < vi.validators.Len() {
		return 0, errors.New("inconsistent BranchIDCreators len (inconsistent DB)")
	}

	if e.SelfParent() == nil {
		// is it first event indeed?
		if vi.branchesInfo.BranchIDLastSeq[meIdx] == 0 {
			// OK, not a new equivocation
			vi.branchesInfo.BranchIDLastSeq[meIdx] = e.Seq()
			return meIdx, nil
		}
	} else {
		selfParentBranchID := vi.GetEventBranchID(*e.SelfParent())
		// sanity checks
		if len(vi.branchesInfo.BranchIDCreatorIdxs) != len(vi.branchesInfo.BranchIDLastSeq) {
			return 0, errors.New("inconsistent BranchIDCreators len (inconsistent DB)")
		}

		if vi.branchesInfo.BranchIDLastSeq[selfParentBranchID]+1 == e.Seq() {
			vi.branchesInfo.BranchIDLastSeq[selfParentBranchID] = e.Seq()
			// OK, not a new equivocation
			return selfParentBranchID, nil
		}
	}

	// if we're here, then new equivocation is reachable (only globally), create new branchID due to a new equivocation
	vi.branchesInfo.BranchIDLastSeq = append(vi.branchesInfo.BranchIDLastSeq, e.Seq())
	vi.branchesInfo.BranchIDCreatorIdxs = append(vi.branchesInfo.BranchIDCreatorIdxs, meIdx)
	newBranchID := consensus.ValidatorIndex(len(vi.branchesInfo.BranchIDLastSeq) - 1)
	vi.branchesInfo.BranchIDByCreators[meIdx] = append(vi.branchesInfo.BranchIDByCreators[meIdx], newBranchID)
	return newBranchID, nil
}

// fillEventVectors calculates (and stores) event's vectors, and updates LowestAfter of newly-reachable events.
func (vi *Index) fillEventVectors(e consensus.Event) (allVecs, error) {
	meIdx := vi.validatorIdxs[e.Creator()]
	myVecs := allVecs{
		before: NewHighestBefore(consensus.ValidatorIndex(len(vi.branchesInfo.BranchIDCreatorIdxs))),
		after:  NewLowestAfterSeq(consensus.ValidatorIndex(len(vi.branchesInfo.BranchIDCreatorIdxs))),
	}

	meBranchID, err := vi.fillGlobalBranchID(e, meIdx)
	if err != nil {
		return myVecs, err
	}

	// pre-load parents into RAM for quick access
	parentsVecs := make([]*HighestBefore, len(e.Parents()))
	parentsBranchIDs := make([]consensus.ValidatorIndex, len(e.Parents()))
	for i, p := range e.Parents() {
		parentsBranchIDs[i] = vi.GetEventBranchID(p)
		parentsVecs[i] = vi.GetHighestBefore(p)
		if parentsVecs[i] == nil {
			return myVecs, fmt.Errorf("processed out of order, parent not found (inconsistent DB), parent=%s", p.String())
		}
	}

	uid := vi.addEventUId(e.ID())
	// reachable by himself
	myVecs.after.InitWithEvent(meBranchID, e)
	myVecs.before.InitWithEvent(meBranchID, e, uid)

	newEvents := make([]consensus.Seq, len(vi.branchesInfo.BranchIDCreatorIdxs)) // branchID -> number of new events
	for i, pVec := range parentsVecs {
		diff := newEvents
		if i == 0 && e.SelfParent() != nil {
			diff = nil
		}
		// calculate HighestBefore  Detect equivocations for a case when parent reaches an equivocation
		myVecs.before.CollectFrom(pVec, consensus.ValidatorIndex(len(vi.branchesInfo.BranchIDCreatorIdxs)), diff)
	}
	// Detect equivocations, which were not reachable by parents
	if vi.AtLeastOneEquivocation() {
		for n := consensus.ValidatorIndex(0); n < vi.validators.Len(); n++ {
			if len(vi.branchesInfo.BranchIDByCreators[n]) <= 1 {
				continue
			}
			for _, branchID := range vi.branchesInfo.BranchIDByCreators[n] {
				if myVecs.before.IsEquivocationDetected(branchID) {
					// if one branch reaches an equivocation, mark all the branches as observing the equivocation
					vi.setEquivocationDetected(myVecs.before, n)
					break
				}
			}
		}

	nextCreator:
		for n := consensus.ValidatorIndex(0); n < vi.validators.Len(); n++ {
			if myVecs.before.IsEquivocationDetected(n) {
				continue
			}
			for _, branchID1 := range vi.branchesInfo.BranchIDByCreators[n] {
				for _, branchID2 := range vi.branchesInfo.BranchIDByCreators[n] {
					a := branchID1
					b := branchID2
					if a == b {
						continue
					}

					if myVecs.before.IsEmpty(a) || myVecs.before.IsEmpty(b) {
						continue
					}
					if myVecs.before.MinSeq(a) <= myVecs.before.Seq(b) && myVecs.before.MinSeq(b) <= myVecs.before.Seq(a) {
						vi.setEquivocationDetected(myVecs.before, n)
						continue nextCreator
					}
				}
			}
		}
	}

	// for each branch where new events were inserted, start at the highest
	// ancestor of e on that branch and go down, from self-parent to self-parent,
	// updating each one's lowest descendants, until all new events have been
	// processed.
	for branchID, newEvents := range newEvents {
		if newEvents == 0 || myVecs.before.IsEquivocationDetected(consensus.ValidatorIndex(branchID)) {
			continue
		}
		// if b is the highest ancestor of e on this branch, then blid is its
		// key in the EventLookup, and bh is its hash
		blid := myVecs.before.VSeq.LookupKey(consensus.ValidatorIndex(branchID))
		bh := vi.getEventByUId(blid)
		for remaining := newEvents; remaining > 0; remaining-- {
			wLowestAfterSeq := vi.GetLowestAfter(bh)
			// update LowestAfter vector of the old event, because newly-connected event observes it
			if wLowestAfterSeq.Visit(meBranchID, e) {
				vi.SetLowestAfter(bh, wLowestAfterSeq)
				le := vi.getEvent(bh)
				if sp := le.SelfParent(); sp != nil {
					bh = *sp
					continue
				}
			}
			break
		}
	}
	// store calculated vectors
	vi.SetHighestBefore(e.ID(), myVecs.before)
	vi.SetLowestAfter(e.ID(), myVecs.after)
	vi.SetEventBranchID(e.ID(), meBranchID)

	return myVecs, nil
}

// GetMergedHighestBefore returns HighestBefore vector clock without branches, where branches are merged into one
func (vi *Index) GetMergedHighestBefore(id consensus.EventHash) *HighestBefore {
	vi.InitBranchesInfo()

	if vi.AtLeastOneEquivocation() {
		scatteredBefore := vi.GetHighestBefore(id)

		mergedBefore := NewHighestBefore(vi.validators.Len())

		for creatorIdx, branches := range vi.branchesInfo.BranchIDByCreators {
			mergedBefore.GatherFrom(consensus.ValidatorIndex(creatorIdx), scatteredBefore, branches)
		}

		return mergedBefore
	}
	return vi.GetHighestBefore(id)
}
