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
	"errors"

	"github.com/0xsoniclabs/consensus/consensus"
)

type eventFilterFn func(event consensus.Event) bool

// dfsSubgraph iterates all the events which are observed by head, and accepted by a filter.
// filter MAY BE called twice for the same event.
func (p *Orderer) dfsSubgraph(head consensus.EventHash, filter eventFilterFn) error {
	stack := make(consensus.EventHashStack, 0, 300)

	for pwalk := &head; pwalk != nil; pwalk = stack.Pop() {
		walk := *pwalk

		event := p.Input.GetEvent(walk)
		if event == nil {
			return errors.New("event not found " + walk.String())
		}

		// filter
		if !filter(event) {
			continue
		}

		// memorize parents
		for _, parent := range event.Parents() {
			stack.Push(parent)
		}
	}

	return nil
}
