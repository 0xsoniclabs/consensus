// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensustest

import (
	"math/rand/v2"
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"
)

func TestEventsByParents(t *testing.T) {
	nodes := GenNodes(5)
	events := GenRandEvents(nodes, 10, 3, nil)
	var ee consensus.Events
	for _, e := range events {
		ee = append(ee, e...)
	}
	// shuffle
	unordered := make(consensus.Events, len(ee))
	for i, j := range rand.Perm(len(ee)) {
		unordered[i] = ee[j]
	}

	ordered := ByParents(unordered)
	position := make(map[consensus.EventHash]int)
	for i, e := range ordered {
		position[e.ID()] = i
	}

	for i, e := range ordered {
		for _, p := range e.Parents() {
			pos, ok := position[p]
			if !ok {
				continue
			}
			if pos > i {
				t.Fatalf("parent %s is not before %s", p.String(), e.ID().String())
				return
			}
		}
	}
}

func TestTestEventsByParents(t *testing.T) {
	nodes := GenNodes(5)
	events := GenRandEvents(nodes, 10, 3, nil)

	var testEvents TestEvents
	for _, e := range events {
		for _, event := range e {
			testEvents = append(testEvents, event.(*TestEvent))
		}
	}

	// shuffle
	unordered := make(TestEvents, len(testEvents))
	for i, j := range rand.Perm(len(testEvents)) {
		unordered[i] = testEvents[j]
	}

	// order the events using TestEvents.ByParents()
	ordered := unordered.ByParents()

	// validate the ordering
	position := make(map[consensus.EventHash]int)
	for i, e := range ordered {
		position[e.ID()] = i
	}

	for i, e := range ordered {
		for _, p := range e.Parents() {
			pos, ok := position[p]
			if !ok {
				continue
			}
			if pos > i {
				t.Fatalf("parent %s is not before %s", p.String(), e.ID().String())
				return
			}
		}
	}

	if len(ordered) != len(unordered) {
		t.Fatalf("expected %d events, got %d", len(unordered), len(ordered))
	}
}
