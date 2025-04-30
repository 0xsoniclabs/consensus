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
	"container/heap"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"
)

func TestLeaderHeap_RandomPushPop(t *testing.T) {
	leaderHeap := NewLeaderHeap()
	leaders := make([]*leaderDecision, 100)
	for i := range leaders {
		leaders[i] = &leaderDecision{LeaderHash: consensus.EventHash{byte(i)}, Frame: consensus.Frame(i)}
	}
	rand.Shuffle(len(leaders), func(i, j int) { leaders[i], leaders[j] = leaders[j], leaders[i] })
	for _, leaderDecision := range leaders {
		heap.Push(leaderHeap, leaderDecision)
	}
	for i := range leaders {
		want, got := consensus.EventHash{byte(i)}, heap.Pop(leaderHeap).(*leaderDecision).LeaderHash
		if want != got {
			t.Errorf("expected popped leader hash to be %v, got: %v", want, got)
		}
	}
}

func TestLeaderHeap_SingleDeliveredSequence(t *testing.T) {
	testLeaderHeapDelivery(
		t,
		100,
		[]*leaderDecision{{100, consensus.EventHash{100}}, {101, consensus.EventHash{101}}, {102, consensus.EventHash{102}}},
		[]*leaderDecision{{100, consensus.EventHash{100}}, {101, consensus.EventHash{101}}, {102, consensus.EventHash{102}}},
		[]*leaderDecision{},
	)
}
func TestLeaderHeap_EmptyDeliverySequence(t *testing.T) {
	testLeaderHeapDelivery(
		t,
		100,
		[]*leaderDecision{{101, consensus.EventHash{101}}, {102, consensus.EventHash{102}}},
		[]*leaderDecision{},
		[]*leaderDecision{{101, consensus.EventHash{101}}, {102, consensus.EventHash{102}}},
	)
}
func TestLeaderHeap_BrokenDeliverySequence(t *testing.T) {
	testLeaderHeapDelivery(
		t,
		100,
		[]*leaderDecision{{100, consensus.EventHash{100}}, {101, consensus.EventHash{101}}, {104, consensus.EventHash{104}}, {105, consensus.EventHash{105}}},
		[]*leaderDecision{{100, consensus.EventHash{100}}, {101, consensus.EventHash{101}}},
		[]*leaderDecision{{104, consensus.EventHash{104}}, {105, consensus.EventHash{105}}},
	)
}

func testLeaderHeapDelivery(
	t *testing.T,
	frameToDeliver consensus.Frame,
	leaders []*leaderDecision,
	expectedDelivered []*leaderDecision,
	expectedContainer []*leaderDecision,
) {
	leaderHeap := NewLeaderHeap()
	for _, leader := range leaders {
		heap.Push(leaderHeap, leader)
	}
	delivered := leaderHeap.getDeliveryReadyLeaders(frameToDeliver)
	if !slices.EqualFunc(delivered, expectedDelivered, func(a, b *leaderDecision) bool { return a.LeaderHash == b.LeaderHash }) {
		t.Errorf("incorrect delivered leaders sequence, expected: %v, got: %v", expectedDelivered, delivered)
	}
	if !slices.EqualFunc(leaderHeap.container, expectedContainer, func(a, b *leaderDecision) bool { return a.LeaderHash == b.LeaderHash }) {
		t.Errorf("incorrect remaining leaders container, expected: %v, got: %v", expectedContainer, leaderHeap.container)
	}
}

func (ad *leaderDecision) String() string {
	return fmt.Sprintf("[frame: %d, hash: %v]", ad.Frame, ad.LeaderHash)
}
