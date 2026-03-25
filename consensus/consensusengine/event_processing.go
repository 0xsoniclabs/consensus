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
	_, frame := p.constructEventFrame(e)
	e.SetFrame(frame)

	return nil
}

// Process takes event into processing.
// Event order matter: parents first.
// All the event checkers must be launched.
// Process is not safe for concurrent use.
func (p *Orderer) Process(e consensus.Event) (err error) {
	selfParentFrame, frameIdx := p.constructEventFrame(e)
	if !p.config.SuppressFramePanic && e.Frame() != frameIdx {
		return ErrWrongFrame
	}
	if selfParentFrame == e.Frame() {
		return nil
	}
	p.store.AddBase(selfParentFrame, e)

	err = p.handleElection(selfParentFrame, e)
	if err != nil {
		p.crit(err)
	}
	return err
}

// handleElection iterates frames from selfParentFrame+1 to base.Frame(),
// calling ProcessBase for each. This matches lachesis-base's handleElection.
func (p *Orderer) handleElection(selfParentFrame consensus.Frame, base consensus.Event) error {
	for f := selfParentFrame + 1; f <= base.Frame(); f++ {
		decided, err := p.election.ProcessBase(BaseAndSlot{
			ID: base.ID(),
			Slot: Slot{
				Frame:     f,
				Validator: base.Creator(),
			},
		})
		if err != nil {
			return err
		}
		if decided == nil {
			continue
		}

		sealed, err := p.onFrameCertified(decided.Frame, decided.Leader)
		if err != nil {
			return err
		}
		if sealed {
			break
		}
		sealed, err = p.bootstrapElection()
		if err != nil {
			return err
		}
		if sealed {
			break
		}
	}
	return nil
}

// bootstrapElection calls processKnownBases until it returns nil.
func (p *Orderer) bootstrapElection() (bool, error) {
	for {
		decided, err := p.processKnownBases()
		if err != nil {
			return false, err
		}
		if decided == nil {
			break
		}

		sealed, err := p.onFrameCertified(decided.Frame, decided.Leader)
		if err != nil {
			return false, err
		}
		if sealed {
			return true, nil
		}
	}
	return false, nil
}

// processKnownBases re-processes all stored bases from LastCertifiedFrame+1
// upwards, calling ProcessBase for each. Used after node startup and after
// each decided frame.
func (p *Orderer) processKnownBases() (*ElectionRes, error) {
	lastCertifiedFrame := p.store.GetLastCertifiedFrame()
	for f := lastCertifiedFrame + 1; ; f++ {
		frameBases := p.store.GetFrameBases(f)
		if len(frameBases) == 0 {
			break
		}
		for _, base := range frameBases {
			decided, err := p.election.ProcessBase(BaseAndSlot{
				ID: base.BaseHash,
				Slot: Slot{
					Frame:     f,
					Validator: base.ValidatorID,
				},
			})
			if err != nil {
				return nil, err
			}
			if decided != nil {
				return decided, nil
			}
		}
	}
	return nil, nil
}

// stronglyReachableByQuorum returns true if event is strongly reachable by
// 2/3W bases on specified frame.
func (p *Orderer) stronglyReachableByQuorum(e consensus.Event, f consensus.Frame) bool {
	observedCounter := p.store.GetValidators().NewCounter()
	for _, baseDescriptor := range p.store.GetFrameBases(f) {
		if p.dagIndex.StronglyReach(e.ID(), baseDescriptor.BaseHash) {
			observedCounter.CountVoteByID(baseDescriptor.ValidatorID)
		}
		if observedCounter.QuorumReached() {
			break
		}
	}
	return observedCounter.QuorumReached()
}

// constructEventFrame calculates the frame for an event. It starts from
// selfParentFrame and loops, incrementing frame as long as the event is
// strongly-reachable by a quorum of bases on that frame.
func (p *Orderer) constructEventFrame(e consensus.Event) (selfParentFrame, frame consensus.Frame) {
	if e.SelfParent() == nil {
		return 0, 1
	}
	selfParentFrame = p.Input.GetEvent(*e.SelfParent()).Frame()
	frame = selfParentFrame
	for p.stronglyReachableByQuorum(e, frame) {
		frame++
	}
	return selfParentFrame, frame
}
