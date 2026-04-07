// Copyright (c) 2026 Sonic Operations Ltd
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
	"github.com/0xsoniclabs/consensus/consensus/consensusstore"
	"github.com/0xsoniclabs/consensus/consensus/consensustest"
)

func TestBootstrap_AlreadyBootstrapped(t *testing.T) {
	nodes := consensustest.GenNodes(3)
	lachesis, _, _, _ := NewCoreConsensus(nodes, []consensus.Weight{1, 1, 1})
	consensusCallbacks := consensus.ConsensusCallbacks{}
	if err := lachesis.Bootstrap(consensusCallbacks); err != nil {
		t.Fatalf("unexpected error on bootstrapping: %v", err)
	}
	if err := lachesis.Bootstrap(consensusCallbacks); err != ErrAlreadyBootstrapped {
		t.Fatalf("expected an `already bootstrapped` error but recieved: %v", err)
	}
}

func TestBootstrap_NoNewBases(t *testing.T) {
	testBootstrap_ReprocessBases(t, 10, 0, 10)
}
func TestBootstrap_NoCertifiedFramesNoSealing(t *testing.T) {
	testBootstrap_ReprocessBases(t, 0, 0, 10)
}
func TestBootstrap_NoCertifiedFramesSealing(t *testing.T) {
	testBootstrap_ReprocessBases(t, 0, 4, 10)
}
func TestBootstrap_CertifiedFramesNoSealing(t *testing.T) {
	testBootstrap_ReprocessBases(t, 3, 0, 10)
}
func TestBootstrap_CertifiedFramesSealing(t *testing.T) {
	testBootstrap_ReprocessBases(t, 2, 6, 10)
}

// bootstrapping can be triggered on a mid-epoch DB checkpoint (due to a crash for example)
// testBootstrap_ReprocessBases tests for a correct starting and ending point of the election bootstrap process
// by varying last checkpoint's last certified Frame, future sealing Frame and number of frame bases
// available to be run through the election
func testBootstrap_ReprocessBases(t *testing.T, lastCertifiedFrame, sealingFrame, numFrames consensus.Frame) {
	nodes := consensustest.GenNodes(1)
	engine, _, eventSource, _ := NewCoreConsensus(nodes, []consensus.Weight{1})
	engine.store.SetLastCertifiedState(&consensusstore.LastCertifiedState{LastCertifiedFrame: lastCertifiedFrame})
	numLeadersDelivered := consensus.Frame(0)
	if err := engine.Bootstrap(consensus.ConsensusCallbacks{
		BeginBlock: func(block *consensus.Block) consensus.BlockCallbacks {
			return consensus.BlockCallbacks{
				EndBlock: func() (sealEpoch *consensus.Validators) {
					numLeadersDelivered++
					if currentFrame := lastCertifiedFrame + numLeadersDelivered; currentFrame == sealingFrame {
						return engine.election.validators
					}
					return nil
				},
			}
		},
	}); err != nil {
		t.Fatal(err)
	}
	bases := make([]*consensustest.TestEvent, numFrames)
	bases[0] = prepareTestBase(t, engine, eventSource, 0, nodes[0], consensus.EventHashes{})
	for i := 1; i < len(bases); i++ {
		bases[i] = prepareTestBase(t, engine, eventSource, i, nodes[0], consensus.EventHashes{bases[i-1].ID()})
	}
	if err := engine.bootstrapElection(); err != nil {
		t.Fatal(err)
	}

	// scenario 1 - not enough frames to deliver anything (at least 2 above are necessary) i.e. numFrames < lastCertifiedFrame + 2
	expectedNumLeadersDelivered := consensus.Frame(0)
	if numFrames >= lastCertifiedFrame+2 {
		// scenario 2 - enough frames to deliver and sealingFrame value (!= 0) provided
		if sealingFrame != 0 {
			expectedNumLeadersDelivered = sealingFrame - lastCertifiedFrame
		} else {
			// scenario 3 - enough frames to deliver and no sealingFrame value provided
			// implies that all frames with 2+ frames above will recieve their leaders
			// offset the expected number by -2 as last two frames don't have enough frames above to make a certification for them
			expectedNumLeadersDelivered = numFrames - 2 - lastCertifiedFrame
		}
	}
	if expectedNumLeadersDelivered != numLeadersDelivered {
		t.Fatalf("unexpected number of leaders delivered, expected: %d, got: %d", expectedNumLeadersDelivered, numLeadersDelivered)
	}
}

// prepareTestBase creates, indexes and persists frame bases
// we omit base elections to simulate a mid-epoch bootstrap scenario
func prepareTestBase(
	t *testing.T,
	lachesis *IndexedLachesis,
	eventSource *consensustest.TestEventSource,
	enumeration int,
	validatorID consensus.ValidatorID,
	parents consensus.EventHashes,
) *consensustest.TestEvent {
	base := &consensustest.TestEvent{}
	base.SetCreator(validatorID)
	base.SetID([24]byte{byte(enumeration)})
	base.SetFrame(consensus.Frame(enumeration + 1))
	base.SetSeq(consensus.Seq(enumeration + 1))
	base.SetParents(parents)
	eventSource.SetEvent(base)
	if err := lachesis.DagIndexer.Add(base); err != nil {
		t.Fatal(err)
	}
	lachesis.store.AddBase(base)
	return base
}
