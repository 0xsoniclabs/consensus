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
	"container/heap"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"
)

func TestLeaderHeap_RandomPushPop(t *testing.T) {
	leaderHeap := NewLeaderHeap()
	leaders := make([]*leaderCertification, 100)
	for i := range leaders {
		leaders[i] = &leaderCertification{LeaderHash: consensus.EventHash{byte(i)}, Frame: consensus.Frame(i)}
	}
	rand.Shuffle(len(leaders), func(i, j int) { leaders[i], leaders[j] = leaders[j], leaders[i] })
	for _, leaderCertification := range leaders {
		heap.Push(leaderHeap, leaderCertification)
	}
	for i := range leaders {
		want, got := consensus.EventHash{byte(i)}, heap.Pop(leaderHeap).(*leaderCertification).LeaderHash
		if want != got {
			t.Errorf("expected popped leader hash to be %v, got: %v", want, got)
		}
	}
}

func TestLeaderHeap_SingleDeliveredSequence(t *testing.T) {
	testLeaderHeapDelivery(
		t,
		100,
		[]*leaderCertification{{100, consensus.EventHash{100}}, {101, consensus.EventHash{101}}, {102, consensus.EventHash{102}}},
		[]*leaderCertification{{100, consensus.EventHash{100}}, {101, consensus.EventHash{101}}, {102, consensus.EventHash{102}}},
		[]*leaderCertification{},
	)
}
func TestLeaderHeap_EmptyDeliverySequence(t *testing.T) {
	testLeaderHeapDelivery(
		t,
		100,
		[]*leaderCertification{{101, consensus.EventHash{101}}, {102, consensus.EventHash{102}}},
		[]*leaderCertification{},
		[]*leaderCertification{{101, consensus.EventHash{101}}, {102, consensus.EventHash{102}}},
	)
}
func TestLeaderHeap_BrokenDeliverySequence(t *testing.T) {
	testLeaderHeapDelivery(
		t,
		100,
		[]*leaderCertification{{100, consensus.EventHash{100}}, {101, consensus.EventHash{101}}, {104, consensus.EventHash{104}}, {105, consensus.EventHash{105}}},
		[]*leaderCertification{{100, consensus.EventHash{100}}, {101, consensus.EventHash{101}}},
		[]*leaderCertification{{104, consensus.EventHash{104}}, {105, consensus.EventHash{105}}},
	)
}

func testLeaderHeapDelivery(
	t *testing.T,
	frameToDeliver consensus.Frame,
	leaders []*leaderCertification,
	expectedDelivered []*leaderCertification,
	expectedContainer []*leaderCertification,
) {
	leaderHeap := NewLeaderHeap()
	for _, leader := range leaders {
		heap.Push(leaderHeap, leader)
	}
	delivered := leaderHeap.getDeliveryReadyLeaders(frameToDeliver)
	if !slices.EqualFunc(delivered, expectedDelivered, func(a, b *leaderCertification) bool { return a.LeaderHash == b.LeaderHash }) {
		t.Errorf("incorrect delivered leaders sequence, expected: %v, got: %v", expectedDelivered, delivered)
	}
	if !slices.EqualFunc(leaderHeap.container, expectedContainer, func(a, b *leaderCertification) bool { return a.LeaderHash == b.LeaderHash }) {
		t.Errorf("incorrect remaining leaders container, expected: %v, got: %v", expectedContainer, leaderHeap.container)
	}
}

func (ad *leaderCertification) String() string {
	return fmt.Sprintf("[frame: %d, hash: %v]", ad.Frame, ad.LeaderHash)
}
