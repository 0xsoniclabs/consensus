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
	"github.com/0xsoniclabs/consensus/consensus"
)

type TestEvent struct {
	consensus.MutableBaseEvent
	Name         string
	CreationTime int
}

func (e *TestEvent) AddParent(id consensus.EventHash) {
	parents := e.Parents()
	parents.Add(id)
	e.SetParents(parents)
}

func (t *TestEvent) CalcCreationTime(eventStore *TestEventSource) {
	t.CreationTime = 0
	selfParentHashPtr := t.SelfParent()
	if selfParentHashPtr == nil {
		return
	}
	for _, parentHash := range t.Parents() {
		parentEvent := eventStore.GetEvent(parentHash).(*TestEvent)
		candidateCreationTime := parentEvent.CreationTime
		if parentHash != *selfParentHashPtr {
			candidateCreationTime += 1
		}
		t.CreationTime = max(t.CreationTime, candidateCreationTime)
	}
}
