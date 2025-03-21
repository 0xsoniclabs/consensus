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
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"
)

func TestCalFrameIdx_10000(t *testing.T) {
	testCalcFrameIdx(t, 10000)
}

// testCalcFrameIdx verifies that lagging validator calculates correct frame numbers after a (large) pause
func testCalcFrameIdx(t *testing.T, gap int) {
	nodes := consensus.GenNodes(2)
	// Give one validator quorum power to advance the frames on it's own
	lch, _, store, _ := NewCoreLachesis(nodes, []consensus.Weight{1, 3})

	laggyGenesis := processTestEvent(t, lch, store, nodes[0], 1, consensus.EventHashes{})
	parentEvent := processTestEvent(t, lch, store, nodes[1], 1, consensus.EventHashes{})
	for i := 0; i < gap; i++ {
		parentEvent = processTestEvent(t, lch, store, nodes[1], consensus.Seq(parentEvent.Seq()+1), consensus.EventHashes{parentEvent.ID()})
	}
	// Lagging validator creates an event after a frame gap
	finalEvent := processTestEvent(t, lch, store, nodes[0], laggyGenesis.Seq()+1, consensus.EventHashes{laggyGenesis.ID(), parentEvent.ID()})

	if want, got := laggyGenesis.Frame()+consensus.Frame(gap)+1, finalEvent.Frame(); want != got {
		t.Errorf("expected calculated frame number of lagging validator to be: %d, got: %d", gap, finalEvent.Frame())
	}
}

var maxLamport consensus.Lamport = 0

// processTestEvent builds and pipes the event through main Lacehsis' DAG manipulation pipeline
func processTestEvent(t *testing.T, lch *CoreLachesis, store *EventStore, validatorId consensus.ValidatorID, seq consensus.Seq, parents consensus.EventHashes) *consensus.TestEvent {
	event := &consensus.TestEvent{}
	event.SetSeq(seq)
	event.SetCreator(validatorId)
	event.SetParents(parents)
	maxLamport = maxLamport + 1
	event.SetLamport(maxLamport)
	event.SetEpoch(lch.store.GetEpoch())
	if err := lch.Build(event); err != nil {
		t.Errorf("error while building event for validator: %d, seq: %d, err: %v", validatorId, seq, err)
	}
	// default sample hash assigned through Build is enough
	store.SetEvent(event)
	if err := lch.Process(event); err != nil {
		t.Errorf("error while processing event for validator: %d, seq: %d, err: %v", validatorId, seq, err)
	}
	return event
}
