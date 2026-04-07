// Copyright (c) 2026 Sonic Operations Ltd

package consensusstore

import (
	"github.com/0xsoniclabs/consensus/consensus"
)

// SetEventConfirmedOn stores confirmed event ctype.
func (s *Store) SetEventConfirmedOn(e consensus.EventHash, on consensus.Frame) {
	key := e.Bytes()

	if err := s.EpochTable.ConfirmedEvent.Put(key, on.Bytes()); err != nil {
		s.crit(err)
	}
}

// GetEventConfirmedOn returns confirmed event ctype.
func (s *Store) GetEventConfirmedOn(e consensus.EventHash) consensus.Frame {
	key := e.Bytes()

	buf, err := s.EpochTable.ConfirmedEvent.Get(key)
	if err != nil {
		s.crit(err)
	}
	if buf == nil {
		return 0
	}

	return consensus.BytesToFrame(buf)
}
