package election

import (
	"container/heap"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/idx"
)

func TestAtroposHeap_RandomPushPop(t *testing.T) {
	atroposHeap := NewAtroposHeap()
	atropoi := make([]*AtroposDecision, 100)
	for i := range atropoi {
		atropoi[i] = &AtroposDecision{AtroposHash: hash.Event{byte(i)}, Frame: idx.Frame(i)}
	}
	rand.Shuffle(len(atropoi), func(i, j int) { atropoi[i], atropoi[j] = atropoi[j], atropoi[i] })
	for _, atroposDecision := range atropoi {
		heap.Push(atroposHeap, atroposDecision)
	}
	for i := range atropoi {
		want, got := hash.Event{byte(i)}, heap.Pop(atroposHeap).(*AtroposDecision).AtroposHash
		if want != got {
			t.Errorf("expected popped atropos hash to be %v, got: %v", want, got)
		}
	}
}

func TestAtroposHeap_SingleDeliveredSequence(t *testing.T) {
	testAtroposHeapDelivery(
		t,
		100,
		[]*AtroposDecision{{100, hash.Event{100}}, {101, hash.Event{101}}, {102, hash.Event{102}}},
		[]*AtroposDecision{{100, hash.Event{100}}, {101, hash.Event{101}}, {102, hash.Event{102}}},
		[]*AtroposDecision{},
	)
}
func TestAtroposHeap_EmptyDeliverySequence(t *testing.T) {
	testAtroposHeapDelivery(
		t,
		100,
		[]*AtroposDecision{{101, hash.Event{101}}, {102, hash.Event{102}}},
		[]*AtroposDecision{},
		[]*AtroposDecision{{101, hash.Event{101}}, {102, hash.Event{102}}},
	)
}
func TestAtroposHeap_BrokenDeliverySequence(t *testing.T) {
	testAtroposHeapDelivery(
		t,
		100,
		[]*AtroposDecision{{100, hash.Event{100}}, {101, hash.Event{101}}, {104, hash.Event{104}}, {105, hash.Event{105}}},
		[]*AtroposDecision{{100, hash.Event{100}}, {101, hash.Event{101}}},
		[]*AtroposDecision{{104, hash.Event{104}}, {105, hash.Event{105}}},
	)
}

func testAtroposHeapDelivery(
	t *testing.T,
	frameToDeliver idx.Frame,
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
