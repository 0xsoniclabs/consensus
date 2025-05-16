package dagindexer

import (
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensustest"
	"github.com/0xsoniclabs/consensus/consensus/vecflushable"
	"github.com/0xsoniclabs/kvdb/memorydb"
	"github.com/stretchr/testify/assert"
)

func createScenario(dagAscii string) (*Index, map[string]consensus.Event) {
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
				panic(err)
			}
			vi.Flush()
		},
	})

	return vi, named
}

// No forks at all.
func TestNoCheatersTrivial(t *testing.T) {
	dagAscii := `
		a00           b00            c00
		║             ║              ║
		║             ║              ║
		║             ╠              c01
		║             ║              ║
		║             ║              ║
		║             ║              ║
		║             ║              ║
		║             ╠              ║
		║             ║              ║
		║             ║              c02
		║             ║              ║
		║             ║              ║
		║             ║              ║
		╠             b01            ╣
		║             ║              ║
		║             ║              ║
		║             ║              ║
		║             ║              ║
		a01           ╣              ║
		║             ║              ║
		║             ║              ║
		║             ║              ║
		║             ║              ║ 
		║             ║              ║
	`

	t.Helper()
	assertar := assert.New(t)

	vi, named := createScenario(dagAscii)

	a01 := named["a01"].ID()
	initialList := consensus.EventHashes{named["b00"].ID(), named["b01"].ID()}
	filteredList := vi.NoCheaters(&a01, initialList)
	assertar.Equal(initialList, filteredList)
	filteredList = vi.NoCheaters(nil, initialList)
	assertar.Equal(initialList, filteredList)
}

func TestNoCheatersStandard(t *testing.T) {
	dagAscii := `
b00
		║       ║
		║       c00
		║       ║
		b01═════╣
		║       ║
		╠══════ c02
		║       ║
		b02═════╣
		║       ║
		╠══════ c04
		║       ║       ║
		║       ║       a00
		║3      ║       ║
		║╚═════─╫─═════ a01
		║      3║       ║
		║      ╚ c01════╣ // equivocation
		║║      ║       ║
		║╚══════╬══════ a02
		║      3║       ║
		║      ╚ c03════╣ // equivocation
		║       ║       ║
		╠═══════╬══════ a03
	`

	t.Helper()
	assertar := assert.New(t)

	vi, named := createScenario(dagAscii)

	a03 := named["a03"].ID()
	initialList := consensus.EventHashes{named["c02"].ID(), named["c03"].ID()}
	filteredList := vi.NoCheaters(&a03, initialList)
	assertar.Equal(consensus.EventHashes{}, filteredList)
}
