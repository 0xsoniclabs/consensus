package dagindexer

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensustest"
	"github.com/0xsoniclabs/kvdb/memorydb"
)

func TestMedianTimeOnIndex(t *testing.T) {
	nodes := consensustest.GenNodes(5)
	weights := []consensus.Weight{5, 4, 3, 2, 1}
	validators := consensus.ArrayToValidators(nodes, weights)

	vi := NewIndex(func(err error) { panic(err) }, LiteConfig())
	vi.Reset(validators, vi.WrapWithFlushable(memorydb.New()), nil)

	assertar := assert.New(t)
	{ // seq=0
		e := consensus.ZeroEventHash
		// validator indexes are sorted by weight amount
		before := NewHighestBefore(consensus.ValidatorIndex(validators.Len()))

		before.VSeq.Set(0, BranchSeq{Seq: 0})
		before.VTime.Set(0, 100)

		before.VSeq.Set(1, BranchSeq{Seq: 0})
		before.VTime.Set(1, 100)

		before.VSeq.Set(2, BranchSeq{Seq: 1})
		before.VTime.Set(2, 10)

		before.VSeq.Set(3, BranchSeq{Seq: 1})
		before.VTime.Set(3, 10)

		before.VSeq.Set(4, BranchSeq{Seq: 1})
		before.VTime.Set(4, 10)

		vi.SetHighestBefore(e, before)
		assertar.Equal(Timestamp(1), vi.MedianTime(e, 1))
	}

	{ // fork seen = true
		e := consensus.ZeroEventHash
		// validator indexes are sorted by weight amount
		before := NewHighestBefore(consensus.ValidatorIndex(validators.Len()))

		before.SetForkDetected(0)
		before.VTime.Set(0, 100)

		before.SetForkDetected(1)
		before.VTime.Set(1, 100)

		before.VSeq.Set(2, BranchSeq{Seq: 1})
		before.VTime.Set(2, 10)

		before.VSeq.Set(3, BranchSeq{Seq: 1})
		before.VTime.Set(3, 10)

		before.VSeq.Set(4, BranchSeq{Seq: 1})
		before.VTime.Set(4, 10)

		vi.SetHighestBefore(e, before)
		assertar.Equal(Timestamp(10), vi.MedianTime(e, 1))
	}

	{ // normal
		e := consensus.ZeroEventHash
		// validator indexes are sorted by weight amount
		before := NewHighestBefore(consensus.ValidatorIndex(validators.Len()))

		before.VSeq.Set(0, BranchSeq{Seq: 1})
		before.VTime.Set(0, 11)

		before.VSeq.Set(1, BranchSeq{Seq: 2})
		before.VTime.Set(1, 12)

		before.VSeq.Set(2, BranchSeq{Seq: 2})
		before.VTime.Set(2, 13)

		before.VSeq.Set(3, BranchSeq{Seq: 3})
		before.VTime.Set(3, 14)

		before.VSeq.Set(4, BranchSeq{Seq: 4})
		before.VTime.Set(4, 15)

		vi.SetHighestBefore(e, before)
		assertar.Equal(Timestamp(12), vi.MedianTime(e, 1))
	}

}

func TestMedianTimeOnDAG(t *testing.T) {
	dagAscii := `
 ║
 nodeA001
 ║
 nodeA012
 ║            ║
 ║            nodeB001
 ║            ║            ║
 ║            ╠═══════════ nodeC001
 ║║           ║            ║            ║
 ║╚══════════─╫─══════════─╫─══════════ nodeD001
║║            ║            ║            ║
╚ nodeA002════╬════════════╬════════════╣
 ║║           ║            ║            ║
 ║╚══════════─╫─══════════─╫─══════════ nodeD002
 ║            ║            ║            ║
 nodeA003════─╫─══════════─╫─═══════════╣
 ║            ║            ║
 ╠════════════nodeB002     ║
 ║            ║            ║
 ╠════════════╫═══════════ nodeC002
`

	weights := []consensus.Weight{3, 4, 2, 1}
	genesisTime := Timestamp(1)
	creationTimes := map[string]Timestamp{
		"nodeA001": Timestamp(111),
		"nodeB001": Timestamp(112),
		"nodeC001": Timestamp(13),
		"nodeD001": Timestamp(14),
		"nodeA002": Timestamp(120),
		"nodeD002": Timestamp(20),
		"nodeA012": Timestamp(120),
		"nodeA003": Timestamp(20),
		"nodeB002": Timestamp(20),
		"nodeC002": Timestamp(35),
	}
	medianTimes := map[string]Timestamp{
		"nodeA001": genesisTime,
		"nodeB001": genesisTime,
		"nodeC001": Timestamp(13),
		"nodeD001": genesisTime,
		"nodeA002": Timestamp(112),
		"nodeD002": genesisTime,
		"nodeA012": genesisTime,
		"nodeA003": Timestamp(20),
		"nodeB002": Timestamp(20),
		"nodeC002": Timestamp(35),
	}
	t.Run("testMedianTimeOnDAG", func(t *testing.T) {
		testMedianTime(t, dagAscii, weights, creationTimes, medianTimes, genesisTime)
	})
}

func testMedianTime(t *testing.T, dagAscii string, weights []consensus.Weight, creationTimes map[string]Timestamp, medianTimes map[string]Timestamp, genesis Timestamp) {
	assertar := assert.New(t)

	var ordered consensus.Events
	nodes, _, named := consensustest.ASCIIschemeForEach(dagAscii, consensustest.ForEachEvent{
		Process: func(e consensus.Event, name string) {
			ordered = append(ordered, &eventWithCreationTime{e, creationTimes[name]})
		},
	})

	validators := consensus.ArrayToValidators(nodes, weights)

	events := make(map[consensus.EventHash]consensus.Event)
	getEvent := func(id consensus.EventHash) consensus.Event {
		return events[id]
	}

	vi := NewIndex(func(err error) { panic(err) }, LiteConfig())
	vi.Reset(validators, vi.WrapWithFlushable(memorydb.New()), getEvent)

	// push
	for _, e := range ordered {
		events[e.ID()] = e
		assertar.NoError(vi.Add(e))
		vi.Flush()
	}

	// check
	for name, e := range named {
		expected, ok := medianTimes[name]
		if !ok {
			continue
		}
		assertar.Equal(expected, vi.MedianTime(e.ID(), genesis), name)
	}
}
