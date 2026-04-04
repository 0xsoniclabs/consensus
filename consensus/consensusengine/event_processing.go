// Copyright (c) 2026 Sonic Operations Ltd
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
	_, frame, _ := p.constructEventFrame(e)
	e.SetFrame(frame)

	return nil
}

// Process takes event into processing.
// Event order matter: parents first.
// All the event checkers must be launched.
// Process is not safe for concurrent use.
func (p *Orderer) Process(e consensus.Event) (err error) {
	// Verify event's frame and proceed with election if it is a base
	selfParentFrame, frameIdx, stronglyReachableBases := p.constructEventFrame(e)
	if !p.config.SuppressFramePanic && e.Frame() != frameIdx {
		return ErrWrongFrame
	}
	if selfParentFrame == e.Frame() {
		return nil
	}
	p.store.AddBase(e)
	// Gather strongly reachable bases if they haven't been memoized by the frame construction
	if stronglyReachableBases == nil {
		stronglyReachableBases = p.stronglyReachableBases(e.ID(), e.Frame()-1)
	}
	if _, err := p.runElectionOnBase(e.Frame(), e.Creator(), e.ID(), stronglyReachableBases); err != nil {
		// election doesn't fail under normal circumstances
		// storage is in an inconsistent state
		p.crit(err)
	}
	return err
}

// runElectionOnBase runs Leader election for the base and triggers block closure callbacks if election was certified
func (p *Orderer) runElectionOnBase(frame consensus.Frame, validatorID consensus.ValidatorID, baseHash consensus.EventHash, stronglyReachableBases []*consensus.BaseDescriptor) (bool, error) {
	certifications, err := p.election.VoteAndAggregate(frame, validatorID, baseHash, stronglyReachableBases)
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
			sealed, err := p.runElectionOnBase(frame, base.ValidatorID, base.BaseHash, p.stronglyReachableBases(base.BaseHash, frame-1))
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
	frameBases := p.store.GetFrameBases(f)
	// Traverse the frame bases in a descending order of their respective validator stake to reach the quorum/AntiQuorum with the least amount of StronglyReach calls
	slices.SortFunc(frameBases, func(a, b consensus.BaseDescriptor) int {
		return int(validators.GetWeightByIdx(validators.GetIdx(b.ValidatorID))) - int(validators.GetWeightByIdx(validators.GetIdx(a.ValidatorID)))
	})
	for _, baseDescriptor := range frameBases {
		if p.dagIndex.StronglyReach(e.ID(), baseDescriptor.BaseHash) {
			reachableCounter.CountVoteByID(baseDescriptor.ValidatorID)
		} else {
			reachableCounter.CountAntiVoteByID(baseDescriptor.ValidatorID)
		}
		if reachableCounter.AntiQuorumReached() || reachableCounter.QuorumReached() {
			break
		}
	}
	return reachableCounter.QuorumReached()
}

// stronglyReachableByQuorumMemoize extends the standard stronglyReachableByQuorum by memoizing the strongly reachable bases.
// It performs worse than its vanilla counterpart as it checks the SR relation for all the bases - even when quorum is reached mid-iteration.
func (p *Orderer) stronglyReachableByQuorumMemoize(e consensus.Event, f consensus.Frame) (bool, []*consensus.BaseDescriptor) {
	reachableCounter := p.store.GetValidators().NewCounter()
	frameBases := p.store.GetFrameBases(f)
	memoizedDescriptors := make([]*consensus.BaseDescriptor, 0)
	for idx, baseDescriptor := range frameBases {
		if p.dagIndex.StronglyReach(e.ID(), baseDescriptor.BaseHash) {
			reachableCounter.CountVoteByID(baseDescriptor.ValidatorID)
			memoizedDescriptors = append(memoizedDescriptors, &frameBases[idx])
		}
	}
	return reachableCounter.QuorumReached(), memoizedDescriptors
}

func (p *Orderer) stronglyReachableBases(eventHash consensus.EventHash, f consensus.Frame) []*consensus.BaseDescriptor {
	frameBases := p.store.GetFrameBases(f)
	memoizedDescriptors := make([]*consensus.BaseDescriptor, 0)
	for idx, baseDescriptor := range p.store.GetFrameBases(f) {
		if p.dagIndex.StronglyReach(eventHash, baseDescriptor.BaseHash) {
			memoizedDescriptors = append(memoizedDescriptors, &frameBases[idx])
		}
	}
	return memoizedDescriptors
}

// calcFrameIdx is not safe for concurrent use.
func (p *Orderer) constructEventFrame(e consensus.Event) (selfParentFrame, frame consensus.Frame, memoizedDescriptors []*consensus.BaseDescriptor) {
	if e.SelfParent() == nil {
		return 0, 1, []*consensus.BaseDescriptor{}
	}
	selfParentFrame = p.Input.GetEvent(*e.SelfParent()).Frame()
	frame = selfParentFrame
	for _, parent := range e.Parents() {
		frame = max(frame, p.Input.GetEvent(parent).Frame())
	}
	var reachable bool
	// Memoize the strongly reachable calculations only if they can be reused by the election algorithm:
	// The event comes in with an already assigned frame by its creator, where providing an incorrect frame invalidates the event.
	// To this end, the algorithm utilizes the provided frame to predict whether the SR should be memoized and can be
	// reused by the upcoming election algorithm. The calculations can only be reused if the events frame ends up being one
	// above the maximum frame of its parents.
	if e.Frame() == frame+1 {
		reachable, memoizedDescriptors = p.stronglyReachableByQuorumMemoize(e, frame)
	} else {
		reachable = p.stronglyReachableByQuorum(e, frame)
		memoizedDescriptors = nil
	}
	if reachable {
		frame++
	}
	return selfParentFrame, frame, memoizedDescriptors
}
