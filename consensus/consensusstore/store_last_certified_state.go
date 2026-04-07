// Copyright (c) 2026 Sonic Operations Ltd

package consensusstore

import "github.com/0xsoniclabs/consensus/consensus"

// certified state key ("d" for legacy reasons for "decided")
const csKey = "d"

// LastCertifiedState is for persistent storing.
type LastCertifiedState struct {
	// fields can change only after a frame is certified
	LastCertifiedFrame consensus.Frame
}

// SetLastCertifiedState save LastCertifiedState.
// LastCertifiedState is seldom read; so no cache.
func (s *Store) SetLastCertifiedState(v *LastCertifiedState) {
	s.cache.LastCertifiedState = v

	s.set(s.table.LastCertifiedState, []byte(csKey), v)
}

// GetLastCertifiedState returns stored LastCertifiedState.
// State is seldom read; so no cache.
func (s *Store) GetLastCertifiedState() *LastCertifiedState {
	if s.cache.LastCertifiedState != nil {
		return s.cache.LastCertifiedState
	}

	w, exists := s.get(s.table.LastCertifiedState, []byte(csKey), &LastCertifiedState{}).(*LastCertifiedState)
	if !exists {
		s.crit(ErrNoGenesis)
	}

	s.cache.LastCertifiedState = w
	return w
}

func (s *Store) GetLastCertifiedFrame() consensus.Frame {
	return s.GetLastCertifiedState().LastCertifiedFrame
}
