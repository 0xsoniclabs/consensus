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

// SetLastDecidedState save LastDecidedState.
// LastDecidedState is seldom read; so no cache.
func (s *Store) SetLastDecidedState(v *LastDecidedState) {
	s.cache.LastDecidedState = v

	s.set(s.table.LastDecidedState, []byte(dsKey), v)
}

// GetLastDecidedState returns stored LastDecidedState.
// State is seldom read; so no cache.
func (s *Store) GetLastDecidedState() *LastDecidedState {
	if s.cache.LastDecidedState != nil {
		return s.cache.LastDecidedState
	}

	w, exists := s.get(s.table.LastDecidedState, []byte(dsKey), &LastDecidedState{}).(*LastDecidedState)
	if !exists {
		s.crit(ErrNoGenesis)
	}

	s.cache.LastDecidedState = w
	return w
}

func (s *Store) GetLastDecidedFrame() consensus.Frame {
	return s.GetLastDecidedState().LastDecidedFrame
}
