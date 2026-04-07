// Copyright (c) 2026 Sonic Operations Ltd

package consensusengine

import (
	"container/heap"

	"github.com/0xsoniclabs/consensus/consensus"
)

// leaderHeap is a min-heap of Leader certifications ordered by Frames.
type leaderHeap struct {
	container []*leaderCertification
}

func NewLeaderHeap() *leaderHeap {
	return &leaderHeap{make([]*leaderCertification, 0)}
}

func (h leaderHeap) Len() int           { return len(h.container) }
func (h leaderHeap) Less(i, j int) bool { return h.container[i].Frame < h.container[j].Frame }
func (h leaderHeap) Swap(i, j int)      { h.container[i], h.container[j] = h.container[j], h.container[i] }

func (h *leaderHeap) Push(x any) {
	h.container = append(h.container, x.(*leaderCertification))
}

func (h *leaderHeap) Pop() any {
	backIdx := len(h.container) - 1
	toPop := h.container[backIdx]
	h.container = h.container[0:backIdx]
	return toPop
}

// getDeliveryReadyLeaders pops and returns only continuous sequences of certified leaders
// that begin with `frameToDeliver` frame number
// example 1: frameToDeliver = 100, heapBuffer = [100, 101, 102] -> deliveredLeaders = [100, 101, 102], heapBuffer = []
// example 2: frameToDeliver = 100, heapBuffer = [101, 102] -> deliveredLeaders = [], heapBuffer = [101, 102]
// example 3: frameToDeliver = 100, heapBuffer = [100, 101, 104, 105] -> deliveredLeaders = [100, 101], heapBuffer=[104, 105]
func (ah *leaderHeap) getDeliveryReadyLeaders(frameToDeliver consensus.Frame) []*leaderCertification {
	leaders := make([]*leaderCertification, 0)
	for len(ah.container) > 0 && ah.container[0].Frame == frameToDeliver {
		leaders = append(leaders, heap.Pop(ah).(*leaderCertification))
		frameToDeliver++
	}
	return leaders
}

func (ah *leaderHeap) isCertificationBuffered(frame consensus.Frame) bool {
	for _, certification := range ah.container {
		if certification.Frame == frame {
			return true
		}
	}
	return false
}
