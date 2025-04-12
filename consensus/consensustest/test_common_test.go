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
	"bytes"
	"fmt"
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/stretchr/testify/assert"
)

func TestGenNodes_CreateSpecifiedNodeCount(t *testing.T) {
	counts := []int{0, 1, 5, 10}

	for _, count := range counts {
		nodes := GenNodes(count)
		assert.Equal(t, count, len(nodes), "GenNodes should create exactly the requested number of nodes")

		if count > 0 {
			for i, node := range nodes {
				expectedName := "node" + string('A'+rune(i))
				actualName := consensus.GetNodeName(node)
				assert.Equal(t, expectedName, actualName, "Node names should be set correctly")
			}
		}

		seen := make(map[string]bool)
		for _, node := range nodes {
			nodeStr := string(node.Bytes())
			assert.False(t, seen[nodeStr], "All nodes should be unique")
			seen[nodeStr] = true
		}
	}
}

func TestForEachRandFork_CreateEventsWithoutForks(t *testing.T) {
	nodes := GenNodes(3)
	eventCount := 2
	parentCount := 2
	forksCount := 0
	r := NewIntSeededRandGenerator(42) // fixed seed for reproducibility

	events := ForEachRandFork(nodes, nil, eventCount, parentCount, forksCount, r, ForEachEvent{})

	assert.Equal(t, len(nodes), len(events), "Events should be created for each node")

	for _, node := range nodes {
		nodeEvents := events[node]
		assert.Equal(t, eventCount, len(nodeEvents), "Each node should have exactly eventCount events")
	}
}

func TestForEachRandFork_CreateEventsWithForks(t *testing.T) {
	nodes := GenNodes(3)
	eventCount := 5
	parentCount := 2
	forksCount := 2
	cheaters := []consensus.ValidatorID{nodes[0]} // first node is a cheater
	r := NewIntSeededRandGenerator(42)

	var processedEvents []*TestEvent
	callback := ForEachEvent{
		Process: func(e consensus.Event, name string) {
			processedEvents = append(processedEvents, e.(*TestEvent))
		},
	}

	events := ForEachRandFork(nodes, cheaters, eventCount, parentCount, forksCount, r, callback)

	assert.NotEmpty(t, processedEvents, "Process callback should be called")

	for _, node := range nodes {
		assert.NotEmpty(t, events[node], "Each node should have events")
	}

	cheaterEvents := events[cheaters[0]]

	// Detecting duplicate seq
	seqNums := make(map[consensus.Seq]bool)
	for _, e := range cheaterEvents {
		seqNums[e.Seq()] = true
	}

	assert.Less(t, len(seqNums), eventCount,
		"Number of unique sequence numbers should be less than eventCount for cheater nodes")
}

func TestForEachRandFork_WithCallbacks(t *testing.T) {
	nodes := GenNodes(2)
	eventCount := 2
	parentCount := 2
	forksCount := 0
	r := NewIntSeededRandGenerator(42)

	var builtEvents []*TestEvent
	var processedEvents []*TestEvent

	callback := ForEachEvent{
		Build: func(e consensus.MutableEvent, name string) error {
			builtEvents = append(builtEvents, e.(*TestEvent))
			return nil
		},
		Process: func(e consensus.Event, name string) {
			processedEvents = append(processedEvents, e.(*TestEvent))
		},
	}

	ForEachRandFork(nodes, nil, eventCount, parentCount, forksCount, r, callback)

	assert.NotEmpty(t, builtEvents, "Build callback should be called")
	assert.NotEmpty(t, processedEvents, "Process callback should be called")
	assert.Equal(t, len(builtEvents), len(processedEvents),
		"Build and Process callbacks should be called the same number of times")
}

func TestForEachRandFork_BuildCallbackError(t *testing.T) {
	nodes := GenNodes(3)
	eventCount := 5
	parentCount := 2
	forksCount := 0
	r := NewIntSeededRandGenerator(42)

	var processedEventNames []string

	// Create a callback that returns error for specific events
	callback := ForEachEvent{
		Build: func(e consensus.MutableEvent, name string) error {
			if name == "a001" || name == "b001" {
				return fmt.Errorf("intentional error for %s", name)
			}
			return nil
		},
		Process: func(e consensus.Event, name string) {
			processedEventNames = append(processedEventNames, name)
		},
	}

	events := ForEachRandFork(nodes, nil, eventCount, parentCount, forksCount, r, callback)

	for _, names := range processedEventNames {
		assert.NotEqual(t, "a001", names, "Event with Build error should be skipped")
		assert.NotEqual(t, "b001", names, "Event with Build error should be skipped")
	}

	totalEvents := 0
	for _, nodeEvents := range events {
		totalEvents += len(nodeEvents)
	}
	assert.Less(t, totalEvents, len(nodes)*eventCount, "Total events should be less than expected due to skipped events")
}

func TestForEachRandEvent_DelegateToForEachRandFork(t *testing.T) {
	nodes := GenNodes(2)
	eventCount := 2
	parentCount := 2
	r := NewIntSeededRandGenerator(42)

	events1 := ForEachRandEvent(nodes, eventCount, parentCount, r, ForEachEvent{})

	r = NewIntSeededRandGenerator(42)
	events2 := ForEachRandFork(nodes, []consensus.ValidatorID{}, eventCount, parentCount, 0, r, ForEachEvent{})
	assert.Equal(t, len(events1), len(events2), "ForEachRandEvent should produce the same number of node events as ForEachRandFork")

	for node, nodeEvents1 := range events1 {
		nodeEvents2 := events2[node]
		assert.Equal(t, len(nodeEvents1), len(nodeEvents2),
			"Each node should have the same number of events in both functions")

		for i := range nodeEvents1 {
			assert.Equal(t, nodeEvents1[i].ID(), nodeEvents2[i].ID(),
				"Events should have the same ID")
			assert.Equal(t, nodeEvents1[i].Seq(), nodeEvents2[i].Seq(),
				"Events should have the same sequence")
		}
	}
}

func TestGenRandEvents_DelegateToForEachRandEvent(t *testing.T) {
	nodes := GenNodes(2)
	eventCount := 2
	parentCount := 2
	r := NewIntSeededRandGenerator(42)

	events1 := GenRandEvents(nodes, eventCount, parentCount, r)

	r = NewIntSeededRandGenerator(42)
	events2 := ForEachRandEvent(nodes, eventCount, parentCount, r, ForEachEvent{})

	assert.Equal(t, len(events1), len(events2), "GenRandEvents should produce the same number of node events as ForEachRandEvent")

	for node, nodeEvents1 := range events1 {
		nodeEvents2 := events2[node]
		assert.Equal(t, len(nodeEvents1), len(nodeEvents2),
			"Each node should have the same number of events in both functions")

		for i := range nodeEvents1 {
			assert.Equal(t, nodeEvents1[i].ID(), nodeEvents2[i].ID(),
				"Events should have the same ID")
			assert.Equal(t, nodeEvents1[i].Seq(), nodeEvents2[i].Seq(),
				"Events should have the same sequence")
		}
	}
}

func TestCalcHashForTestEvent_CalculateHash(t *testing.T) {
	event := &TestEvent{}
	event.SetCreator(FakePeer())
	event.SetSeq(1)
	event.SetLamport(1)
	event.Name = "test"

	hash := CalcHashForTestEvent(event)

	var zeroHash [24]byte
	assert.False(t, bytes.Equal(hash[:], zeroHash[:]), "Hash should not be all zeros")

	hash2 := CalcHashForTestEvent(event)
	assert.Equal(t, hash, hash2, "Hash calculation should be deterministic")

	event.SetSeq(2)
	hash3 := CalcHashForTestEvent(event)
	assert.NotEqual(t, hash, hash3, "Hash should change when event content changes")
}

func TestDelPeerIndex_FlattenEventMap(t *testing.T) {
	nodes := GenNodes(3)
	eventCount := 2
	r := NewIntSeededRandGenerator(42)
	eventMap := GenRandEvents(nodes, eventCount, 2, r)

	totalEventsCount := 0
	for _, events := range eventMap {
		totalEventsCount += len(events)
	}

	flatEvents := delPeerIndex(eventMap)

	assert.Equal(t, totalEventsCount, len(flatEvents),
		"Flattened events should contain all events from the map")

	eventIDs := make(map[consensus.EventHash]bool)
	for _, event := range flatEvents {
		eventIDs[event.ID()] = true
	}

	for _, events := range eventMap {
		for _, event := range events {
			assert.True(t, eventIDs[event.ID()],
				"All events from the map should be in the flattened result")
		}
	}
}

func TestFakePeer_GenerateRandomPeerID(t *testing.T) {
	peer1 := FakePeer()
	peer2 := FakePeer()

	assert.NotEqual(t, peer1, peer2, "Generated fake peers should be different")

	var emptyID consensus.ValidatorID
	assert.NotEqual(t, emptyID, peer1, "Peer ID should not be empty")
	assert.Len(t, peer1.Bytes(), 4, "Peer ID should be 4 bytes")
}

func TestFakeEpoch_ReturnConstantValue(t *testing.T) {
	epoch := FakeEpoch()
	assert.Equal(t, consensus.Epoch(123456), epoch, "FakeEpoch should return 123456")

	epoch2 := FakeEpoch()
	assert.Equal(t, epoch, epoch2, "FakeEpoch should always return the same value")
}

func TestFakeEventHash_GenerateHashWithEpoch(t *testing.T) {
	hash := FakeEventHash()

	epoch := consensus.BytesToEpoch(hash[0:4])
	assert.Equal(t, FakeEpoch(), epoch, "First 4 bytes of the hash should encode the fake epoch")

	hash2 := FakeEventHash()
	assert.NotEqual(t, hash, hash2, "Generated fake event hashes should be different")
}

func TestFakeEventHashes_GenerateMultipleHashes(t *testing.T) {
	counts := []int{0, 1, 5, 10}

	for _, count := range counts {
		hashes := FakeEventHashes(count)
		assert.Equal(t, count, len(hashes), "FakeEventHashes should generate exactly the requested number of hashes")

		seen := make(map[consensus.EventHash]bool)
		for _, hash := range hashes {
			assert.False(t, seen[hash], "All hashes should be unique")
			seen[hash] = true
		}

		for _, hash := range hashes {
			epoch := consensus.BytesToEpoch(hash[0:4])
			assert.Equal(t, FakeEpoch(), epoch, "Each hash should encode the fake epoch in the first 4 bytes")
		}
	}
}

func TestFakeHash_GenerateRandomHash(t *testing.T) {
	hash1 := FakeHash()
	hash2 := FakeHash()

	assert.NotEqual(t, hash1, hash2, "Generated hashes should be different")

	hashWithSeed1 := FakeHash(42)
	hashWithSeed2 := FakeHash(42)

	assert.Equal(t, hashWithSeed1, hashWithSeed2, "Hashes generated with the same seed should be equal")

	hashWithDiffSeed := FakeHash(43)
	assert.NotEqual(t, hashWithSeed1, hashWithDiffSeed, "Hashes generated with different seeds should be different")
}

func TestNewIntSeededRandGenerator_CreateDeterministicGenerator(t *testing.T) {
	r1 := NewIntSeededRandGenerator(42)
	r2 := NewIntSeededRandGenerator(42)

	const count = 10
	nums1 := make([]uint64, count)
	nums2 := make([]uint64, count)

	for i := 0; i < count; i++ {
		nums1[i] = r1.Uint64()
		nums2[i] = r2.Uint64()
	}

	assert.Equal(t, nums1, nums2, "Random generators with the same seed should produce the same sequence")

	r3 := NewIntSeededRandGenerator(43)
	nums3 := make([]uint64, count)

	for i := 0; i < count; i++ {
		nums3[i] = r3.Uint64()
	}

	assert.NotEqual(t, nums1, nums3, "Random generators with different seeds should produce different sequences")
}
