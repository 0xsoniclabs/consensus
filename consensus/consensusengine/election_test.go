// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensusengine

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensusstore"
	"github.com/0xsoniclabs/consensus/consensus/consensustest"
)

type fakeEdge struct {
	from consensus.EventHash
	to   consensus.EventHash
}

type (
	weights map[string]consensus.Weight
)

type testExpected struct {
	CertifiedFrame  consensus.Frame
	CertifiedLeader string
	CertifyingBases map[string]bool
}

func TestProcessBase(t *testing.T) {
	t.Run("4 equalWeights notCertified", func(t *testing.T) {
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
	`, map[string]string{"a1_1": "a1_1_equivocation", "b_1_1": "b1_1_equivocation"})
	})

	t.Run("4 equalWeights", func(t *testing.T) {
		testVoteAndAggregate(t,
			&testExpected{
				CertifiedFrame:  1,
				CertifiedLeader: "c1_1",
				CertifyingBases: map[string]bool{"a3_3": true},
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
			`, map[string]string{"c1_1": "c1_1_equivocation"})
	})

	t.Run("4 equalWeights missingBase", func(t *testing.T) {
		testVoteAndAggregate(t,
			&testExpected{
				CertifiedFrame:  1,
				CertifiedLeader: "c1_1",
				CertifyingBases: map[string]bool{"a3_3": true},
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
				CertifiedFrame:  1,
				CertifiedLeader: "a1_1",
				CertifyingBases: map[string]bool{"b3_3": true},
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
		`, map[string]string{"a1_1": "a1_1_equivocation", "d1_1": "d1_1_equivocation"})
	})

	t.Run("4 differentWeights 4rounds", func(t *testing.T) {
		testVoteAndAggregate(t,
			&testExpected{
				CertifiedFrame:  1,
				CertifiedLeader: "a1_1",
				CertifyingBases: map[string]bool{"c3_3": true, "b3_3": true},
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
	`, map[string]string{"a1_1": "a1_1_equivocation"})
	})

	t.Run("4 equalWeights notCertified", func(t *testing.T) {
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
	`, map[string]string{"a1_1": "a1_1_equivocation", "d1_1": "d1_1_equivocation"})
	})

}

type slot struct {
	frame       consensus.Frame
	validatorID consensus.ValidatorID
}

type testState struct {
	ordered    consensustest.TestEvents
	frameBases map[consensus.Frame][]consensusstore.BaseDescriptor
	vertices   map[consensus.EventHash]slot
	edges      map[fakeEdge]bool
}

func testVoteAndAggregate(
	t *testing.T,
	expected *testExpected,
	weights weights,
	dagAscii string,
	equivocations map[string]string,
) {
	t.Helper()
	assertar := assert.New(t)

	state := testState{
		ordered:    make(consensustest.TestEvents, 0),
		frameBases: make(map[consensus.Frame][]consensusstore.BaseDescriptor),
		vertices:   make(map[consensus.EventHash]slot),
		edges:      make(map[fakeEdge]bool),
	}

	nodes, _, _ := consensustest.ASCIIschemeForEach(dagAscii, consensustest.ForEachEvent{
		Process: func(_base consensus.Event, name string) {
			base := _base.(*consensustest.TestEvent)
			indexTestEvent(&state, base, false)
			if equivocatedBaseName, ok := equivocations[name]; ok {
				equivocatedBase := *base
				equivocatedBase.Name = equivocatedBaseName
				equivocatedBase.SetID(consensustest.CalcHashForTestEvent(&equivocatedBase))
				indexTestEvent(&state, &equivocatedBase, true)
			}
		},
	})

	validatorsBuilder := consensus.NewValidatorsBuilder()
	for _, node := range nodes {
		nodeName := consensus.GetNodeName(node)
		if len(nodeName) == 0 {
			nodeName = fmt.Sprintf("%d", node)
		}
		validatorsBuilder.Set(node, weights[nodeName])
	}
	validators := validatorsBuilder.Build()

	stronglyReachFn := func(a consensus.EventHash, b consensus.EventHash) bool {
		edge := fakeEdge{
			from: a,
			to:   b,
		}
		return state.edges[edge]
	}
	getFrameBasesFn := func(f consensus.Frame) []consensusstore.BaseDescriptor {
		return state.frameBases[f]
	}

	// re-order events randomly, preserving parents order
	unordered := make(consensustest.TestEvents, len(state.ordered))
	for i, j := range rand.Perm(len(state.ordered)) {
		unordered[i] = state.ordered[j]
	}
	state.ordered = unordered.ByParents()

	election := NewElection(consensus.FirstFrame, validators, stronglyReachFn, getFrameBasesFn)

	// processing:
	for _, base := range state.ordered {
		baseHash := base.ID()
		baseSlot, ok := state.vertices[baseHash]
		if !ok {
			t.Fatal("inconsistent vertices")
		}
		leaders, err := election.VoteAndAggregate(baseSlot.frame, baseSlot.validatorID, baseHash)
		if err != nil {
			t.Fatal(err)
		}

		// checking:
		certifying := expected != nil && expected.CertifyingBases[base.ID().String()]
		if certifying {
			assertar.NotNil(leaders)
			assertar.NotEmpty(leaders)
			assertar.Equal(expected.CertifiedFrame, leaders[0].Frame)
			assertar.Equal(expected.CertifiedLeader, leaders[0].LeaderHash.String())
			return
		} else {
			assertar.Empty(leaders)
		}
	}
}

func frameOf(dsc string) consensus.Frame {
	s := strings.Split(dsc, "_")[1]
	h, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		panic(err)
	}
	return consensus.Frame(h)
}

func indexTestEvent(state *testState, base *consensustest.TestEvent, isEquivocation bool) {
	state.ordered = append(state.ordered, base)
	slt := slot{
		frame:       frameOf(base.Name),
		validatorID: base.Creator(),
	}
	state.vertices[base.ID()] = slt
	hsh := base.ID()
	state.frameBases[frameOf(base.Name)] = append(
		state.frameBases[frameOf(base.Name)],
		consensusstore.BaseDescriptor{
			BaseHash:    hsh,
			ValidatorID: slt.validatorID,
		},
	)
	if !isEquivocation {
		noPrev := false
		if strings.HasPrefix(base.Name, "+") {
			noPrev = true
		}
		from := base.ID()
		for _, reachable := range base.Parents() {
			if base.IsSelfParent(reachable) && noPrev {
				continue
			}
			to := reachable
			edge := fakeEdge{
				from: from,
				to:   to,
			}
			state.edges[edge] = true
		}
	} else {
		selfParent := base.SelfParent()
		if selfParent != nil {
			base.SetParents(consensus.EventHashes{*selfParent})
		} else {
			base.SetParents(consensus.EventHashes{})
		}
	}
}
