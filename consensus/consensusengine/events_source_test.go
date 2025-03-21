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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/0xsoniclabs/consensus/consensus"
)

/*
 * Tests:
 */

func TestEventStore(t *testing.T) {
	store := NewEventStore()

	t.Run("NotExisting", func(t *testing.T) {
		assertar := assert.New(t)

		h := consensus.FakeEvent()
		e1 := store.GetEvent(h)
		assertar.Nil(e1)
	})

	t.Run("Events", func(t *testing.T) {
		assertar := assert.New(t)

		nodes := consensus.GenNodes(5)
		consensus.ForEachRandEvent(nodes, int(TestMaxEpochEvents)-1, 4, nil, consensus.ForEachEvent{
			Process: func(e consensus.Event, name string) {
				store.SetEvent(e)
				e1 := store.GetEvent(e.ID())

				if !assertar.Equal(e, e1) {
					t.Fatal(e.String() + " != " + e1.String())
				}
			},
		})
	})

	store.Close()
}
