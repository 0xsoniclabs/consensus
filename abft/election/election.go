package election

import (
	"container/heap"

	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/inter/pos"
	"github.com/viterin/vek/vek32"
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

type ElectorRoot struct {
	frameToDeliverOffset idx.Frame
	rootHash             hash.Event
	voteMatr             []float32
}

type Election struct {
	validators *pos.Validators

	forklessCauses ForklessCauseFn
	getFrameRoots  GetFrameRootsFn

	vote           map[idx.Frame]map[idx.ValidatorID]*ElectorRoot
	validatorIDMap map[idx.ValidatorID]idx.Validator
	validatorCount idx.Frame

	deliveryBuffer heapBuffer
	frameToDeliver idx.Frame
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
	el.deliveryBuffer = make(heapBuffer, 0)
	heap.Init(&el.deliveryBuffer)
	el.frameToDeliver = frameToDeliver
	el.validators = validators
	el.vote = make(map[idx.Frame]map[idx.ValidatorID]*ElectorRoot)
	el.validatorCount = idx.Frame(validators.Len())
	el.validatorIDMap = validators.Idxs()
}

func (el *Election) ElectForRoot(
	frame idx.Frame,
	validatorId idx.ValidatorID,
	rootHash hash.Event,
) ([]*AtroposDecision, error) {
	el.prepareNewElectorRoot(frame, validatorId, rootHash)
	if frame <= el.frameToDeliver {
		return []*AtroposDecision{}, nil
	}
	aggregationMatrix := make([]float32, (frame-el.frameToDeliver-1)*el.validatorCount, (frame-el.frameToDeliver)*el.validatorCount)
	voteVec := vek32.Repeat(-1., int(el.validatorCount))

	observedRoots := el.observedRoots(rootHash, frame-1)
	stakeAccul := float32(0)
	for _, observedRoot := range observedRoots {
		voteVec[el.validatorIDMap[observedRoot.ValidatorID]] = 1.
		stakeAccul += float32(el.validators.GetWeightByIdx(el.validators.GetIdx(observedRoot.ValidatorID)))
		if rootContext, ok := el.vote[frame-1][observedRoot.ValidatorID]; ok {
			vek32.Add_Inplace(aggregationMatrix, rootContext.voteMatr[(el.frameToDeliver-rootContext.frameToDeliverOffset)*el.validatorCount:])
		}
	}
	el.decideRoots(frame, aggregationMatrix, stakeAccul)
	kroneckerDeltaMask := vek32.GteNumber(aggregationMatrix, 0.)
	vek32.FromBool_Into(aggregationMatrix, kroneckerDeltaMask)
	vek32.MulNumber_Inplace(aggregationMatrix, 2.)
	vek32.SubNumber_Inplace(aggregationMatrix, 1.)
	aggregationMatrix = append(aggregationMatrix, voteVec...)
	vek32.MulNumber_Inplace(aggregationMatrix, float32(el.validators.GetWeightByIdx(el.validatorIDMap[validatorId])))
	el.vote[frame][validatorId].voteMatr = aggregationMatrix
	return el.getDeliveryReadyAtropoi(), nil
}

func (el *Election) decideRoots(aggregatingFrame idx.Frame, aggregationMatr []float32, seenRootsStake float32) {
	Q := (4.*float32(el.validators.TotalWeight()) - 3*seenRootsStake) / 4
	yesDecisions := vek32.GtNumber(aggregationMatr, Q)
	noDecisions := vek32.LtNumber(aggregationMatr, -Q)

	for frame := range el.vote {
		if frame < el.frameToDeliver || frame >= aggregatingFrame-1 {
			continue
		}
		for _, v := range el.validators.SortedIDs() {
			offset := (frame-el.frameToDeliver)*el.validatorCount + idx.Frame(el.validators.GetIdx(v))
			if yesDecisions[offset] {
				heap.Push(&el.deliveryBuffer, &AtroposDecision{frame, el.vote[frame][v].rootHash})
				el.cleanupDecidedFrame(frame)
				break
			}
			if !noDecisions[offset] {
				break
			}
		}
	}
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

// getDeliveryReadyAtropoi pops and returns only continuous sequence of decided atropoi
// that start with `frameToDeliver` frame number
// example 1: frameToDeliver = 100, heapBuffer = [100, 101, 102], deliveredAtropoi = [100, 101, 102]
// example 2: frameToDeliver = 100, heapBuffer = [101, 102], deliveredAtropoi = []
// example 3: frameToDeliver = 100, heapBuffer = [100, 101, 104, 105], deliveredAtropoi = [100, 101], heapBuffer=[104, 105]
func (el *Election) getDeliveryReadyAtropoi() []*AtroposDecision {
	atropoi := make([]*AtroposDecision, 0)
	for len(el.deliveryBuffer) > 0 && el.deliveryBuffer[0].Frame == el.frameToDeliver {
		atropoi = append(atropoi, heap.Pop(&el.deliveryBuffer).(*AtroposDecision))
		el.frameToDeliver++
	}
	return atropoi
}

func (el *Election) prepareNewElectorRoot(frame idx.Frame, validatorId idx.ValidatorID, root hash.Event) {
	if _, ok := el.vote[frame]; !ok {
		el.vote[frame] = make(map[idx.ValidatorID]*ElectorRoot)
	}
	el.vote[frame][validatorId] = &ElectorRoot{frameToDeliverOffset: el.frameToDeliver, rootHash: root}
}

func (el *Election) cleanupDecidedFrame(frame idx.Frame) {
	delete(el.vote, frame)
}
