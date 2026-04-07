// Copyright (c) 2026 Sonic Operations Ltd

package consensustest

import (
	"github.com/0xsoniclabs/consensus/consensus"
)

// TestEvents is a ordered slice of events.
type TestEvents []*TestEvent

// ByParents returns events topologically ordered by parent dependency.
// Used only for tests.
func ByParents(ee consensus.Events) (res consensus.Events) {
	unsorted := make(consensus.Events, len(ee))
	exists := consensus.EventHashSet{}
	for i, e := range ee {
		unsorted[i] = e
		exists.Add(e.ID())
	}
	ready := consensus.EventHashSet{}
	for len(unsorted) > 0 {
	EVENTS:
		for i, e := range unsorted {

			for _, p := range e.Parents() {
				if exists.Contains(p) && !ready.Contains(p) {
					continue EVENTS
				}
			}

			res = append(res, e)
			unsorted = append(unsorted[0:i], unsorted[i+1:]...)
			ready.Add(e.ID())
			break
		}
	}

	return
}

// ByParents returns events topologically ordered by parent dependency.
// Used only for tests.
func (ee TestEvents) ByParents() (res TestEvents) {
	unsorted := make(consensus.Events, len(ee))
	for i, e := range ee {
		unsorted[i] = e
	}
	sorted := ByParents(unsorted)
	res = make(TestEvents, len(ee))
	for i, e := range sorted {
		res[i] = e.(*TestEvent)
	}

	return
}
