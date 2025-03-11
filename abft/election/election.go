// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package election

import (
	"container/heap"

	"github.com/0xsoniclabs/consensus/consensustypes"
)

type (
	ForklessCauseFn func(a consensustypes.EventHash, b consensustypes.EventHash) bool
	GetFrameRootsFn func(f consensustypes.Frame) []RootContext
)

type RootContext struct {
	ValidatorID consensustypes.ValidatorID
	RootHash    consensustypes.EventHash
}

type AtroposDecision struct {
	Frame       consensustypes.Frame
	AtroposHash consensustypes.EventHash
}

type RootVoteContext struct {
	frameToDeliverOffset consensustypes.Frame
	voteMatrix           []int32
}

type Election struct {
	validators *consensustypes.Validators

	forklessCauses ForklessCauseFn
	getFrameRoots  GetFrameRootsFn

	vote           map[consensustypes.Frame]map[consensustypes.ValidatorID]map[consensustypes.EventHash]*RootVoteContext
	validatorIDMap map[consensustypes.ValidatorID]consensustypes.ValidatorIdx
	validatorCount consensustypes.Frame

	atroposDeliveryBuffer *atroposHeap
	frameToDeliver        consensustypes.Frame
}

func New(
	frameToDeliver consensustypes.Frame,
	validators *consensustypes.Validators,
	forklessCauseFn ForklessCauseFn,
	getFrameRoots GetFrameRootsFn,
) *Election {
	election := &Election{
		forklessCauses: forklessCauseFn,
		getFrameRoots:  getFrameRoots,
		validators:     validators,
	}
	election.ResetEpoch(frameToDeliver, validators)
	return election
}

func (el *Election) ResetEpoch(frameToDeliver consensustypes.Frame, validators *consensustypes.Validators) {
	el.atroposDeliveryBuffer = NewAtroposHeap()
	el.frameToDeliver = frameToDeliver
	el.validators = validators
	el.vote = make(map[consensustypes.Frame]map[consensustypes.ValidatorID]map[consensustypes.EventHash]*RootVoteContext)
	el.validatorCount = consensustypes.Frame(validators.Len())
	el.validatorIDMap = validators.Idxs()
}

func (el *Election) VoteAndAggregate(
	frame consensustypes.Frame,
	validatorId consensustypes.ValidatorID,
	rootHash consensustypes.EventHash,
) ([]*AtroposDecision, error) {
	el.prepareNewElectorRoot(frame, validatorId, rootHash)
	if frame <= el.frameToDeliver {
		return []*AtroposDecision{}, nil
	}
	aggregationMatrix := make([]int32, (frame-el.frameToDeliver-1)*el.validatorCount, (frame-el.frameToDeliver)*el.validatorCount)
	directVoteVector := initInt32WithConst(-1, int(el.validatorCount))

	observedRoots := el.observedRoots(rootHash, frame-1)
	observedRootsWeight := int32(0)
	for _, observedRoot := range observedRoots {
		directVoteVector[el.validatorIDMap[observedRoot.ValidatorID]] = 1.
		observedRootsWeight += int32(el.validators.GetWeightByIdx(el.validatorIDMap[observedRoot.ValidatorID]))
		if rootContext, ok := el.vote[frame-1][observedRoot.ValidatorID][observedRoot.RootHash]; ok {
			nonDeliveredFramesOffset := (el.frameToDeliver - rootContext.frameToDeliverOffset) * el.validatorCount
			addInt32Vecs(aggregationMatrix, aggregationMatrix, rootContext.voteMatrix[nonDeliveredFramesOffset:])
		}
	}
	el.decide(frame, aggregationMatrix, observedRootsWeight)

	normalizeInt32Vec(aggregationMatrix, aggregationMatrix)
	aggregationMatrix = append(aggregationMatrix, directVoteVector...)
	mulInt32VecWithConst(aggregationMatrix, aggregationMatrix, int32(el.validators.GetWeightByIdx(el.validatorIDMap[validatorId])))
	el.vote[frame][validatorId][rootHash].voteMatrix = aggregationMatrix

	atropoi := el.atroposDeliveryBuffer.getDeliveryReadyAtropoi(el.frameToDeliver)
	el.frameToDeliver += consensustypes.Frame(len(atropoi))
	return atropoi, nil
}

func (el *Election) decide(aggregatingFrame consensustypes.Frame, aggregationMatr []int32, observedRootsWeight int32) {
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
			voteMatrixOffset := (frame-el.frameToDeliver)*el.validatorCount + consensustypes.Frame(el.validators.GetIdx(candidateValidator))
			if yesDecisions[voteMatrixOffset] {
				atroposHash := el.elect(frame, candidateValidator)
				heap.Push(el.atroposDeliveryBuffer, &AtroposDecision{frame, atroposHash})
				el.cleanupDecidedFrame(frame)
				break
			}
			if !noDecisions[voteMatrixOffset] {
				break
			}
		}
	}
}

// elect picks the final atropos event once it's frame and validator number have been finalized
// by the "upper frame" root votes'. This is trivial in case of non-forking events as such
// roots are uniquely identified by (frame, validator).
// In the case of a fork, a tiebreaker algorithms has to be run.
func (el *Election) elect(frame consensustypes.Frame, validatorCandidate consensustypes.ValidatorID) consensustypes.EventHash {
	candidateMap := el.vote[frame][validatorCandidate]
	// get any hash identifed by (frame, validatorCandidate) tuple
	// for non-forking scenarios, only a single such root is possible
	atroposHash := consensustypes.EventHash{}
	for hash := range candidateMap {
		atroposHash = hash
	}
	// tiebreaker can simply pick the first encountered root that is forkless caused by any event.
	// It is easiest to look for any vote (forkless cause) by frame + 1 roots.
	// Due to forkless cause semantics, only one forking root can exist with specified frame and validator number.
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

func (el *Election) observedRoots(root consensustypes.EventHash, frame consensustypes.Frame) []RootContext {
	observedRoots := make([]RootContext, 0, el.validators.Len())
	frameRoots := el.getFrameRoots(frame)
	for _, frameRoot := range frameRoots {
		if el.forklessCauses(root, frameRoot.RootHash) {
			observedRoots = append(observedRoots, frameRoot)
		}
	}
	return observedRoots
}

func (el *Election) prepareNewElectorRoot(frame consensustypes.Frame, validatorId consensustypes.ValidatorID, root consensustypes.EventHash) {
	if _, ok := el.vote[frame]; !ok {
		el.vote[frame] = make(map[consensustypes.ValidatorID]map[consensustypes.EventHash]*RootVoteContext)
	}
	if _, ok := el.vote[frame][validatorId]; !ok {
		el.vote[frame][validatorId] = make(map[consensustypes.EventHash]*RootVoteContext)
	}
	el.vote[frame][validatorId][root] = &RootVoteContext{frameToDeliverOffset: el.frameToDeliver}
}

func (el *Election) cleanupDecidedFrame(frame consensustypes.Frame) {
	delete(el.vote, frame)
}
