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
	"fmt"

	"github.com/0xsoniclabs/consensus/consensus"
)

// Genesis stores genesis state
type Genesis struct {
	Epoch      consensus.Epoch
	Validators *consensus.Validators
}

func (s *Store) ApplyGenesis(g *Genesis) error {
	if ok, _ := s.table.LastDecidedState.Has([]byte(dsKey)); ok {
		return fmt.Errorf("genesis already applied")
	}
	return s.SwitchGenesis(g)
}

func (s *Store) SwitchGenesis(g *Genesis) error {
	if g == nil {
		return fmt.Errorf("genesis config shouldn't be nil")
	}
	if g.Validators.Len() == 0 {
		return fmt.Errorf("genesis validators shouldn't be empty")
	}
	es := &EpochState{}
	ds := &LastDecidedState{}
	es.Validators = g.Validators
	es.Epoch = g.Epoch
	ds.LastDecidedFrame = consensus.FirstFrame - 1
	s.SetEpochState(es)
	s.SetLastDecidedState(ds)
	return nil
}
