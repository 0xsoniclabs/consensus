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
	"github.com/pkg/errors"

	"github.com/0xsoniclabs/consensus/inter/dag"
	"github.com/0xsoniclabs/consensus/inter/idx"
)

var (
	ErrWrongFrame = errors.New("claimed frame mismatched with calculated")
)

// Build fills consensus-related fields: Frame, IsRoot
// returns error if event should be dropped
func (p *Orderer) Build(e dag.MutableEvent) error {
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
func (p *Orderer) Process(e dag.Event) (err error) {
	err, selfParentFrame := p.checkAndSaveEvent(e)
	if err != nil {
		return err
	}

	if selfParentFrame == e.Frame() {
		return nil
	}
	if err := p.handleElection(e); err != nil {
		// election doesn't fail under normal circumstances
		// storage is in an inconsistent state
		p.crit(err)
	}
	return err
}

// Process event that's been built locally
func (p *Orderer) ProcessLocalEvent(e dag.Event) (err error) {
	selfParentFrame := p.getSelfParentFrame(e)
	if selfParentFrame == e.Frame() {
		return nil
	}
	// It's a root
	p.store.AddRoot(e)
	if err := p.handleElection(e); err != nil {
		// election doesn't fail under normal circumstances
		// storage is in an inconsistent state
		p.crit(err)
	}
	return err
}

// checkAndSaveEvent checks consensus-related fields: Frame, IsRoot
func (p *Orderer) checkAndSaveEvent(e dag.Event) (error, idx.Frame) {
	// check frame & isRoot
	selfParentFrame, frameIdx := p.calcFrameIdx(e)
	if !p.config.SuppressFramePanic && e.Frame() != frameIdx {
		return ErrWrongFrame, 0
	}

	if selfParentFrame != frameIdx {
		p.store.AddRoot(e)
	}
	return nil, selfParentFrame
}

// calculates Atropos election for the root, calls p.onFrameDecided if election was decided
func (p *Orderer) handleElection(root dag.Event) error {
	decisions, err := p.election.VoteAndAggregate(root.Frame(), root.Creator(), root.ID())
	if err != nil {
		return err
	}
	for _, atroposDecision := range decisions {
		sealed, err := p.onFrameDecided(atroposDecision.Frame, atroposDecision.AtroposHash)
		if err != nil {
			return err
		}
		if sealed {
			return nil
		}
	}
	return nil
}

func (p *Orderer) bootstrapElection() error {
	for frame := p.store.GetLastDecidedFrame() + 1; ; frame++ {
		frameRoots := p.store.GetFrameRoots(frame)
		if len(frameRoots) == 0 {
			break
		}
		for _, root := range frameRoots {
			decisions, err := p.election.VoteAndAggregate(frame, root.ValidatorID, root.RootHash)
			if err != nil {
				return err
			}
			for _, atroposDecision := range decisions {
				sealed, err := p.onFrameDecided(atroposDecision.Frame, atroposDecision.AtroposHash)
				if err != nil {
					return err
				}
				if sealed {
					return nil
				}
			}
		}
	}
	return nil
}

// forklessCausedByQuorumOn returns true if event is forkless caused by 2/3W roots on specified frame
func (p *Orderer) forklessCausedByQuorumOn(e dag.Event, f idx.Frame) bool {
	observedCounter := p.store.GetValidators().NewCounter()
	// check "observing" prev roots only if called by creator, or if creator has marked that event as root
	for _, it := range p.store.GetFrameRoots(f) {
		if p.dagIndex.ForklessCause(e.ID(), it.RootHash) {
			observedCounter.Count(it.ValidatorID)
		}
		if observedCounter.HasQuorum() {
			break
		}
	}
	return observedCounter.HasQuorum()
}

// calcFrameIdx is not safe for concurrent use.
func (p *Orderer) calcFrameIdx(e dag.Event) (selfParentFrame, frame idx.Frame) {
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

func (p *Orderer) getSelfParentFrame(e dag.Event) idx.Frame {
	if e.SelfParent() == nil {
		return 0
	}
	return p.Input.GetEvent(*e.SelfParent()).Frame()
}
