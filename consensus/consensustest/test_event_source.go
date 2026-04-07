// Copyright (c) 2026 Sonic Operations Ltd
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensustest

import "github.com/0xsoniclabs/consensus/consensus"

var _ consensus.EventSource = (*TestEventSource)(nil)

// TestEventSource is a abft event storage for test purpose.
// It implements EventSource interface.
type TestEventSource struct {
	db map[consensus.EventHash]consensus.Event
}

// NewTestEventSource creates store over memory map.
func NewTestEventSource() *TestEventSource {
	return &TestEventSource{
		db: map[consensus.EventHash]consensus.Event{},
	}
}

// Close leaves underlying database.
func (s *TestEventSource) Close() {
	s.db = nil
}

// SetEvent stores event.
func (s *TestEventSource) SetEvent(e consensus.Event) {
	s.db[e.ID()] = e
}

// GetEvent returns stored event.
func (s *TestEventSource) GetEvent(h consensus.EventHash) consensus.Event {
	return s.db[h]
}

// HasEvent returns true if event exists.
func (s *TestEventSource) HasEvent(h consensus.EventHash) bool {
	_, ok := s.db[h]
	return ok
}
