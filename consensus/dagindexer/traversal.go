// Copyright (c) 2026 Sonic Operations Ltd

package dagindexer

import (
	"errors"

	"github.com/0xsoniclabs/consensus/consensus"
)

// DfsSubgraph iterates all the event which are reachable by head, and accepted by a filter
// Excluding head
// filter MAY BE called twice for the same event.
func (vi *Index) DfsSubgraph(head consensus.Event, walk func(consensus.EventHash) (godeeper bool)) error {
	stack := make(consensus.EventHashStack, 0, vi.validators.Len()*5)

	// first element
	stack.PushAll(head.Parents())

	for next := stack.Pop(); next != nil; next = stack.Pop() {
		curr := *next

		// filter
		if !walk(curr) {
			continue
		}

		event := vi.getEvent(curr)
		if event == nil {
			return errors.New("event not found " + curr.String())
		}

		// memorize parents
		stack.PushAll(event.Parents())
	}

	return nil
}
