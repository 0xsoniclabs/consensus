package consensusengine

import (
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensustest"
)

func TestTraversal_EventNotFound(t *testing.T) {
	engine, _, eventSource, _ := NewCoreConsensus(consensustest.GenNodes(1), []consensus.Weight{1})
	parentEvent, childEvent := &consensustest.TestEvent{}, &consensustest.TestEvent{}
	// don't set the parent event in the EventSource
	parentEvent.SetID([24]byte{1})
	childEvent.SetID([24]byte{2})
	childEvent.SetParents(consensus.EventHashes{parentEvent.ID()})
	eventSource.SetEvent(childEvent)

	if err := engine.dfsSubgraph(consensus.EventHash{0}, func(event consensus.Event) bool { return true }); err == nil {
		t.Fatal("expected event not found error but recieved none")
	}
}

func TestTraversal_CorrectEventsVisitedMultipleTraversals(t *testing.T) {
	const (
		NUM_NODES           = 5
		NUM_EVENTS_PER_NODE = 100
	)
	nodes := consensustest.GenNodes(NUM_NODES)
	engine, _, eventSource, _ := NewCoreConsensus(consensustest.GenNodes(1), []consensus.Weight{1})
	events := consensustest.ForEachRandEvent(nodes, NUM_EVENTS_PER_NODE, 3, nil,
		consensustest.ForEachEvent{
			Process: func(e consensus.Event, name string) {
				eventSource.SetEvent(e)
			},
		})

	// pick a random event in the ~middle of the DAG
	firstAnchor := events[nodes[0]][len(events[nodes[0]])/2]
	alreadyVisitedEvents := consensus.EventHashSet{}
	numVisited := 0
	filterFn := func(event consensus.Event) bool {
		if alreadyVisitedEvents.Contains(event.ID()) {
			return false
		}
		alreadyVisitedEvents.Add(event.ID())
		numVisited++
		return true
	}
	// first traversal stops at genesis events
	if err := engine.dfsSubgraph(firstAnchor.ID(), filterFn); err != nil {
		t.Fatal(err)
	}

	// second anchor has all validator's last events as parents
	secondAnchor := &consensustest.TestEvent{}
	for _, events := range events {
		for _, event := range events {
			secondAnchor.AddParent(event.ID())
		}
	}
	eventSource.SetEvent(secondAnchor)
	// second traversal should stop at already visited events
	if err := engine.dfsSubgraph(secondAnchor.ID(), filterFn); err != nil {
		t.Fatal(err)
	}
	// +1 for second anchor
	if want := NUM_NODES*NUM_EVENTS_PER_NODE + 1; numVisited != want {
		t.Fatalf("incorrect number of visited events. expected: %d, got: %d", want, numVisited)
	}
}
