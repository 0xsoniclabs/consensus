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

func TestBootstrap_NoNewRoots(t *testing.T) {
	testBootstrap_ReprocessRoots(t, 10, 0, 10)
}
func TestBootstrap_NoDecidedFramesNoSealing(t *testing.T) {
	testBootstrap_ReprocessRoots(t, 0, 0, 10)
}
func TestBootstrap_NoDecidedFramesSealing(t *testing.T) {
	testBootstrap_ReprocessRoots(t, 0, 4, 10)
}
func TestBootstrap_DecidedFramesNoSealing(t *testing.T) {
	testBootstrap_ReprocessRoots(t, 3, 0, 10)
}
func TestBootstrap_DecidedFramesSealing(t *testing.T) {
	testBootstrap_ReprocessRoots(t, 2, 6, 10)
}

// bootstrapping can be triggered on a mid-epoch DB checkpoint (due to a crash for example)
// testBootstrap_ReprocessRoots tests for a correct starting and ending point of the election bootstrap process
// by varying last checkpoint's last decided Frame, future sealing Frame and number of frame roots
// available to be run through the election
func testBootstrap_ReprocessRoots(t *testing.T, lastDecidedFrame, sealingFrame, numFrames consensus.Frame) {
	nodes := consensustest.GenNodes(1)
	engine, _, eventSource, _ := NewCoreConsensus(nodes, []consensus.Weight{1})
	engine.store.SetLastDecidedState(&consensusstore.LastDecidedState{LastDecidedFrame: lastDecidedFrame})
	numAtropoiDelivered := consensus.Frame(0)
	engine.Bootstrap(consensus.ConsensusCallbacks{
		BeginBlock: func(block *consensus.Block) consensus.BlockCallbacks {
			return consensus.BlockCallbacks{
				EndBlock: func() (sealEpoch *consensus.Validators) {
					numAtropoiDelivered++
					if currentFrame := lastDecidedFrame + numAtropoiDelivered; currentFrame == sealingFrame {
						return engine.election.validators
					}
					return nil
				},
			}
		},
	})
	roots := make([]*consensustest.TestEvent, numFrames)
	roots[0] = prepareTestRoot(t, engine, eventSource, 0, nodes[0], consensus.EventHashes{})
	for i := 1; i < len(roots); i++ {
		roots[i] = prepareTestRoot(t, engine, eventSource, i, nodes[0], consensus.EventHashes{roots[i-1].ID()})
	}
	if err := engine.bootstrapElection(); err != nil {
		t.Fatal(err)
	}

	// scenario 1 - not enough frames to deliver anything (at least 2 above are necessary) i.e. numFrames < lastDecidedFrame + 2
	expectedNumAtropoiDelivered := consensus.Frame(0)
	if numFrames >= lastDecidedFrame+2 {
		// scenario 2 - enough frames to deliver and sealingFrame value (!= 0) provided
		if sealingFrame != 0 {
			expectedNumAtropoiDelivered = sealingFrame - lastDecidedFrame
		} else {
			// scenario 3 - enough frames to deliver and no sealingFrame value provided
			// implies that all frames with 2+ frames above will recieve their atropoi
			// offset the expected number by -2 as last two frames don't have enough frames above to make a decision for them
			expectedNumAtropoiDelivered = numFrames - 2 - lastDecidedFrame
		}
	}
	if expectedNumAtropoiDelivered != numAtropoiDelivered {
		t.Fatalf("unexpected number of atropoi delivered, expected: %d, got: %d", expectedNumAtropoiDelivered, numAtropoiDelivered)
	}
}

// prepareTestRoot creates, indexes and persists frame roots
// we omit root elections to simulate a mid-epoch bootstrap scenario
func prepareTestRoot(
	t *testing.T,
	lachesis *IndexedLachesis,
	eventSource *consensustest.TestEventSource,
	enumeration int,
	validatorID consensus.ValidatorID,
	parents consensus.EventHashes,
) *consensustest.TestEvent {
	root := &consensustest.TestEvent{}
	root.SetCreator(validatorID)
	root.SetID([24]byte{byte(enumeration)})
	root.SetFrame(consensus.Frame(enumeration + 1))
	root.SetSeq(consensus.Seq(enumeration + 1))
	root.SetParents(parents)
	eventSource.SetEvent(root)
	if err := lachesis.DagIndexer.Add(root); err != nil {
		t.Fatal(err)
	}
	lachesis.store.AddRoot(root)
	return root
}
