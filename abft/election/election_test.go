package election

import (
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/dag"
	"github.com/0xsoniclabs/consensus/inter/dag/tdag"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/inter/pos"
	"github.com/0xsoniclabs/consensus/utils"
)

type fakeEdge struct {
	from hash.Event
	to   hash.Event
}

type (
	weights map[string]pos.Weight
)

type testExpected struct {
	DecidedFrame   idx.Frame
	DecidedAtropos string
	DecisiveRoots  map[string]bool
}

func TestProcessRoot(t *testing.T) {

	t.Run("4 equalWeights notDecided", func(t *testing.T) {
		testProcessRoot(t,
			nil,
			weights{
				"nodeA": 2,
				"nodeB": 1,
				"nodeC": 1,
				"nodeD": 1,
			}, `
	a1_1  b1_1  c1_1  d1_1
	║     ║     ║     ║
	a2_2══╬═════╣     ║
	║     ║     ║     ║
	║╚════b2_2══╣     ║
	║     ║     ║     ║
	║     ║╚════c2_2══╣
	║     ║     ║     ║
	║     ║╚═══─╫╩════d2_2
	║     ║     ║     ║
	a3_3══╬═════╬═════╣
	║     ║     ║     ║
	`)
	})

	t.Run("4 equalWeights", func(t *testing.T) {
		testProcessRoot(t,
			&testExpected{
				DecidedFrame:   1,
				DecidedAtropos: "c1_1",
				DecisiveRoots:  map[string]bool{"a3_3": true},
			},
			weights{
				"nodeA": 1,
				"nodeB": 1,
				"nodeC": 1,
				"nodeD": 1,
			}, `
			a1_1  b1_1  c1_1  d1_1
			║     ║     ║     ║
			a2_2══╬═════╣     ║
			║     ║     ║     ║
			║     b2_2══╬═════╣
			║     ║     ║     ║
			║     ║╚════c2_2══╣
			║     ║     ║     ║
			║     ║╚═══─╫╩════d2_2
			║     ║     ║     ║
			a3_3══╬═════╬═════╣
			║     ║     ║     ║
			`)
	})

	t.Run("4 equalWeights missingRoot", func(t *testing.T) {
		testProcessRoot(t,
			&testExpected{
				DecidedFrame:   1,
				DecidedAtropos: "c1_1",
				DecisiveRoots:  map[string]bool{"a3_3": true},
			},
			weights{
				"nodeA": 1,
				"nodeB": 1,
				"nodeC": 1,
				"nodeD": 1,
			}, `
		a1_1  b1_1  c1_1  d1_1
		║     ║     ║     ║
		a2_2══╬═════╣     ║
		║     ║     ║     ║
		║╚════b2_2══╣     ║
		║     ║     ║     ║
		║╚═══─╫╩════c2_2  ║
		║     ║     ║     ║
		a3_3══╬═════╣     ║
		║     ║     ║     ║
		`)
	})

	t.Run("4 differentWeights", func(t *testing.T) {
		testProcessRoot(t,
			&testExpected{
				DecidedFrame:   1,
				DecidedAtropos: "a1_1",
				DecisiveRoots:  map[string]bool{"b3_3": true},
			},
			weights{
				"nodeA": math.MaxUint32/2 - 3,
				"nodeB": 1,
				"nodeC": 1,
				"nodeD": 1,
			}, `
		a1_1  b1_1  c1_1  d1_1
		║     ║     ║     ║
		a2_2══╬═════╣     ║
		║     ║     ║     ║
		║╚════+b2_2 ║     ║
		║     ║     ║     ║
		║╚═══─╫─════+c2_2 ║
		║     ║     ║     ║
		║╚═══─╫╩═══─╫╩════d2_2
		║     ║     ║     ║
		╠═════b3_3══╬═════╣
		║     ║     ║     ║
		`)
	})

	t.Run("4 differentWeights 4rounds", func(t *testing.T) {
		testProcessRoot(t,
			&testExpected{
				DecidedFrame:   1,
				DecidedAtropos: "a1_1",
				DecisiveRoots:  map[string]bool{"c3_3": true, "b3_3": true},
			},
			weights{
				"nodeA": 4,
				"nodeB": 2,
				"nodeC": 1,
				"nodeD": 1,
			}, `
	a1_1  b1_1  c1_1  d1_1
	║     ║     ║     ║
	a2_2══╣     ║     ║
	║     ║     ║     ║
	║     +b2_2═╬═════╣
	║     ║     ║     ║
	║╚═══─╫─════c2_2══╣
	║     ║     ║     ║
	║╚═══─╫─═══─╫╩════d2_2
	║     ║     ║     ║
	a3_3  ╣     ║     ║
	║     ║     ║     ║
	║╚════b3_3══╬═════╣
	║     ║     ║     ║
	║╚═══─╫╩════c3_3══╣
	║     ║     ║     ║
	║╚═══─╫╩═══─╫─════+d3_3
	`)
	})

}

type slot struct {
	frame       idx.Frame
	validatorID idx.ValidatorID
}

func testProcessRoot(
	t *testing.T,
	expected *testExpected,
	weights weights,
	dagAscii string,
) {
	t.Helper()
	assertar := assert.New(t)

	// events:
	ordered := make(tdag.TestEvents, 0)
	frameRoots := make(map[idx.Frame][]RootContext)
	vertices := make(map[hash.Event]slot)
	edges := make(map[fakeEdge]bool)

	nodes, _, _ := tdag.ASCIIschemeForEach(dagAscii, tdag.ForEachEvent{
		Process: func(_root dag.Event, name string) {
			root := _root.(*tdag.TestEvent)
			// store all the events
			ordered = append(ordered, root)

			slot := slot{
				frame:       frameOf(name),
				validatorID: root.Creator(),
			}
			vertices[root.ID()] = slot

			hsh := root.ID()
			frameRoots[frameOf(name)] = append(
				frameRoots[frameOf(name)],
				RootContext{
					RootHash:    hsh,
					ValidatorID: slot.validatorID,
				},
			)

			// build edges to be able to fake forkless cause fn
			noPrev := false
			if strings.HasPrefix(name, "+") {
				noPrev = true
			}
			from := root.ID()
			for _, observed := range root.Parents() {
				if root.IsSelfParent(observed) && noPrev {
					continue
				}
				to := observed
				edge := fakeEdge{
					from: from,
					to:   to,
				}
				edges[edge] = true
			}
		},
	})

	validatorsBuilder := pos.NewBuilder()
	for _, node := range nodes {
		validatorsBuilder.Set(node, weights[utils.NameOf(node)])
	}
	validators := validatorsBuilder.Build()

	forklessCauseFn := func(a hash.Event, b hash.Event) bool {
		edge := fakeEdge{
			from: a,
			to:   b,
		}
		return edges[edge]
	}
	getFrameRootsFn := func(f idx.Frame) []RootContext {
		return frameRoots[f]
	}

	// re-order events randomly, preserving parents order
	unordered := make(tdag.TestEvents, len(ordered))
	for i, j := range rand.Perm(len(ordered)) {
		unordered[i] = ordered[j]
	}
	ordered = unordered.ByParents()

	el := New(1, validators, forklessCauseFn, getFrameRootsFn)

	// processing:
	for _, root := range ordered {
		rootHash := root.ID()
		rootSlot, ok := vertices[rootHash]
		if !ok {
			t.Fatal("inconsistent vertices")
		}
		atropoi, err := el.VoteAndAggregate(rootSlot.frame, rootSlot.validatorID, rootHash)
		if err != nil {
			t.Fatal(err)
		}

		// checking:
		decisive := expected != nil && expected.DecisiveRoots[root.ID().String()]
		if decisive {
			assertar.NotNil(atropoi)
			assertar.NotEmpty(atropoi)
			assertar.Equal(expected.DecidedFrame, atropoi[0].Frame)
			assertar.Equal(expected.DecidedAtropos, atropoi[0].AtroposHash.String())
			return
		} else {
			assertar.Empty(atropoi)
		}
	}
}

func frameOf(dsc string) idx.Frame {
	s := strings.Split(dsc, "_")[1]
	h, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		panic(err)
	}
	return idx.Frame(h)
}
