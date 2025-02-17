package vecengine

import (
	"errors"
	"fmt"

	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/dag"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/inter/pos"
	"github.com/0xsoniclabs/consensus/kvdb"
	"github.com/0xsoniclabs/consensus/kvdb/table"
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

type Engine struct {
	ErrorHandler  func(error)
	Validators    *pos.Validators
	ValidatorIdxs map[idx.ValidatorID]idx.Validator

	BranchesInfo *BranchesInfo

	GetEvent func(hash.Event) dag.Event

	Callback Callbacks

	VecDb kvdb.FlushableKVStore
	Table struct {
		EventBranch  kvdb.Store `table:"b"`
		BranchesInfo kvdb.Store `table:"B"`
	}
}

// NewIndex creates Engine instance.
func NewIndex(errorHandler func(error), callbacks Callbacks) *Engine {
	vi := &Engine{
		ErrorHandler: errorHandler,
		Callback:     callbacks,
	}

	return vi
}

// Reset resets buffers.
func (vi *Engine) Reset(validators *pos.Validators, db kvdb.FlushableKVStore, getEvent func(hash.Event) dag.Event) {
	// use wrapper to be able to drop failed events by dropping cache
	vi.GetEvent = getEvent
	vi.VecDb = db
	vi.Validators = validators
	vi.ValidatorIdxs = validators.Idxs()
	vi.DropNotFlushed()

	table.MigrateTables(&vi.Table, vi.VecDb)
}

// Add calculates vector clocks for the event and saves into DB.
func (vi *Engine) Add(e dag.Event) error {
	vi.InitBranchesInfo()
	_, err := vi.fillEventVectors(e)
	return err
}

// Flush writes vector clocks to persistent store.
func (vi *Engine) Flush() {
	if vi.BranchesInfo != nil {
		vi.setBranchesInfo(vi.BranchesInfo)
	}
	if err := vi.VecDb.Flush(); err != nil {
		vi.ErrorHandler(err)
	}
}

// DropNotFlushed not connected clocks. Call it if event has failed.
func (vi *Engine) DropNotFlushed() {
	vi.BranchesInfo = nil
	if vi.VecDb.NotFlushedPairs() != 0 {
		vi.VecDb.DropNotFlushed()
		if vi.Callback.OnDropNotFlushed != nil {
			vi.Callback.OnDropNotFlushed()
		}
	}
}

func (vi *Engine) setForkDetected(before HighestBeforeI, branchID idx.Validator) {
	creatorIdx := vi.BranchesInfo.BranchIDCreatorIdxs[branchID]
	for _, branchID := range vi.BranchesInfo.BranchIDByCreators[creatorIdx] {
		before.SetForkDetected(branchID)
	}
}

func (vi *Engine) fillGlobalBranchID(e dag.Event, meIdx idx.Validator) (idx.Validator, error) {
	// sanity checks
	if len(vi.BranchesInfo.BranchIDCreatorIdxs) != len(vi.BranchesInfo.BranchIDLastSeq) {
		return 0, errors.New("inconsistent BranchIDCreators len (inconsistent DB)")
	}
	if idx.Validator(len(vi.BranchesInfo.BranchIDCreatorIdxs)) < vi.Validators.Len() {
		return 0, errors.New("inconsistent BranchIDCreators len (inconsistent DB)")
	}

	if e.SelfParent() == nil {
		// is it first event indeed?
		if vi.BranchesInfo.BranchIDLastSeq[meIdx] == 0 {
			// OK, not a new fork
			vi.BranchesInfo.BranchIDLastSeq[meIdx] = e.Seq()
			return meIdx, nil
		}
	} else {
		selfParentBranchID := vi.GetEventBranchID(*e.SelfParent())
		// sanity checks
		if len(vi.BranchesInfo.BranchIDCreatorIdxs) != len(vi.BranchesInfo.BranchIDLastSeq) {
			return 0, errors.New("inconsistent BranchIDCreators len (inconsistent DB)")
		}

		if vi.BranchesInfo.BranchIDLastSeq[selfParentBranchID]+1 == e.Seq() {
			vi.BranchesInfo.BranchIDLastSeq[selfParentBranchID] = e.Seq()
			// OK, not a new fork
			return selfParentBranchID, nil
		}
	}

	// if we're here, then new fork is observed (only globally), create new branchID due to a new fork
	vi.BranchesInfo.BranchIDLastSeq = append(vi.BranchesInfo.BranchIDLastSeq, e.Seq())
	vi.BranchesInfo.BranchIDCreatorIdxs = append(vi.BranchesInfo.BranchIDCreatorIdxs, meIdx)
	newBranchID := idx.Validator(len(vi.BranchesInfo.BranchIDLastSeq) - 1)
	vi.BranchesInfo.BranchIDByCreators[meIdx] = append(vi.BranchesInfo.BranchIDByCreators[meIdx], newBranchID)
	return newBranchID, nil
}

// fillEventVectors calculates (and stores) event's vectors, and updates LowestAfter of newly-observed events.
func (vi *Engine) fillEventVectors(e dag.Event) (allVecs, error) {
	meIdx := vi.ValidatorIdxs[e.Creator()]
	myVecs := allVecs{
		before: vi.Callback.NewHighestBefore(idx.Validator(len(vi.BranchesInfo.BranchIDCreatorIdxs))),
		after:  vi.Callback.NewLowestAfter(idx.Validator(len(vi.BranchesInfo.BranchIDCreatorIdxs))),
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
		parentsVecs[i] = vi.Callback.GetHighestBefore(p)
		if parentsVecs[i] == nil {
			return myVecs, fmt.Errorf("processed out of order, parent not found (inconsistent DB), parent=%s", p.String())
		}
	}

	// observed by himself
	myVecs.after.InitWithEvent(meBranchID, e)
	myVecs.before.InitWithEvent(meBranchID, e)

	for _, pVec := range parentsVecs {
		// calculate HighestBefore  Detect forks for a case when parent observes a fork
		myVecs.before.CollectFrom(pVec, idx.Validator(len(vi.BranchesInfo.BranchIDCreatorIdxs)))
	}
	// Detect forks, which were not observed by parents
	if vi.AtLeastOneFork() {
		for n := idx.Validator(0); n < vi.Validators.Len(); n++ {
			if len(vi.BranchesInfo.BranchIDByCreators[n]) <= 1 {
				continue
			}
			for _, branchID := range vi.BranchesInfo.BranchIDByCreators[n] {
				if myVecs.before.IsForkDetected(branchID) {
					// if one branch observes a fork, mark all the branches as observing the fork
					vi.setForkDetected(myVecs.before, n)
					break
				}
			}
		}

	nextCreator:
		for n := idx.Validator(0); n < vi.Validators.Len(); n++ {
			if myVecs.before.IsForkDetected(n) {
				continue
			}
			for _, branchID1 := range vi.BranchesInfo.BranchIDByCreators[n] {
				for _, branchID2 := range vi.BranchesInfo.BranchIDByCreators[n] {
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
		wLowestAfterSeq := vi.Callback.GetLowestAfter(walk)

		// update LowestAfter vector of the old event, because newly-connected event observes it
		if wLowestAfterSeq.Visit(meBranchID, e) {
			vi.Callback.SetLowestAfter(walk, wLowestAfterSeq)
			return true
		}
		return false
	}
	err = vi.DfsSubgraph(e, onWalk)
	if err != nil {
		vi.ErrorHandler(err)
	}

	// store calculated vectors
	vi.Callback.SetHighestBefore(e.ID(), myVecs.before)
	vi.Callback.SetLowestAfter(e.ID(), myVecs.after)
	vi.SetEventBranchID(e.ID(), meBranchID)

	return myVecs, nil
}

func (vi *Engine) GetMergedHighestBefore(id hash.Event) HighestBeforeI {
	vi.InitBranchesInfo()

	if vi.AtLeastOneFork() {
		scatteredBefore := vi.Callback.GetHighestBefore(id)

		mergedBefore := vi.Callback.NewHighestBefore(vi.Validators.Len())

		for creatorIdx, branches := range vi.BranchesInfo.BranchIDByCreators {
			mergedBefore.GatherFrom(idx.Validator(creatorIdx), scatteredBefore, branches)
		}

		return mergedBefore
	}
	return vi.Callback.GetHighestBefore(id)
}
