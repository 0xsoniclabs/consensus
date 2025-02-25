package election

import (
	"container/heap"

	"github.com/0xsoniclabs/consensus/inter/idx"
)

// atroposHeap is a min-heap of Atropos decisions ordered by Frames.
type atroposHeap struct {
	container []*AtroposDecision
}

func NewAtroposHeap() *atroposHeap {
	return &atroposHeap{make([]*AtroposDecision, 0)}
}

func (h atroposHeap) Len() int           { return len(h.container) }
func (h atroposHeap) Less(i, j int) bool { return h.container[i].Frame < h.container[j].Frame }
func (h atroposHeap) Swap(i, j int)      { h.container[i], h.container[j] = h.container[j], h.container[i] }

func (h *atroposHeap) Push(x any) {
	h.container = append(h.container, x.(*AtroposDecision))
}

func (h *atroposHeap) Pop() any {
	backIdx := len(h.container) - 1
	toPop := h.container[backIdx]
	h.container = h.container[0:backIdx]
	return toPop
}

// getDeliveryReadyAtropoi pops and returns only continuous sequences of decided atropoi
// that begin with `frameToDeliver` frame number
// example 1: frameToDeliver = 100, heapBuffer = [100, 101, 102] -> deliveredAtropoi = [100, 101, 102], heapBuffer = []
// example 2: frameToDeliver = 100, heapBuffer = [101, 102] -> deliveredAtropoi = [], heapBuffer = [101, 102]
// example 3: frameToDeliver = 100, heapBuffer = [100, 101, 104, 105] -> deliveredAtropoi = [100, 101], heapBuffer=[104, 105]
func (ah *atroposHeap) getDeliveryReadyAtropoi(frameToDeliver idx.Frame) []*AtroposDecision {
	atropoi := make([]*AtroposDecision, 0)
	for len(ah.container) > 0 && ah.container[0].Frame == frameToDeliver {
		atropoi = append(atropoi, heap.Pop(ah).(*AtroposDecision))
		frameToDeliver++
	}
	return atropoi
}
