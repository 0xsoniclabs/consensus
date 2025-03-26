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
	"github.com/0xsoniclabs/consensus/consensus/consensustest"
)

/*
 * Tests:
 */

func TestEventStore(t *testing.T) {
	store := consensustest.NewTestEventSource()

	t.Run("NotExisting", func(t *testing.T) {
		assertar := assert.New(t)

		h := consensustest.FakeEventHash()
		e1 := store.GetEvent(h)
		assertar.Nil(e1)
	})

	t.Run("Events", func(t *testing.T) {
		assertar := assert.New(t)

		nodes := consensustest.GenNodes(5)
		consensustest.ForEachRandEvent(nodes, int(TestMaxEpochEvents)-1, 4, nil, consensustest.ForEachEvent{
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
