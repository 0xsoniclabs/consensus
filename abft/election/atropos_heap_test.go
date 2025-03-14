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
	"container/heap"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/0xsoniclabs/consensus/consensustypes"
)

func TestAtroposHeap_RandomPushPop(t *testing.T) {
	atroposHeap := NewAtroposHeap()
	atropoi := make([]*AtroposDecision, 100)
	for i := range atropoi {
		atropoi[i] = &AtroposDecision{AtroposHash: consensustypes.EventHash{byte(i)}, Frame: consensustypes.Frame(i)}
	}
	rand.Shuffle(len(atropoi), func(i, j int) { atropoi[i], atropoi[j] = atropoi[j], atropoi[i] })
	for _, atroposDecision := range atropoi {
		heap.Push(atroposHeap, atroposDecision)
	}
	for i := range atropoi {
		want, got := consensustypes.EventHash{byte(i)}, heap.Pop(atroposHeap).(*AtroposDecision).AtroposHash
		if want != got {
			t.Errorf("expected popped atropos hash to be %v, got: %v", want, got)
		}
	}
}

func TestAtroposHeap_SingleDeliveredSequence(t *testing.T) {
	testAtroposHeapDelivery(
		t,
		100,
		[]*AtroposDecision{{100, consensustypes.EventHash{100}}, {101, consensustypes.EventHash{101}}, {102, consensustypes.EventHash{102}}},
		[]*AtroposDecision{{100, consensustypes.EventHash{100}}, {101, consensustypes.EventHash{101}}, {102, consensustypes.EventHash{102}}},
		[]*AtroposDecision{},
	)
}
func TestAtroposHeap_EmptyDeliverySequence(t *testing.T) {
	testAtroposHeapDelivery(
		t,
		100,
		[]*AtroposDecision{{101, consensustypes.EventHash{101}}, {102, consensustypes.EventHash{102}}},
		[]*AtroposDecision{},
		[]*AtroposDecision{{101, consensustypes.EventHash{101}}, {102, consensustypes.EventHash{102}}},
	)
}
func TestAtroposHeap_BrokenDeliverySequence(t *testing.T) {
	testAtroposHeapDelivery(
		t,
		100,
		[]*AtroposDecision{{100, consensustypes.EventHash{100}}, {101, consensustypes.EventHash{101}}, {104, consensustypes.EventHash{104}}, {105, consensustypes.EventHash{105}}},
		[]*AtroposDecision{{100, consensustypes.EventHash{100}}, {101, consensustypes.EventHash{101}}},
		[]*AtroposDecision{{104, consensustypes.EventHash{104}}, {105, consensustypes.EventHash{105}}},
	)
}

func testAtroposHeapDelivery(
	t *testing.T,
	frameToDeliver consensustypes.Frame,
	atropoi []*AtroposDecision,
	expectedDelivered []*AtroposDecision,
	expectedContainer []*AtroposDecision,
) {
	atroposHeap := NewAtroposHeap()
	for _, atropos := range atropoi {
		heap.Push(atroposHeap, atropos)
	}
	delivered := atroposHeap.getDeliveryReadyAtropoi(frameToDeliver)
	if !slices.EqualFunc(delivered, expectedDelivered, func(a, b *AtroposDecision) bool { return a.AtroposHash == b.AtroposHash }) {
		t.Errorf("incorrect delivered atropi sequence, expected: %v, got: %v", expectedDelivered, delivered)
	}
	if !slices.EqualFunc(atroposHeap.container, expectedContainer, func(a, b *AtroposDecision) bool { return a.AtroposHash == b.AtroposHash }) {
		t.Errorf("incorrect remaining atropi container, expected: %v, got: %v", expectedContainer, atroposHeap.container)
	}
}

func (ad *AtroposDecision) String() string {
	return fmt.Sprintf("[frame: %d, hash: %v]", ad.Frame, ad.AtroposHash)
}
