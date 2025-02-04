package tdag

import (
	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/dag"
)

type TestEvent struct {
	dag.MutableBaseEvent
	Name string
}

func (e *TestEvent) AddParent(id hash.Event) {
	parents := e.Parents()
	parents.Add(id)
	e.SetParents(parents)
}
