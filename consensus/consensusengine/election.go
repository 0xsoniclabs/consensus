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
	"container/heap"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensusstore"
)

type (
	StronglyReachFn func(a consensus.EventHash, b consensus.EventHash) bool
	GetFrameBasesFn func(f consensus.Frame) []consensusstore.BaseDescriptor
)

type leaderCertification struct {
	Frame      consensus.Frame
	LeaderHash consensus.EventHash
}

type baseVoteContext struct {
	frameToDeliverOffset consensus.Frame
	voteMatrix           []int32
}

type election struct {
	validators *consensus.Validators

	stronglyReaches StronglyReachFn
	getFrameBases   GetFrameBasesFn

	vote           map[consensus.Frame][]map[consensus.EventHash]*baseVoteContext
	validatorIDMap map[consensus.ValidatorID]consensus.ValidatorIndex
	validatorCount consensus.Frame

	leaderDeliveryBuffer *leaderHeap
	frameToDeliver       consensus.Frame
}

func NewElection(
	frameToDeliver consensus.Frame,
	validators *consensus.Validators,
	stronglyReachFn StronglyReachFn,
	getFrameBases GetFrameBasesFn,
) *election {
	election := &election{
		stronglyReaches: stronglyReachFn,
		getFrameBases:   getFrameBases,
		validators:      validators,
	}
	election.ResetEpoch(frameToDeliver, validators)
	return election
}

func (el *election) ResetEpoch(frameToDeliver consensus.Frame, validators *consensus.Validators) {
	el.leaderDeliveryBuffer = NewLeaderHeap()
	el.frameToDeliver = frameToDeliver
	el.validators = validators
	el.vote = make(map[consensus.Frame][]map[consensus.EventHash]*baseVoteContext)
	el.validatorCount = consensus.Frame(validators.Len())
	el.validatorIDMap = validators.Idxs()
}

func (el *election) VoteAndAggregate(
	frame consensus.Frame,
	validatorId consensus.ValidatorID,
	baseHash consensus.EventHash,
) ([]*leaderCertification, error) {
	if el.isAlreadyCertified(frame) {
		return []*leaderCertification{}, nil
	}
	validatorIdx := el.validatorIDMap[validatorId]
	el.prepareNewElectorBase(frame, validatorIdx, baseHash)
	if frame <= el.frameToDeliver {
		return []*leaderCertification{}, nil
	}

	aggregationMatrix := make([]int32, (frame-el.frameToDeliver-1)*el.validatorCount, (frame-el.frameToDeliver)*el.validatorCount)
	directVoteVector := initInt32WithConst(-1, int(el.validatorCount))

	reachableBases := el.reachableBases(baseHash, frame-1)
	reachableBasesWeight := int32(0)

	for _, reachableBase := range reachableBases {
		validatorIdx := el.validatorIDMap[reachableBase.ValidatorID]
		directVoteVector[validatorIdx] = 1
		reachableBasesWeight += int32(el.validators.GetWeightByIdx(validatorIdx))

		if el.vote[frame-1][validatorIdx] != nil {
			if baseContext, ok := el.vote[frame-1][validatorIdx][reachableBase.BaseHash]; ok {
				nonDeliveredFramesOffset := (el.frameToDeliver - baseContext.frameToDeliverOffset) * el.validatorCount
				addInt32Vecs(aggregationMatrix, aggregationMatrix, baseContext.voteMatrix[nonDeliveredFramesOffset:])
			}
		}
	}

	deliveryReadyLeaders := el.certify(frame, aggregationMatrix, reachableBasesWeight)

	// Prepare matrix for future aggregations
	normalizeInt32Vec(aggregationMatrix, aggregationMatrix)
	aggregationMatrix = append(aggregationMatrix, directVoteVector...)
	mulInt32VecWithConst(aggregationMatrix, aggregationMatrix, int32(el.validators.GetWeightByIdx(validatorIdx)))
	el.vote[frame][validatorIdx][baseHash].voteMatrix = aggregationMatrix

	return deliveryReadyLeaders, nil
}

func (el *election) certify(aggregatingFrame consensus.Frame, aggregationMatr []int32, reachableBasesWeight int32) []*leaderCertification {
	// Q = ceil((4*TotalValidatorWeight - 3*reachableBasesWeight)/3)
	// numerator (Q_0) can exceed the int32 limits before division
	Q_0 := 4*int64(el.validators.TotalWeight()) - 3*int64(reachableBasesWeight)
	Q := int32((Q_0 + 3 - 1) / 3)
	yesDecisions := boolMaskInt32Vec(aggregationMatr, func(x int32) bool { return x >= Q })
	noDecisions := boolMaskInt32Vec(aggregationMatr, func(x int32) bool { return x <= -Q })

	for frame := range el.vote {
		if el.isAlreadyCertified(frame) || frame >= aggregatingFrame-1 {
			continue
		}

		for _, candidateValidator := range el.validators.SortedIDs() {
			validatorIdx := el.validatorIDMap[candidateValidator]
			voteMatrixOffset := (frame-el.frameToDeliver)*el.validatorCount + consensus.Frame(validatorIdx)

			if yesDecisions[voteMatrixOffset] {
				leaderHash := el.elect(frame, candidateValidator)
				heap.Push(el.leaderDeliveryBuffer, &leaderCertification{frame, leaderHash})
				break
			}

			if !noDecisions[voteMatrixOffset] {
				break
			}
		}
	}

	deliveryReadyLeaders := el.leaderDeliveryBuffer.getDeliveryReadyLeaders(el.frameToDeliver)
	for _, leaderCertification := range deliveryReadyLeaders {
		el.cleanupCertifiedFrame(leaderCertification.Frame)
		el.frameToDeliver++
	}
	return deliveryReadyLeaders
}

// elect picks the final leader event once its frame and validator number have been finalized
// by the "upper frame" base votes'. This is trivial in case of non-equivocating events as such
// bases are uniquely identified by (frame, validator).
// In the case of an equivocation, a tiebreaker algorithm has to be run.
func (el *election) elect(frame consensus.Frame, validatorCandidate consensus.ValidatorID) consensus.EventHash {
	validatorIdx := el.validatorIDMap[validatorCandidate]
	candidateMap := el.vote[frame][validatorIdx]
	leaderHash := consensus.EventHash{}
	for hash := range candidateMap {
		leaderHash = hash
	}
	// tiebreaker can simply pick the first encountered base that is strongly reachable by any event.
	// It is easiest to look for any vote (strongly reach) by frame + 1 bases.
	// Due to strongly reach semantics, only one strongly-reachable base can exist with specified frame and validator number.
	if len(candidateMap) > 1 {
		judgeBases := el.getFrameBases(frame + 1)
		for leaderCandidateHash := range candidateMap {
			for _, judge := range judgeBases {
				if el.stronglyReaches(judge.BaseHash, leaderCandidateHash) {
					return leaderCandidateHash
				}
			}
		}
	}

	return leaderHash
}

func (el *election) reachableBases(base consensus.EventHash, frame consensus.Frame) []consensusstore.BaseDescriptor {
	reachableBases := make([]consensusstore.BaseDescriptor, 0, el.validators.Len())
	frameBases := el.getFrameBases(frame)
	for _, frameBase := range frameBases {
		if el.stronglyReaches(base, frameBase.BaseHash) {
			reachableBases = append(reachableBases, frameBase)
		}
	}
	return reachableBases
}

func (el *election) prepareNewElectorBase(frame consensus.Frame, validatorIdx consensus.ValidatorIndex, base consensus.EventHash) {
	if _, ok := el.vote[frame]; !ok {
		el.vote[frame] = make([]map[consensus.EventHash]*baseVoteContext, el.validatorCount)
	}

	if el.vote[frame][validatorIdx] == nil {
		el.vote[frame][validatorIdx] = make(map[consensus.EventHash]*baseVoteContext)
	}

	el.vote[frame][validatorIdx][base] = &baseVoteContext{frameToDeliverOffset: el.frameToDeliver}
}

func (el *election) isAlreadyCertified(frame consensus.Frame) bool {
	return frame < el.frameToDeliver || el.leaderDeliveryBuffer.isCertificationBuffered(frame)
}

func (el *election) cleanupCertifiedFrame(frame consensus.Frame) {
	delete(el.vote, frame)
}
