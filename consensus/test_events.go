// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensus

import (
	"strings"
)

// TestEvents is a ordered slice of events.
type TestEvents []*TestEvent

// String returns human readable representation.
func (ee TestEvents) String() string {
	ss := make([]string, len(ee))
	for i := 0; i < len(ee); i++ {
		ss[i] = ee[i].String()
	}
	return strings.Join(ss, " ")
}

// ByParents returns events topologically ordered by parent dependency.
// Used only for tests.
func ByParents(ee Events) (res Events) {
	unsorted := make(Events, len(ee))
	exists := EventHashSet{}
	for i, e := range ee {
		unsorted[i] = e
		exists.Add(e.ID())
	}
	ready := EventHashSet{}
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
	unsorted := make(Events, len(ee))
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
