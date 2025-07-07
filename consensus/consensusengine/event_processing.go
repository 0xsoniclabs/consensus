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
	"slices"

	"github.com/pkg/errors"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensusstore"
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

	_, frame, _ := p.calcFrameIdx(e)
	e.SetFrame(frame)

	return nil
}

// Process takes event into processing.
// Event order matter: parents first.
// All the event checkers must be launched.
// Process is not safe for concurrent use.
func (p *Orderer) Process(e consensus.Event) (err error) {
	selfParentFrame, srVector, err := p.checkAndSaveEvent(e)
	if err != nil {
		return err
	}

	if selfParentFrame == e.Frame() {
		return nil
	}
	if srVector == nil {
		srVector = p.srVector(e.ID(), e.Frame()-1)
	}
	if _, err := p.runElectionOnBase(e.Frame(), e.Creator(), e.ID(), srVector); err != nil {
		// election doesn't fail under normal circumstances
		// storage is in an inconsistent state
		p.crit(err)
	}
	return err
}

// checkAndSaveEvent checks consensus-related fields: Frame, IsBase
func (p *Orderer) checkAndSaveEvent(e consensus.Event) (consensus.Frame, map[consensus.ValidatorID]bool, error) {
	// check frame & isBase
	selfParentFrame, frameIdx, srVector := p.calcFrameIdx(e)
	if !p.config.SuppressFramePanic && e.Frame() != frameIdx {
		return 0, nil, ErrWrongFrame
	}

	if selfParentFrame != frameIdx {
		p.store.AddBase(e)
	}
	return selfParentFrame, srVector, nil
}

// runElectionOnBase runs Leader election for the base and triggers block closure callbacks if election was certified
func (p *Orderer) runElectionOnBase(frame consensus.Frame, validatorID consensus.ValidatorID, baseHash consensus.EventHash, srVector map[consensus.ValidatorID]bool) (bool, error) {
	certifications, err := p.election.VoteAndAggregate(frame, validatorID, baseHash, srVector)
	if err != nil {
		return false, err
	}
	for _, leaderCertification := range certifications {
		sealed, err := p.onFrameCertified(leaderCertification.Frame, leaderCertification.LeaderHash)
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
	for frame := p.store.GetLastCertifiedFrame() + 1; ; frame++ {
		frameBases := p.store.GetFrameBases(frame)
		if len(frameBases) == 0 {
			break
		}
		for _, base := range frameBases {
			srVector := p.srVector(base.BaseHash, frame-1)
			sealed, err := p.runElectionOnBase(frame, base.ValidatorID, base.BaseHash, srVector)
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

// stronglyReachableByQuorum returns true if event is strongly reachable by 2/3W bases on specified frame
func (p *Orderer) stronglyReachableByQuorum(e consensus.Event, f consensus.Frame) bool {
	validators := p.store.GetValidators()
	reachableCounter := validators.NewCounter()
	// check "observing" prev bases only if called by creator, or if creator has marked that event as base
	frameBases := p.store.GetFrameBases(f)
	slices.SortFunc(frameBases, func(a, b consensusstore.BaseDescriptor) int {
		return int(validators.GetWeightByIdx(validators.GetIdx(b.ValidatorID))) - int(validators.GetWeightByIdx(validators.GetIdx(a.ValidatorID)))
	})

	for _, it := range frameBases {
		if p.dagIndex.StronglyReach(e.ID(), it.BaseHash) {
			reachableCounter.CountVoteByID(it.ValidatorID)
		} else {
			reachableCounter.CountAntiVoteByID(it.ValidatorID)
		}
		if reachableCounter.HasAntiQuorum() {
			break
		}
		if reachableCounter.HasQuorum() {
			break
		}
	}
	return reachableCounter.HasQuorum()
}

// stronglyReachableByQuorum returns true if event is strongly reachable by 2/3W bases on specified frame
func (p *Orderer) stronglyReachableByQuorumVec(e consensus.Event, f consensus.Frame) (bool, map[consensus.ValidatorID]bool) {
	validators := p.store.GetValidators()
	reachableCounter := validators.NewCounter()
	srVector := make(map[consensus.ValidatorID]bool, validators.Len())
	// check "observing" prev bases only if called by creator, or if creator has marked that event as base
	for _, it := range p.store.GetFrameBases(f) {
		if p.dagIndex.StronglyReach(e.ID(), it.BaseHash) {
			reachableCounter.CountVoteByID(it.ValidatorID)
			srVector[it.ValidatorID] = true
		}
	}
	return reachableCounter.HasQuorum(), srVector
}

// stronglyReachableByQuorum returns true if event is strongly reachable by 2/3W bases on specified frame
func (p *Orderer) srVector(e consensus.EventHash, f consensus.Frame) map[consensus.ValidatorID]bool {
	srVector := make(map[consensus.ValidatorID]bool, p.store.GetValidators().Len())
	// check "observing" prev bases only if called by creator, or if creator has marked that event as base
	for _, it := range p.store.GetFrameBases(f) {
		if p.dagIndex.StronglyReach(e, it.BaseHash) {
			srVector[it.ValidatorID] = true
		}
	}
	return srVector
}

// calcFrameIdx is not safe for concurrent use.
func (p *Orderer) calcFrameIdx(e consensus.Event) (selfParentFrame, frame consensus.Frame, srVector map[consensus.ValidatorID]bool) {
	if e.SelfParent() == nil {
		return 0, 1, map[consensus.ValidatorID]bool{}
	}
	selfParentFrame = p.Input.GetEvent(*e.SelfParent()).Frame()
	frame = selfParentFrame
	for _, parent := range e.Parents() {
		frame = max(frame, p.Input.GetEvent(parent).Frame())
	}

	reachable := false
	if e.Frame() == frame {
		reachable = p.stronglyReachableByQuorum(e, frame)
		srVector = nil
	} else {
		reachable, srVector = p.stronglyReachableByQuorumVec(e, frame)
	}
	if reachable {
		frame++
	}
	return selfParentFrame, frame, srVector
}
