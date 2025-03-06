// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package abft

import (
	"fmt"

	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/inter/pos"
)

// Genesis stores genesis state
type Genesis struct {
	Epoch      idx.Epoch
	Validators *pos.Validators
}

// ApplyGenesis writes initial state.
func (s *Store) ApplyGenesis(g *Genesis) error {
	if g == nil {
		return fmt.Errorf("genesis config shouldn't be nil")
	}
	if g.Validators.Len() == 0 {
		return fmt.Errorf("genesis validators shouldn't be empty")
	}
	if ok, _ := s.table.LastDecidedState.Has([]byte(dsKey)); ok {
		return fmt.Errorf("genesis already applied")
	}

	s.applyGenesis(g.Epoch, g.Validators)
	return nil
}

// applyGenesis switches epoch state to a new empty epoch.
func (s *Store) applyGenesis(epoch idx.Epoch, validators *pos.Validators) {
	es := &EpochState{}
	ds := &LastDecidedState{}

	es.Validators = validators
	es.Epoch = epoch
	ds.LastDecidedFrame = FirstFrame - 1

	s.SetEpochState(es)
	s.SetLastDecidedState(ds)

}
