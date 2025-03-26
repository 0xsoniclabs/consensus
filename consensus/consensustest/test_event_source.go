package consensustest

import "github.com/0xsoniclabs/consensus/consensus"

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
