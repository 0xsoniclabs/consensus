// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensusengine

import (
	"github.com/0xsoniclabs/consensus/consensus"
)

// onFrameCertified moves LastCertifiedFrameN to frame.
// It includes: moving current certified frame, txs ordering and execution, epoch sealing.
func (p *Orderer) onFrameCertified(frame consensus.Frame, leader consensus.EventHash) (bool, error) {
	// new checkpoint
	var newValidators *consensus.Validators
	if p.callback.ApplyLeader != nil {
		newValidators = p.callback.ApplyLeader(frame, leader)
	}

	lastCertifiedState := *p.store.GetLastCertifiedState()
	if newValidators != nil {
		lastCertifiedState.LastCertifiedFrame = consensus.FirstFrame - 1
		err := p.sealEpoch(newValidators)
		if err != nil {
			return true, err
		}
		p.election.ResetEpoch(consensus.FirstFrame, newValidators)
	} else {
		lastCertifiedState.LastCertifiedFrame = frame
	}
	p.store.SetLastCertifiedState(&lastCertifiedState)
	return newValidators != nil, nil
}

func (p *Orderer) resetEpochStore(newEpoch consensus.Epoch) error {
	err := p.store.DropEpochDB()
	if err != nil {
		return err
	}
	err = p.store.OpenEpochDB(newEpoch)
	if err != nil {
		return err
	}

	if p.callback.EpochDBLoaded != nil {
		p.callback.EpochDBLoaded(newEpoch)
	}
	return nil
}

func (p *Orderer) sealEpoch(newValidators *consensus.Validators) error {
	// new PrevEpoch state
	epochState := *p.store.GetEpochState()
	epochState.Epoch++
	epochState.Validators = newValidators
	p.store.SetEpochState(&epochState)

	return p.resetEpochStore(epochState.Epoch)
}
