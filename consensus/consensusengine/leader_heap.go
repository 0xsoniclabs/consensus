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

	"github.com/0xsoniclabs/consensus/consensus"
)

// leaderHeap is a min-heap of Leader decisions ordered by Frames.
type leaderHeap struct {
	container []*leaderDecision
}

func NewLeaderHeap() *leaderHeap {
	return &leaderHeap{make([]*leaderDecision, 0)}
}

func (h leaderHeap) Len() int           { return len(h.container) }
func (h leaderHeap) Less(i, j int) bool { return h.container[i].Frame < h.container[j].Frame }
func (h leaderHeap) Swap(i, j int)      { h.container[i], h.container[j] = h.container[j], h.container[i] }

func (h *leaderHeap) Push(x any) {
	h.container = append(h.container, x.(*leaderDecision))
}

func (h *leaderHeap) Pop() any {
	backIdx := len(h.container) - 1
	toPop := h.container[backIdx]
	h.container = h.container[0:backIdx]
	return toPop
}

// getDeliveryReadyLeaders pops and returns only continuous sequences of decided leaders
// that begin with `frameToDeliver` frame number
// example 1: frameToDeliver = 100, heapBuffer = [100, 101, 102] -> deliveredLeaders = [100, 101, 102], heapBuffer = []
// example 2: frameToDeliver = 100, heapBuffer = [101, 102] -> deliveredLeaders = [], heapBuffer = [101, 102]
// example 3: frameToDeliver = 100, heapBuffer = [100, 101, 104, 105] -> deliveredLeaders = [100, 101], heapBuffer=[104, 105]
func (ah *leaderHeap) getDeliveryReadyLeaders(frameToDeliver consensus.Frame) []*leaderDecision {
	leaders := make([]*leaderDecision, 0)
	for len(ah.container) > 0 && ah.container[0].Frame == frameToDeliver {
		leaders = append(leaders, heap.Pop(ah).(*leaderDecision))
		frameToDeliver++
	}
	return leaders
}
