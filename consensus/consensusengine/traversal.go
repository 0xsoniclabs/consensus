// Copyright (c) 2026 Sonic Operations Ltd

package consensusengine

import (
	"errors"

	"github.com/0xsoniclabs/consensus/consensus"
)

type eventFilterFn func(event consensus.Event) bool

// dfsSubgraph iterates all the events which are reachable by head, and accepted by a filter.
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
