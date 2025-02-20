package election

import (
	"container/heap"

	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/inter/pos"
)

type (
	ForklessCauseFn func(a hash.Event, b hash.Event) bool
	GetFrameRootsFn func(f idx.Frame) []RootContext
)

type RootContext struct {
	ValidatorID idx.ValidatorID
	RootHash    hash.Event
}

type AtroposDecision struct {
	Frame       idx.Frame
	AtroposHash hash.Event
}

type RootVoteContext struct {
	frameToDeliverOffset idx.Frame
	voteMatrix           []int32
}

type Election struct {
	validators *pos.Validators

	forklessCauses ForklessCauseFn
	getFrameRoots  GetFrameRootsFn

	vote           map[idx.Frame]map[idx.ValidatorID]map[hash.Event]*RootVoteContext
	validatorIDMap map[idx.ValidatorID]idx.Validator
	validatorCount idx.Frame

	atroposDeliveryBuffer heapBuffer
	frameToDeliver        idx.Frame
}

func New(
	frameToDeliver idx.Frame,
	validators *pos.Validators,
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

func (el *Election) ResetEpoch(frameToDeliver idx.Frame, validators *pos.Validators) {
	el.atroposDeliveryBuffer = make(heapBuffer, 0)
	heap.Init(&el.atroposDeliveryBuffer)
	el.frameToDeliver = frameToDeliver
	el.validators = validators
	el.vote = make(map[idx.Frame]map[idx.ValidatorID]map[hash.Event]*RootVoteContext)
	el.validatorCount = idx.Frame(validators.Len())
	el.validatorIDMap = validators.Idxs()
}

func (el *Election) VoteAndAggregate(
	frame idx.Frame,
	validatorId idx.ValidatorID,
	rootHash hash.Event,
) ([]*AtroposDecision, error) {
	el.prepareNewElectorRoot(frame, validatorId, rootHash)
	if frame <= el.frameToDeliver {
		return []*AtroposDecision{}, nil
	}
	aggregationMatrix := make([]int32, (frame-el.frameToDeliver-1)*el.validatorCount, (frame-el.frameToDeliver)*el.validatorCount)
	directVoteVector := initInt32WithConst(-1, int(el.validatorCount))

	observedRoots := el.observedRoots(rootHash, frame-1)
	observedRootsStake := int32(0)
	for _, observedRoot := range observedRoots {
		directVoteVector[el.validatorIDMap[observedRoot.ValidatorID]] = 1.
		observedRootsStake += int32(el.validators.GetWeightByIdx(el.validatorIDMap[observedRoot.ValidatorID]))
		if rootContext, ok := el.vote[frame-1][observedRoot.ValidatorID][observedRoot.RootHash]; ok {
			nonDeliveredFramesOffset := (el.frameToDeliver - rootContext.frameToDeliverOffset) * el.validatorCount
			addInt32Vecs(aggregationMatrix, aggregationMatrix, rootContext.voteMatrix[nonDeliveredFramesOffset:])
		}
	}
	el.decide(frame, aggregationMatrix, observedRootsStake)

	normalizeInt32Vec(aggregationMatrix, aggregationMatrix)
	aggregationMatrix = append(aggregationMatrix, directVoteVector...)
	mulInt32VecWithConst(aggregationMatrix, aggregationMatrix, int32(el.validators.GetWeightByIdx(el.validatorIDMap[validatorId])))
	el.vote[frame][validatorId][rootHash].voteMatrix = aggregationMatrix
	return el.getDeliveryReadyAtropoi(), nil
}

func (el *Election) decide(aggregatingFrame idx.Frame, aggregationMatr []int32, observedRootsStake int32) {
	// Q = ceil((4*TotalStake - 3*observedRootsStake)/3)
	// numerator (Q_0) can exceed the int32 limits before division
	Q_0 := 4*int64(el.validators.TotalWeight()) - 3*int64(observedRootsStake)
	Q := int32((Q_0 + 3 - 1) / 3)
	yesDecisions := boolMaskInt32Vec(aggregationMatr, func(x int32) bool { return x >= Q })
	noDecisions := boolMaskInt32Vec(aggregationMatr, func(x int32) bool { return x <= -Q })

	for frame := range el.vote {
		if frame < el.frameToDeliver || frame >= aggregatingFrame-1 {
			continue
		}
		for _, candidateValidator := range el.validators.SortedIDs() {
			voteMatrixOffset := (frame-el.frameToDeliver)*el.validatorCount + idx.Frame(el.validators.GetIdx(candidateValidator))
			if yesDecisions[voteMatrixOffset] {
				atroposHash := el.elect(frame, candidateValidator)
				heap.Push(&el.atroposDeliveryBuffer, &AtroposDecision{frame, atroposHash})
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
func (el *Election) elect(frame idx.Frame, validatorCandidate idx.ValidatorID) hash.Event {
	candidateMap := el.vote[frame][validatorCandidate]
	atroposHash := getAnyKey(candidateMap)
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

func (el *Election) observedRoots(root hash.Event, frame idx.Frame) []RootContext {
	observedRoots := make([]RootContext, 0, el.validators.Len())
	frameRoots := el.getFrameRoots(frame)
	for _, frameRoot := range frameRoots {
		if el.forklessCauses(root, frameRoot.RootHash) {
			observedRoots = append(observedRoots, frameRoot)
		}
	}
	return observedRoots
}

// getDeliveryReadyAtropoi pops and returns only continuous sequences of decided atropoi
// that begin with `frameToDeliver` frame number
// example 1: frameToDeliver = 100, heapBuffer = [100, 101, 102] -> deliveredAtropoi = [100, 101, 102], heapBuffer = []
// example 2: frameToDeliver = 100, heapBuffer = [101, 102] -> deliveredAtropoi = [], heapBuffer = [101, 102]
// example 3: frameToDeliver = 100, heapBuffer = [100, 101, 104, 105] -> deliveredAtropoi = [100, 101], heapBuffer=[104, 105]
func (el *Election) getDeliveryReadyAtropoi() []*AtroposDecision {
	atropoi := make([]*AtroposDecision, 0)
	for len(el.atroposDeliveryBuffer) > 0 && el.atroposDeliveryBuffer[0].Frame == el.frameToDeliver {
		atropoi = append(atropoi, heap.Pop(&el.atroposDeliveryBuffer).(*AtroposDecision))
		el.frameToDeliver++
	}
	return atropoi
}

func (el *Election) prepareNewElectorRoot(frame idx.Frame, validatorId idx.ValidatorID, root hash.Event) {
	if _, ok := el.vote[frame]; !ok {
		el.vote[frame] = make(map[idx.ValidatorID]map[hash.Event]*RootVoteContext)
	}
	if _, ok := el.vote[frame][validatorId]; !ok {
		el.vote[frame][validatorId] = make(map[hash.Event]*RootVoteContext)
	}
	el.vote[frame][validatorId][root] = &RootVoteContext{frameToDeliverOffset: el.frameToDeliver}
}

func (el *Election) cleanupDecidedFrame(frame idx.Frame) {
	delete(el.vote, frame)
}

func getAnyKey(vote map[hash.Event]*RootVoteContext) hash.Event {
	for k := range vote {
		return k
	}
	return hash.Event{}
}
