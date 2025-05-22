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

type ()

type electionB struct {
	validators *consensus.Validators

	stronglyReach StronglyReachFn
	getFrameBases GetFrameBasesFn

	// Base:Frame -> Voter:Layer -> Voter:ValidatorID -> Voter:EventHash -> int32[len(validators)]
	vote map[consensus.Frame][]map[consensus.ValidatorID]map[consensus.EventHash][]int32
	// Frame x ValidatorIndex -> EventHash
	bases          map[consensus.Frame][]consensus.EventHash
	validatorIDMap map[consensus.ValidatorID]consensus.ValidatorIndex
	validatorCount consensus.Frame

	leaderDeliveryBuffer *leaderHeap
	frameToDeliver       consensus.Frame
}

func NewElectionB(
	frameToDeliver consensus.Frame,
	validators *consensus.Validators,
	stronglyReachFn StronglyReachFn,
	getFrameBases GetFrameBasesFn,
) *electionB {
	election := &electionB{
		stronglyReach: stronglyReachFn,
		getFrameBases: getFrameBases,
		validators:    validators,
	}
	election.ResetEpoch(frameToDeliver, validators)
	return election
}

func (el *electionB) ResetEpoch(frameToDeliver consensus.Frame, validators *consensus.Validators) {
	el.leaderDeliveryBuffer = NewLeaderHeap()
	el.frameToDeliver = frameToDeliver
	el.validators = validators
	el.vote = make(map[consensus.Frame][]map[consensus.ValidatorID]map[consensus.EventHash][]int32)
	el.bases = make(map[consensus.Frame][]consensus.EventHash)
	el.validatorCount = consensus.Frame(validators.Len())
	el.validatorIDMap = validators.Idxs()
}

// RegisterBase saves frame x validatorId -> baseHash mapping for all bases that will be voted upon.
func (el *electionB) RegisterBase(frame consensus.Frame, validatorID consensus.ValidatorID, hash consensus.EventHash) {
	if el.frameToDeliver > frame {
		return
	}
	if _, ok := el.bases[frame]; !ok {
		el.bases[frame] = make([]consensus.EventHash, el.validatorCount)
	}
	validatorIdx := el.validatorIDMap[validatorID]
	el.bases[frame][validatorIdx] = hash

	// prepare the voting structure for base's frame.
	if _, ok := el.vote[frame]; !ok {
		el.vote[frame] = make([]map[consensus.ValidatorID]map[consensus.EventHash][]int32, 0)
	}
}

// GetVoters fetches all voters voting TO a specific _frame_, FROM a specific _layer_.
// As this method will exclusively be invoked from at least layer+1, all relevant voters are present in the vote structure.
func (el *electionB) GetVoters(frame consensus.Frame, layer consensus.Layer) []consensusstore.BaseDescriptor {
	voters := make([]consensusstore.BaseDescriptor, 0)
	for vID, maps := range el.vote[frame][layer] {
		for v := range maps {
			voters = append(voters, consensusstore.BaseDescriptor{BaseHash: v, ValidatorID: vID})
		}
	}
	return voters
}

func (el *electionB) ShouldVote(
	frame consensus.Frame,
	layer consensus.Layer,
	validatorID consensus.ValidatorID,
	voterHash consensus.EventHash,
) bool {
	if el.frameToDeliver > frame {
		return false
	}
	if _, ok := el.vote[frame]; !ok {
		// Noone voted for the frame yet
		return true
	}
	if len(el.vote[frame]) <= int(layer) {
		// Noone voted from the layer yet
		return true
	}
	if _, ok := el.vote[frame][layer][validatorID]; !ok {
		// Noone voted for the frame from the layer in the name of validatorID
		return true
	}
	// Someone already voted by this validator for this layer so stop.
	return false
}

func (el *electionB) Vote(
	frame consensus.Frame,
	layer consensus.Layer,
	validatorID consensus.ValidatorID,
	voterHash consensus.EventHash,
) ([]*leaderCertification, error) {
	if !el.ShouldVote(frame, layer, validatorID, voterHash) {
		return []*leaderCertification{}, nil
	}
	// Round completion property - i.e. fill all the voters gaps for your validator
	// E.g. A specific validator has voters for frame Y, Layers 1,2,3 and subsequently Frame Y, Layer 6 voter arrives
	// for the same validator, it will vote from the perspective of Frame Y, Layer 4,5,6 Voter
	for l := consensus.Layer(0); l < layer; l++ {
		// 1) If no voters ever voted to the frame base from the layer-1
		if len(el.vote[frame]) <= int(l) {
			el.Vote(frame, l, validatorID, voterHash)
			continue
		}
		// 2) If your validator never voted to the frame base from the layer-1
		if _, ok := el.vote[frame][l][validatorID]; !ok {
			el.Vote(frame, l, validatorID, voterHash)
		}
	}
	// If the Voter is first that votes from a specified layer, allocate the layer level structure
	if len(el.vote[frame]) <= int(layer) {
		el.vote[frame] = append(el.vote[frame], make(map[consensus.ValidatorID]map[consensus.EventHash][]int32))
	}

	el.vote[frame][layer][validatorID] = make(map[consensus.EventHash][]int32)
	el.vote[frame][layer][validatorID][voterHash] = initInt32WithConst(-1, int(el.validatorCount))
	voteVec := el.vote[frame][layer][validatorID][voterHash]
	validatorIdx := el.validatorIDMap[validatorID]
	//------------------------------------------------

	if layer == 0 {
		observedBases := el.observedBases(voterHash, frame)
		for _, observedBase := range observedBases {
			baseValidatorIdx := el.validatorIDMap[observedBase.ValidatorID]
			voteVec[baseValidatorIdx] = 1
		}
		mulInt32VecWithConst(voteVec, voteVec, int32(el.validators.GetWeightByIdx(validatorIdx)))
		return []*leaderCertification{}, nil
	}

	//------------------------------------------------/
	// layer is not 0
	// get these from layer-1
	observedVotersWeight := int32(0)
	for _, observedVoter := range el.GetVoters(frame, layer-1) {
		if !el.stronglyReach(voterHash, observedVoter.BaseHash) {
			continue
		}
		observedVoterValidatorIdx := el.validatorIDMap[observedVoter.ValidatorID]
		observedVotersWeight += int32(el.validators.GetWeightByIdx(observedVoterValidatorIdx))
		addInt32Vecs(voteVec, voteVec, el.vote[frame][layer-1][observedVoter.ValidatorID][observedVoter.BaseHash])
	}
	if el.certify(frame, voteVec, observedVotersWeight) {
		leaders := el.leaderDeliveryBuffer.getDeliveryReadyLeaders(el.frameToDeliver)
		el.frameToDeliver += consensus.Frame(len(leaders))
		return leaders, nil
	}
	normalizeInt32Vec(voteVec, voteVec)
	mulInt32VecWithConst(voteVec, voteVec, int32(el.validators.GetWeightByIdx(validatorIdx)))
	return []*leaderCertification{}, nil
}

func (el *electionB) certify(frame consensus.Frame, aggregationMatr []int32, observedBasesWeight int32) bool {
	// Q = ceil((4*TotalValidatorWeight - 3*observedBasesWeight)/3)
	// numerator (Q_0) can exceed the int32 limits before division
	Q_0 := 4*int64(el.validators.TotalWeight()) - 3*int64(observedBasesWeight)
	Q := int32((Q_0 + 3 - 1) / 3)

	yesDecisions := boolMaskInt32Vec(aggregationMatr, func(x int32) bool { return x >= Q })
	noDecisions := boolMaskInt32Vec(aggregationMatr, func(x int32) bool { return x <= -Q })
	for _, candidateValidator := range el.validators.SortedIDs() {
		validatorIdx := el.validatorIDMap[candidateValidator]
		if yesDecisions[validatorIdx] {

			heap.Push(el.leaderDeliveryBuffer, &leaderCertification{frame, el.bases[frame][validatorIdx]})
			el.cleanupCertifiedFrame(frame)
			return true
		}
		if !noDecisions[validatorIdx] {
			return false
		}
	}
	return false
}

func (el *electionB) observedBases(base consensus.EventHash, frame consensus.Frame) []consensusstore.BaseDescriptor {
	observedBases := make([]consensusstore.BaseDescriptor, 0, el.validators.Len())
	frameBases := el.getFrameBases(frame)
	for _, frameBase := range frameBases {
		if el.stronglyReach(base, frameBase.BaseHash) {
			observedBases = append(observedBases, frameBase)
		}
	}
	return observedBases
}

func (el *electionB) cleanupCertifiedFrame(frame consensus.Frame) {
	delete(el.vote, frame)
	delete(el.bases, frame)
}
