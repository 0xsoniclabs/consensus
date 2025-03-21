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
