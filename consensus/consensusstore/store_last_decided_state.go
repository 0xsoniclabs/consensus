// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensusstore

import "github.com/0xsoniclabs/consensus/consensus"

const dsKey = "d"

// LastCertifiedState is for persistent storing.
type LastCertifiedState struct {
	// fields can change only after a frame is certified
	LastCertifiedFrame consensus.Frame
}

// SetLastCertifiedState save LastCertifiedState.
// LastCertifiedState is seldom read; so no cache.
func (s *Store) SetLastCertifiedState(v *LastCertifiedState) {
	s.cache.LastCertifiedState = v

	s.set(s.table.LastCertifiedState, []byte(dsKey), v)
}

// GetLastCertifiedState returns stored LastCertifiedState.
// State is seldom read; so no cache.
func (s *Store) GetLastCertifiedState() *LastCertifiedState {
	if s.cache.LastCertifiedState != nil {
		return s.cache.LastCertifiedState
	}

	w, exists := s.get(s.table.LastCertifiedState, []byte(dsKey), &LastCertifiedState{}).(*LastCertifiedState)
	if !exists {
		s.crit(ErrNoGenesis)
	}

	s.cache.LastCertifiedState = w
	return w
}

func (s *Store) GetLastCertifiedFrame() consensus.Frame {
	return s.GetLastCertifiedState().LastCertifiedFrame
}
