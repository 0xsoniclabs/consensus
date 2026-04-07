// Copyright (c) 2026 Sonic Operations Ltd

package consensustest

import (
	"github.com/0xsoniclabs/consensus/consensus"
)

type TestEvent struct {
	consensus.MutableBaseEvent
	Name string
}

func (e *TestEvent) AddParent(id consensus.EventHash) {
	parents := e.Parents()
	parents.Add(id)
	e.SetParents(parents)
}
