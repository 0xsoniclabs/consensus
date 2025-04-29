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
	"github.com/pkg/errors"

	"github.com/0xsoniclabs/consensus/consensus"
)

var (
	ErrWrongFrame = errors.New("claimed frame mismatched with calculated")
)

// Build fills consensus-related fields: Frame, IsBase
// returns error if event should be dropped
func (p *Orderer) Build(e consensus.MutableEvent) error {
	// sanity check
	if e.Epoch() != p.store.GetEpoch() {
		p.crit(errors.New("event has wrong epoch"))
	}
	if !p.store.GetValidators().Exists(e.Creator()) {
		p.crit(errors.New("event wasn't created by an existing validator"))
	}

	_, frame := p.calcFrameIdx(e)
	e.SetFrame(frame)

	return nil
}

// Process takes event into processing.
// Event order matter: parents first.
// All the event checkers must be launched.
// Process is not safe for concurrent use.
func (p *Orderer) Process(e consensus.Event) (err error) {
	selfParentFrame, err := p.checkAndSaveEvent(e)
	if err != nil {
		return err
	}

	if selfParentFrame == e.Frame() {
		return nil
	}
	if _, err := p.runElectionOnBase(e.Frame(), e.Creator(), e.ID()); err != nil {
		// election doesn't fail under normal circumstances
		// storage is in an inconsistent state
		p.crit(err)
	}
	return err
}

// checkAndSaveEvent checks consensus-related fields: Frame, IsBase
func (p *Orderer) checkAndSaveEvent(e consensus.Event) (consensus.Frame, error) {
	// check frame & isBase
	selfParentFrame, frameIdx := p.calcFrameIdx(e)
	if !p.config.SuppressFramePanic && e.Frame() != frameIdx {
		return 0, ErrWrongFrame
	}

	if selfParentFrame != frameIdx {
		p.store.AddBase(e)
	}
	return selfParentFrame, nil
}

// runElectionOnBase runs Atropos election for the base and triggers block closure callbacks if election was decided
func (p *Orderer) runElectionOnBase(frame consensus.Frame, validatorID consensus.ValidatorID, baseHash consensus.EventHash) (bool, error) {
	decisions, err := p.election.VoteAndAggregate(frame, validatorID, baseHash)
	if err != nil {
		return false, err
	}
	for _, atroposDecision := range decisions {
		sealed, err := p.onFrameDecided(atroposDecision.Frame, atroposDecision.AtroposHash)
		if err != nil {
			return false, err
		}
		if sealed {
			return true, nil
		}
	}
	return false, nil
}

func (p *Orderer) bootstrapElection() error {
	for frame := p.store.GetLastDecidedFrame() + 1; ; frame++ {
		frameBases := p.store.GetFrameBases(frame)
		if len(frameBases) == 0 {
			break
		}
		for _, base := range frameBases {
			sealed, err := p.runElectionOnBase(frame, base.ValidatorID, base.BaseHash)
			if err != nil {
				return err
			}
			if sealed {
				return nil
			}
		}
	}
	return nil
}

// forklessCausedByQuorumOn returns true if event is forkless caused by 2/3W bases on specified frame
func (p *Orderer) forklessCausedByQuorumOn(e consensus.Event, f consensus.Frame) bool {
	observedCounter := p.store.GetValidators().NewCounter()
	// check "observing" prev bases only if called by creator, or if creator has marked that event as base
	for _, it := range p.store.GetFrameBases(f) {
		if p.dagIndex.ForklessCause(e.ID(), it.BaseHash) {
			observedCounter.CountVoteByID(it.ValidatorID)
		}
		if observedCounter.HasQuorum() {
			break
		}
	}
	return observedCounter.HasQuorum()
}

// calcFrameIdx is not safe for concurrent use.
func (p *Orderer) calcFrameIdx(e consensus.Event) (selfParentFrame, frame consensus.Frame) {
	if e.SelfParent() == nil {
		return 0, 1
	}
	selfParentFrame = p.Input.GetEvent(*e.SelfParent()).Frame()
	frame = selfParentFrame
	for _, parent := range e.Parents() {
		frame = max(frame, p.Input.GetEvent(parent).Frame())
	}

	if p.forklessCausedByQuorumOn(e, frame) {
		frame++
	}
	return selfParentFrame, frame
}

func (p *Orderer) getSelfParentFrame(e consensus.Event) consensus.Frame {
	if e.SelfParent() == nil {
		return 0
	}
	return p.Input.GetEvent(*e.SelfParent()).Frame()
}
