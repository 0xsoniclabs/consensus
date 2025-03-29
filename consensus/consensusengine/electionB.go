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
	"fmt"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensusstore"
)

type ()

type electionB struct {
	validators *consensus.Validators

	forklessCauses ForklessCauseFn
	getFrameRoots  GetFrameRootsFn

	// To:Frame -> From:Layer -> From:EventHash -> int32[len(validators)]
	voteB map[consensus.Frame][]map[consensus.EventHash][]int32
	// Frame x ValidatorIndex -> EventHash
	evidence       map[consensus.Frame][]consensus.EventHash
	vote           map[consensus.Frame][]map[consensus.EventHash]*rootVoteContext
	validatorIDMap map[consensus.ValidatorID]consensus.ValidatorIndex
	validatorCount consensus.Frame

	atroposDeliveryBuffer *atroposHeap
	frameToDeliver        consensus.Frame
}

func NewElectionB(
	frameToDeliver consensus.Frame,
	validators *consensus.Validators,
	forklessCauseFn ForklessCauseFn,
	getFrameRoots GetFrameRootsFn,
) *electionB {
	election := &electionB{
		forklessCauses: forklessCauseFn,
		getFrameRoots:  getFrameRoots,
		validators:     validators,
	}
	election.ResetEpoch(frameToDeliver, validators)
	return election
}

func (el *electionB) ResetEpoch(frameToDeliver consensus.Frame, validators *consensus.Validators) {
	el.atroposDeliveryBuffer = NewAtroposHeap()
	el.frameToDeliver = frameToDeliver
	el.validators = validators
	el.voteB = make(map[consensus.Frame][]map[consensus.EventHash][]int32)
	el.evidence = make(map[consensus.Frame][]consensus.EventHash)
	el.validatorCount = consensus.Frame(validators.Len())
	el.validatorIDMap = validators.Idxs()
}

func (el *electionB) RegisterRoot(frame consensus.Frame, validatorID consensus.ValidatorID, hash consensus.EventHash) {
	if el.frameToDeliver > frame {
		return
	}
	if _, ok := el.evidence[frame]; !ok {
		el.evidence[frame] = make([]consensus.EventHash, el.validatorCount)
	}
	validatorIdx := el.validatorIDMap[validatorID]
	el.evidence[frame][validatorIdx] = hash

	//------------------------------------------------
	if _, ok := el.voteB[frame]; !ok {
		if frame == 1 {
			fmt.Println()
		}
		el.voteB[frame] = make([]map[consensus.EventHash][]int32, 0)
	}
}

func (el *electionB) Vote(
	frame consensus.Frame,
	layer consensus.Layer,
	validatorID consensus.ValidatorID,
	voterHash consensus.EventHash,
) ([]*atroposDecision, error) {
	if el.frameToDeliver > frame {
		return []*atroposDecision{}, nil
	}
	if len(el.voteB[frame]) <= int(layer) {
		el.voteB[frame] = append(el.voteB[frame], make(map[consensus.EventHash][]int32))
	}
	el.voteB[frame][layer][voterHash] = make([]int32, el.validatorCount)
	voteVec := el.voteB[frame][layer][voterHash]
	validatorIdx := el.validatorIDMap[validatorID]
	//------------------------------------------------

	if layer == 0 {
		observedRoots := el.observedRoots(voterHash, frame)
		for _, observedRoot := range observedRoots {
			rootValidatorIdx := el.validatorIDMap[observedRoot.ValidatorID]
			voteVec[rootValidatorIdx] = 1
		}
		mulInt32VecWithConst(voteVec, voteVec, int32(el.validators.GetWeightByIdx(validatorIdx)))
		return []*atroposDecision{}, nil
	}

	//------------------------------------------------/
	// layer is not 0
	// get these from layer-1
	var observedVoters []consensusstore.RootDescriptor
	observedVoters = el.observedRoots(voterHash, frame+consensus.Frame(layer))
	observedVotersWeight := int32(0)
	for _, observedVoter := range observedVoters {
		observedVoterValidatorIdx := el.validatorIDMap[observedVoter.ValidatorID]
		observedVotersWeight += int32(el.validators.GetWeightByIdx(observedVoterValidatorIdx))
		if el.voteB[frame][layer-1][observedVoter.RootHash] == nil {
			fmt.Println("AAA")
		}
		if len(el.voteB[frame][layer-1][observedVoter.RootHash]) == 0 {
			continue
		}
		addInt32Vecs(voteVec, voteVec, el.voteB[frame][layer-1][observedVoter.RootHash])
	}
	if el.decideB(frame, voteVec, observedVotersWeight) {
		atropoi := el.atroposDeliveryBuffer.getDeliveryReadyAtropoi(el.frameToDeliver)
		el.frameToDeliver += consensus.Frame(len(atropoi))
		return atropoi, nil
	}
	normalizeInt32Vec(voteVec, voteVec)
	mulInt32VecWithConst(voteVec, voteVec, int32(el.validators.GetWeightByIdx(validatorIdx)))
	return []*atroposDecision{}, nil
}

func (el *electionB) VoteAndAggregate(
	frame consensus.Frame,
	validatorId consensus.ValidatorID,
	rootHash consensus.EventHash,
) ([]*atroposDecision, error) {
	validatorIdx := el.validatorIDMap[validatorId]
	el.prepareNewElectorRoot(frame, validatorIdx, rootHash)
	if frame <= el.frameToDeliver {
		return []*atroposDecision{}, nil
	}

	aggregationMatrix := make([]int32, (frame-el.frameToDeliver-1)*el.validatorCount, (frame-el.frameToDeliver)*el.validatorCount)
	directVoteVector := initInt32WithConst(-1, int(el.validatorCount))

	observedRoots := el.observedRoots(rootHash, frame-1)
	observedRootsWeight := int32(0)

	for _, observedRoot := range observedRoots {
		validatorIdx := el.validatorIDMap[observedRoot.ValidatorID]
		directVoteVector[validatorIdx] = 1
		observedRootsWeight += int32(el.validators.GetWeightByIdx(validatorIdx))

		if el.vote[frame-1][validatorIdx] != nil {
			if rootContext, ok := el.vote[frame-1][validatorIdx][observedRoot.RootHash]; ok {
				nonDeliveredFramesOffset := (el.frameToDeliver - rootContext.frameToDeliverOffset) * el.validatorCount
				addInt32Vecs(aggregationMatrix, aggregationMatrix, rootContext.voteMatrix[nonDeliveredFramesOffset:])
			}
		}
	}

	el.decide(frame, aggregationMatrix, observedRootsWeight)

	normalizeInt32Vec(aggregationMatrix, aggregationMatrix)
	aggregationMatrix = append(aggregationMatrix, directVoteVector...)

	mulInt32VecWithConst(aggregationMatrix, aggregationMatrix, int32(el.validators.GetWeightByIdx(validatorIdx)))
	el.vote[frame][validatorIdx][rootHash].voteMatrix = aggregationMatrix

	atropoi := el.atroposDeliveryBuffer.getDeliveryReadyAtropoi(el.frameToDeliver)
	el.frameToDeliver += consensus.Frame(len(atropoi))
	return atropoi, nil
}

func (el *electionB) decideB(frame consensus.Frame, aggregationMatr []int32, observedRootsWeight int32) bool {
	// Q = ceil((4*TotalValidatorWeight - 3*observedRootsWeight)/3)
	// numerator (Q_0) can exceed the int32 limits before division
	Q_0 := 4*int64(el.validators.TotalWeight()) - 3*int64(observedRootsWeight)
	Q := int32((Q_0 + 3 - 1) / 3)

	yesDecisions := boolMaskInt32Vec(aggregationMatr, func(x int32) bool { return x >= Q })
	noDecisions := boolMaskInt32Vec(aggregationMatr, func(x int32) bool { return x <= -Q })
	for _, candidateValidator := range el.validators.SortedIDs() {
		validatorIdx := el.validatorIDMap[candidateValidator]
		if yesDecisions[validatorIdx] {
			if frame == 259 {
				fmt.Println()
			}
			heap.Push(el.atroposDeliveryBuffer, &atroposDecision{frame, el.evidence[frame][validatorIdx]})
			el.cleanupDecidedFrame(frame)
			return true
		}
		if !noDecisions[validatorIdx] {
			return false
		}
	}
	return false
}

func (el *electionB) decide(aggregatingFrame consensus.Frame, aggregationMatr []int32, observedRootsWeight int32) {
	// Q = ceil((4*TotalValidatorWeight - 3*observedRootsWeight)/3)
	// numerator (Q_0) can exceed the int32 limits before division
	Q_0 := 4*int64(el.validators.TotalWeight()) - 3*int64(observedRootsWeight)
	Q := int32((Q_0 + 3 - 1) / 3)
	yesDecisions := boolMaskInt32Vec(aggregationMatr, func(x int32) bool { return x >= Q })
	noDecisions := boolMaskInt32Vec(aggregationMatr, func(x int32) bool { return x <= -Q })

	for frame := range el.vote {
		if frame < el.frameToDeliver || frame >= aggregatingFrame-1 {
			continue
		}

		for _, candidateValidator := range el.validators.SortedIDs() {
			validatorIdx := el.validatorIDMap[candidateValidator]
			voteMatrixOffset := (frame-el.frameToDeliver)*el.validatorCount + consensus.Frame(validatorIdx)

			if yesDecisions[voteMatrixOffset] {
				atroposHash := el.elect(frame, candidateValidator)
				heap.Push(el.atroposDeliveryBuffer, &atroposDecision{frame, atroposHash})
				el.cleanupDecidedFrame(frame)
				break
			}

			if !noDecisions[voteMatrixOffset] {
				break
			}
		}
	}
}

// elect picks the final atropos event once its frame and validator number have been finalized
// by the "upper frame" root votes'. This is trivial in case of non-forking events as such
// roots are uniquely identified by (frame, validator).
// In the case of a fork, a tiebreaker algorithm has to be run.
func (el *electionB) elect(frame consensus.Frame, validatorCandidate consensus.ValidatorID) consensus.EventHash {
	validatorIdx := el.validatorIDMap[validatorCandidate]
	candidateMap := el.vote[frame][validatorIdx]
	atroposHash := consensus.EventHash{}
	for hash := range candidateMap {
		atroposHash = hash
	}
	// tiebreaker can simply pick the first encountered root that is forkless caused by any event.
	// It is easiest to look for any vote (forkless cause) by frame + 1 roots.
	// Due to forkless cause semantics, only one forkless-caused root can exist with specified frame and validator number.
	if len(candidateMap) > 1 {
		judgeRoots := el.getFrameRoots(frame + 1)
		for atroposCandidateHash := range candidateMap {
			for _, judge := range judgeRoots {
				if el.forklessCauses(judge.RootHash, atroposCandidateHash) {
					return atroposCandidateHash
				}
			}
		}
	}

	return atroposHash
}

func (el *electionB) observedRoots(root consensus.EventHash, frame consensus.Frame) []consensusstore.RootDescriptor {
	observedRoots := make([]consensusstore.RootDescriptor, 0, el.validators.Len())
	frameRoots := el.getFrameRoots(frame)
	for _, frameRoot := range frameRoots {
		if el.forklessCauses(root, frameRoot.RootHash) {
			observedRoots = append(observedRoots, frameRoot)
		}
	}
	return observedRoots
}

func (el *electionB) prepareNewElectorRoot(frame consensus.Frame, validatorIdx consensus.ValidatorIndex, root consensus.EventHash) {
	if _, ok := el.vote[frame]; !ok {
		el.vote[frame] = make([]map[consensus.EventHash]*rootVoteContext, el.validatorCount)
	}

	if el.vote[frame][validatorIdx] == nil {
		el.vote[frame][validatorIdx] = make(map[consensus.EventHash]*rootVoteContext)
	}

	el.vote[frame][validatorIdx][root] = &rootVoteContext{frameToDeliverOffset: el.frameToDeliver}
}

func (el *electionB) cleanupDecidedFrame(frame consensus.Frame) {
	delete(el.vote, frame)
	delete(el.voteB, frame)
	delete(el.evidence, frame)
}
