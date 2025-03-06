// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package election

import (
	"fmt"
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
		testVoteAndAggregate(t,
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
	`, map[string]string{"a1_1": "a1_1_fork", "b_1_1": "b1_1_fork"})
	})

	t.Run("4 equalWeights", func(t *testing.T) {
		testVoteAndAggregate(t,
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
			`, map[string]string{"c1_1": "c1_1_fork"})
	})

	t.Run("4 equalWeights missingRoot", func(t *testing.T) {
		testVoteAndAggregate(t,
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
		`, map[string]string{})
	})

	t.Run("4 differentWeights", func(t *testing.T) {
		testVoteAndAggregate(t,
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
		`, map[string]string{"a1_1": "a1_1_fork", "d1_1": "d1_1_fork"})
	})

	t.Run("4 differentWeights 4rounds", func(t *testing.T) {
		testVoteAndAggregate(t,
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
	`, map[string]string{"a1_1": "a1_1_fork"})
	})

	t.Run("4 equalWeights notDecided", func(t *testing.T) {
		testVoteAndAggregate(t,
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
	`, map[string]string{"a1_1": "a1_1_fork", "d1_1": "d1_1_fork"})
	})

}

type slot struct {
	frame       idx.Frame
	validatorID idx.ValidatorID
}

type testState struct {
	ordered    tdag.TestEvents
	frameRoots map[idx.Frame][]RootContext
	vertices   map[hash.Event]slot
	edges      map[fakeEdge]bool
}

func testVoteAndAggregate(
	t *testing.T,
	expected *testExpected,
	weights weights,
	dagAscii string,
	forks map[string]string,
) {
	t.Helper()
	assertar := assert.New(t)

	state := testState{
		ordered:    make(tdag.TestEvents, 0),
		frameRoots: make(map[idx.Frame][]RootContext),
		vertices:   make(map[hash.Event]slot),
		edges:      make(map[fakeEdge]bool),
	}

	nodes, _, _ := tdag.ASCIIschemeForEach(dagAscii, tdag.ForEachEvent{
		Process: func(_root dag.Event, name string) {
			root := _root.(*tdag.TestEvent)
			indexTestEvent(&state, root, false)
			if forkedRootName, ok := forks[name]; ok {
				forkedRoot := *root
				forkedRoot.Name = forkedRootName
				forkedRoot.SetID(tdag.CalcHashForTestEvent(&forkedRoot))
				indexTestEvent(&state, &forkedRoot, true)
			}
		},
	})

	validatorsBuilder := pos.NewBuilder()
	for _, node := range nodes {
		nodeName := hash.GetNodeName(node)
		if len(nodeName) == 0 {
			nodeName = fmt.Sprintf("%d", node)
		}
		validatorsBuilder.Set(node, weights[nodeName])
	}
	validators := validatorsBuilder.Build()

	forklessCauseFn := func(a hash.Event, b hash.Event) bool {
		edge := fakeEdge{
			from: a,
			to:   b,
		}
		return state.edges[edge]
	}
	getFrameRootsFn := func(f idx.Frame) []RootContext {
		return state.frameRoots[f]
	}

	// re-order events randomly, preserving parents order
	unordered := make(tdag.TestEvents, len(state.ordered))
	for i, j := range rand.Perm(len(state.ordered)) {
		unordered[i] = state.ordered[j]
	}
	state.ordered = unordered.ByParents()

	el := New(1, validators, forklessCauseFn, getFrameRootsFn)

	// processing:
	for _, root := range state.ordered {
		rootHash := root.ID()
		rootSlot, ok := state.vertices[rootHash]
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

func indexTestEvent(state *testState, root *tdag.TestEvent, isFork bool) {
	state.ordered = append(state.ordered, root)
	slt := slot{
		frame:       frameOf(root.Name),
		validatorID: root.Creator(),
	}
	state.vertices[root.ID()] = slt
	hsh := root.ID()
	state.frameRoots[frameOf(root.Name)] = append(
		state.frameRoots[frameOf(root.Name)],
		RootContext{
			RootHash:    hsh,
			ValidatorID: slt.validatorID,
		},
	)
	if !isFork {
		noPrev := false
		if strings.HasPrefix(root.Name, "+") {
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
			state.edges[edge] = true
		}
	} else {
		selfParent := root.SelfParent()
		if selfParent != nil {
			root.SetParents(hash.Events{*selfParent})
		} else {
			root.SetParents(hash.Events{})
		}
	}
}
