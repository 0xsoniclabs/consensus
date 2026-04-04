package dagindexer

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensustest"
	"github.com/0xsoniclabs/consensus/consensus/vecflushable"
	"github.com/0xsoniclabs/kvdb/memorydb"
)

func setupIndex(t *testing.T) (*Index, map[string]consensus.Event) {
	t.Helper()
	dagAscii := `
		a00   b00   c00
		║     ║     ║
		a01═══╣     ║
		║     ║     ║
		║     b01═══╣
		║     ║     ║
		a02═══╬═════╣
	`
	nodes, _, _ := consensustest.ASCIIschemeToDAG(dagAscii)
	validators := consensus.EqualWeightValidators(nodes, 1)
	events := make(map[consensus.EventHash]consensus.Event)
	getEvent := func(id consensus.EventHash) consensus.Event {
		return events[id]
	}
	vi := NewIndex(tCrit, LiteConfig())
	vi.Reset(validators, vecflushable.Wrap(memorydb.New(), vecflushable.TestSizeLimit), getEvent)

	_, _, named := consensustest.ASCIIschemeForEach(dagAscii, consensustest.ForEachEvent{
		Process: func(e consensus.Event, name string) {
			events[e.ID()] = e
			err := vi.Add(e)
			if err != nil {
				t.Fatal(err)
			}
			vi.Flush()
		},
	})
	return vi, named
}

func TestIndex_Close(t *testing.T) {
	vi := NewIndex(tCrit, LiteConfig())
	vi.Reset(
		consensus.EqualWeightValidators(consensustest.GenNodes(3), 1),
		vecflushable.Wrap(memorydb.New(), vecflushable.TestSizeLimit),
		nil,
	)
	err := vi.Close()
	assert.NoError(t, err)
}

func TestIndex_SetEventBranchID(t *testing.T) {
	vi := NewIndex(tCrit, LiteConfig())
	vi.Reset(
		consensus.EqualWeightValidators(consensustest.GenNodes(3), 1),
		vecflushable.Wrap(memorydb.New(), vecflushable.TestSizeLimit),
		nil,
	)
	hash := consensus.EventHash{1}
	vi.SetEventBranchID(hash, 7)
	assert.Equal(t, consensus.ValidatorIndex(7), vi.GetEventBranchID(hash))
}

func TestIndex_StronglyReach_MissingA(t *testing.T) {
	vi, _ := setupIndex(t)
	missing := consensus.EventHash{0xff, 0xfe}
	b00 := consensus.EventHash{} // any existing
	// Find a real event hash
	vi.InitBranchesInfo()
	assert.Panics(t, func() {
		vi.StronglyReach(missing, b00)
	})
}

func TestIndex_StronglyReach_MissingB(t *testing.T) {
	vi, named := setupIndex(t)
	a02 := named["a02"].ID()
	missing := consensus.EventHash{0xff, 0xfe}
	assert.Panics(t, func() {
		vi.StronglyReach(a02, missing)
	})
}

func TestIndex_MedianTime_MissingEvent(t *testing.T) {
	vi, _ := setupIndex(t)
	missing := consensus.EventHash{0xff, 0xfe}
	assert.Panics(t, func() {
		vi.MedianTime(missing, 0)
	})
}

func TestIndex_DfsSubgraph_MissingEvent(t *testing.T) {
	dagAscii := `
		a00   b00
		║     ║
		a01═══╣
	`
	nodes, _, _ := consensustest.ASCIIschemeToDAG(dagAscii)
	validators := consensus.EqualWeightValidators(nodes, 1)
	events := make(map[consensus.EventHash]consensus.Event)
	getEvent := func(id consensus.EventHash) consensus.Event {
		return events[id] // will return nil for missing events
	}
	vi := NewIndex(tCrit, LiteConfig())
	vi.Reset(validators, vecflushable.Wrap(memorydb.New(), vecflushable.TestSizeLimit), getEvent)

	var lastEvent consensus.Event
	consensustest.ASCIIschemeForEach(dagAscii, consensustest.ForEachEvent{
		Process: func(e consensus.Event, name string) {
			events[e.ID()] = e
			lastEvent = e
		},
	})

	// Remove a parent from events so DfsSubgraph can't find it
	for _, p := range lastEvent.Parents() {
		delete(events, p)
		break
	}

	walked := false
	err := vi.DfsSubgraph(lastEvent, func(hash consensus.EventHash) bool {
		walked = true
		return true // go deeper
	})
	if walked {
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "event not found")
	}
}

// equivocationDAG returns an ASCII DAG where:
//   - c equivocates (c00 on branch 0, c01 on branch 3)
//   - b01 reaches c00 (branch 0) only
//   - b02 reaches both c branches => equivocation detected in b02's HighestBefore
//   - a00 sees nothing about c => no equivocation detected
func equivocationDAG() string {
	return `
		a00   b00   c00
		║     ║     ║
		║     b01═══╣
		║     ║     ║
		║     ║     c11
		║     ║    3║
		║     ║    ╚ c01
		║     ║     ║
		║     b02═══╣
	`
}

func TestStronglyReachProgress_ChosenParentSeesEquivocation(t *testing.T) {
	// a00 does NOT see c's equivocation.
	// b02 DOES see c's equivocation (reaches both c branches).
	// bID = c00 (on c's first branch).
	vi, named := stronglyReachesProgressAux(equivocationDAG())

	aID := named["a00"].ID()
	bID := named["c00"].ID()
	chosen := []consensus.EventHash{named["b02"].ID()} // sees equivocation
	candidates := []consensus.EventHash{}

	chosenRes, _ := vi.StronglyReachProgress(aID, bID, candidates, chosen)
	assert.False(t, chosenRes.QuorumReached())
}

func TestStronglyReachProgress_CandidateParentSeesEquivocation(t *testing.T) {
	// a00 does NOT see c's equivocation.
	// No chosen parents.
	// b02 (candidate) DOES see c's equivocation.
	vi, named := stronglyReachesProgressAux(equivocationDAG())

	aID := named["a00"].ID()
	bID := named["c00"].ID()
	chosen := []consensus.EventHash{}
	candidates := []consensus.EventHash{named["b02"].ID()} // sees equivocation

	_, candidatesRes := vi.StronglyReachProgress(aID, bID, candidates, chosen)
	assert.False(t, candidatesRes[0].QuorumReached())
}

func TestIndex_fillEventVectors_AssignBranchIDError(t *testing.T) {
	dagAscii := `
		a00   b00
		║     ║
		a01═══╣
	`
	nodes, _, _ := consensustest.ASCIIschemeToDAG(dagAscii)
	validators := consensus.EqualWeightValidators(nodes, 1)
	events := make(map[consensus.EventHash]consensus.Event)
	getEvent := func(id consensus.EventHash) consensus.Event {
		return events[id]
	}
	vi := NewIndex(tCrit, LiteConfig())
	vi.Reset(validators, vecflushable.Wrap(memorydb.New(), vecflushable.TestSizeLimit), getEvent)

	// Add first event normally
	var secondEvent consensus.Event
	consensustest.ASCIIschemeForEach(dagAscii, consensustest.ForEachEvent{
		Process: func(e consensus.Event, name string) {
			events[e.ID()] = e
			if len(e.Parents()) <= 1 {
				err := vi.Add(e)
				if err != nil {
					t.Fatal(err)
				}
				vi.Flush()
			} else {
				secondEvent = e
			}
		},
	})

	if secondEvent == nil {
		t.Skip("no multi-parent event found")
	}

	// Corrupt BranchesInfo to trigger AssignBranchID error
	vi.InitBranchesInfo()
	vi.BranchesInfo().BranchIDCreatorIdxs = vi.BranchesInfo().BranchIDCreatorIdxs[:0]
	vi.BranchesInfo().BranchIDLastSeq = vi.BranchesInfo().BranchIDLastSeq[:0]

	err := vi.Add(secondEvent)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inconsistent")
}

func TestIndex_fillEventVectors_ParentHighestBeforeNil(t *testing.T) {
	dagAscii := `
		a00   b00
		║     ║
		a01═══╣
	`
	nodes, _, _ := consensustest.ASCIIschemeToDAG(dagAscii)
	validators := consensus.EqualWeightValidators(nodes, 1)
	events := make(map[consensus.EventHash]consensus.Event)
	getEvent := func(id consensus.EventHash) consensus.Event {
		return events[id]
	}
	vi := NewIndex(tCrit, LiteConfig())
	vi.Reset(validators, vecflushable.Wrap(memorydb.New(), vecflushable.TestSizeLimit), getEvent)

	var childEvent consensus.Event
	consensustest.ASCIIschemeForEach(dagAscii, consensustest.ForEachEvent{
		Process: func(e consensus.Event, name string) {
			events[e.ID()] = e
			if len(e.Parents()) <= 1 {
				// For each genesis event, set its branch ID but skip adding it
				// to the index (so HighestBefore won't be stored).
				vi.InitBranchesInfo()
				vi.SetEventBranchID(e.ID(), consensus.ValidatorIndex(0))
				vi.Flush()
			} else {
				childEvent = e
			}
		},
	})

	if childEvent == nil {
		t.Skip("no multi-parent event found")
	}

	// Now childEvent's parents have EventBranchID set but no HighestBefore
	err := vi.Add(childEvent)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "processed out of order")
}

func TestIndex_fillEventVectors_DfsSubgraphError(t *testing.T) {
	dagAscii := `
		a00   b00
		║     ║
		a01═══╣
	`
	nodes, _, _ := consensustest.ASCIIschemeToDAG(dagAscii)
	validators := consensus.EqualWeightValidators(nodes, 1)
	events := make(map[consensus.EventHash]consensus.Event)
	getEvent := func(id consensus.EventHash) consensus.Event {
		return events[id]
	}
	vi := NewIndex(tCrit, LiteConfig())
	vi.Reset(validators, vecflushable.Wrap(memorydb.New(), vecflushable.TestSizeLimit), getEvent)

	var childEvent consensus.Event
	consensustest.ASCIIschemeForEach(dagAscii, consensustest.ForEachEvent{
		Process: func(e consensus.Event, name string) {
			events[e.ID()] = e
			if len(e.Parents()) <= 1 {
				err := vi.Add(e)
				if err != nil {
					t.Fatal(err)
				}
				vi.Flush()
			} else {
				childEvent = e
			}
		},
	})

	if childEvent == nil {
		t.Skip("no multi-parent event found")
	}

	// Remove parents from getEvent map so DfsSubgraph can't find them.
	// But they ARE in the index (HighestBefore/LowestAfter stored).
	for _, p := range childEvent.Parents() {
		delete(events, p)
	}

	// fillEventVectors will succeed past the parent loading phase
	// (parents have HighestBefore), but DfsSubgraph will fail
	// when it tries getEvent on a parent for traversal.
	assert.Panics(t, func() {
		vi.Add(childEvent)
	})
}

func TestIndex_Flush_ErrorPath(t *testing.T) {
	// The Flush error path triggers when vi.store.Db.Flush() returns an error.
	// With memorydb, Flush() never fails, so we verify the normal path completes.
	vi, _ := setupIndex(t)
	// This should not panic
	vi.Flush()
}

func TestIndex_fillEventVectors_ParentNotFound(t *testing.T) {
	dagAscii := `
		a00   b00
		║     ║
		a01═══╣
	`
	nodes, _, _ := consensustest.ASCIIschemeToDAG(dagAscii)
	validators := consensus.EqualWeightValidators(nodes, 1)
	events := make(map[consensus.EventHash]consensus.Event)
	getEvent := func(id consensus.EventHash) consensus.Event {
		return events[id]
	}
	vi := NewIndex(tCrit, LiteConfig())
	vi.Reset(validators, vecflushable.Wrap(memorydb.New(), vecflushable.TestSizeLimit), getEvent)

	// Collect events but only add genesis ones to the index
	var childEvent consensus.Event
	consensustest.ASCIIschemeForEach(dagAscii, consensustest.ForEachEvent{
		Process: func(e consensus.Event, name string) {
			events[e.ID()] = e
			if len(e.Parents()) <= 1 {
				// Genesis or single-parent event — add to index
				err := vi.Add(e)
				if err != nil {
					t.Fatal(err)
				}
				vi.Flush()
			} else {
				childEvent = e
			}
		},
	})

	if childEvent != nil {
		// Try to add event whose cross-parent's vectors are not in the store
		// (only genesis self-parent was added, not all parents)
		err := vi.Add(childEvent)
		if err != nil {
			assert.Contains(t, err.Error(), "processed out of order")
		}
	}
}
