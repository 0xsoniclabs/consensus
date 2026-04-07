// Copyright (c) 2026 Sonic Operations Ltd

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
	if ok, _ := s.table.LastCertifiedState.Has([]byte(csKey)); ok {
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
	ds := &LastCertifiedState{}
	es.Validators = g.Validators
	es.Epoch = g.Epoch
	ds.LastCertifiedFrame = consensus.FirstFrame - 1
	s.SetEpochState(es)
	s.SetLastCertifiedState(ds)
	return nil
}
