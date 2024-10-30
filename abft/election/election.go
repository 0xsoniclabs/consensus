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
	GetFrameRootsFn func(f idx.Frame) []EventDescriptor
)

type EventDescriptor struct {
	ValidatorID idx.ValidatorID
	EventID     hash.Event
}

type AtroposDecision struct {
	Frame     idx.Frame
	AtroposID hash.Event
}

type RootContext struct {
	startFrame idx.Frame
	rootHash   hash.Event
	voteMatr   []float32
}

type Election struct {
	validators *pos.Validators

	forklessCauses ForklessCauseFn
	getFrameRoots  GetFrameRootsFn

	root   map[idx.Frame]map[idx.ValidatorID]*RootContext
	valMap map[idx.ValidatorID]idx.Validator
	valNum idx.Frame

	deliveryBuffer heapBuffer
	frameToDeliver idx.Frame
}

func New(
	frameToDecide idx.Frame,
	validators *pos.Validators,
	forklessCauseFn ForklessCauseFn,
	getFrameRoots GetFrameRootsFn,
) *Election {
	election := &Election{
		forklessCauses: forklessCauseFn,
		getFrameRoots:  getFrameRoots,
		validators:     validators,
		frameToDeliver: frameToDecide,
	}
	election.Reset(frameToDecide, validators)
	return election
}

func (el *Election) Reset(frameToDecide idx.Frame, validators *pos.Validators) {
	el.deliveryBuffer = make(heapBuffer, 0)
	heap.Init(&el.deliveryBuffer)
	el.frameToDeliver = frameToDecide
	el.validators = validators
	el.root = make(map[idx.Frame]map[idx.ValidatorID]*RootContext)
	el.valNum = idx.Frame(validators.Len())
	el.valMap = validators.Idxs()
}

func (el *Election) newRoot(frame idx.Frame, validatorId idx.ValidatorID, root hash.Event) {
	if _, ok := el.root[frame]; !ok {
		el.root[frame] = make(map[idx.ValidatorID]*RootContext)
	}
	el.root[frame][validatorId] = &RootContext{startFrame: el.frameToDeliver, rootHash: root}
}

func (el *Election) decidedFrameCleanup(frame idx.Frame) {
	delete(el.root, frame)
}

// ProcessRoot calculates Atropos votes only for the new root.
// If this root observes that the current election is decided, then return decided Atropoi
func (el *Election) ProcessRoot(
	frame idx.Frame,
	validatorId idx.ValidatorID,
	voterRoot hash.Event,
) ([]*AtroposDecision, error) {
	el.newRoot(frame, validatorId, voterRoot)
	if frame <= el.frameToDeliver {
		return []*AtroposDecision{}, nil
	}
	voteMatr := make([]float32, (frame-el.frameToDeliver-1)*el.valNum, (frame-el.frameToDeliver)*el.valNum)
	voteVec := vek32.Repeat(-1., int(el.valNum))

	observedRoots := el.observedRoots(voterRoot, frame-1)
	stakeAccul := float32(0)
	for _, seenRoot := range observedRoots {
		voteVec[el.valMap[seenRoot.ValidatorID]] = 1.
		stakeAccul += float32(el.validators.GetWeightByIdx(el.validators.GetIdx(seenRoot.ValidatorID)))
		if rootContext, ok := el.root[frame-1][seenRoot.ValidatorID]; ok {
			vek32.Add_Inplace(voteMatr, rootContext.voteMatr[(el.frameToDeliver-rootContext.startFrame)*el.valNum:])
		}
	}
	el.decideRoots(frame, voteMatr, stakeAccul)
	kroneckerDeltaMask := vek32.GteNumber(voteMatr, 0.)
	vek32.FromBool_Into(voteMatr, kroneckerDeltaMask)
	vek32.MulNumber_Inplace(voteMatr, 2.)
	vek32.SubNumber_Inplace(voteMatr, 1.)
	voteMatr = append(voteMatr, voteVec...)
	vek32.MulNumber_Inplace(voteMatr, float32(el.validators.GetWeightByIdx(el.valMap[validatorId])))
	el.root[frame][validatorId].voteMatr = voteMatr
	return el.alignedAtropoi(), nil
}

func (el *Election) decideRoots(aggregatingFrame idx.Frame, aggregationMatr []float32, seenRootsStake float32) {
	Q := (4.*float32(el.validators.TotalWeight()) - 3*seenRootsStake) / 4
	yesDecisions := vek32.GtNumber(aggregationMatr, Q)
	noDecisions := vek32.LtNumber(aggregationMatr, -Q)

	for frame := range el.root {
		if frame < el.frameToDeliver || frame >= aggregatingFrame-1 {
			continue
		}
		for _, v := range el.validators.SortedIDs() {
			offset := (frame-el.frameToDeliver)*el.valNum + idx.Frame(el.validators.GetIdx(v))
			if yesDecisions[offset] {
				heap.Push(&el.deliveryBuffer, &AtroposDecision{frame, el.root[frame][v].rootHash})
				el.decidedFrameCleanup(frame)
				break
			}
			if !noDecisions[offset] {
				break
			}
		}
	}
}

func (el *Election) observedRoots(root hash.Event, frame idx.Frame) []EventDescriptor {
	observedRoots := make([]EventDescriptor, 0, el.validators.Len())
	frameRoots := el.getFrameRoots(frame)
	for _, frameRoot := range frameRoots {
		if el.forklessCauses(root, frameRoot.EventID) {
			observedRoots = append(observedRoots, frameRoot)
		}
	}
	return observedRoots
}

// alignedAtropoi pops and returns only continuous sequence of decided atropoi
// that start with `frameToDeliver` frame number
// example 1: frameToDeliver = 100, heapBuffer = [100, 101, 102], deliveredAtropoi = [100, 101, 102]
// example 2: frameToDeliver = 100, heapBuffer = [101, 102], deliveredAtropoi = []
// example 3: frameToDeliver = 100, heapBuffer = [100, 101, 104, 105], deliveredAtropoi = [100, 101]
func (el *Election) alignedAtropoi() []*AtroposDecision {
	deliveredAtropoi := make([]*AtroposDecision, 0)
	for len(el.deliveryBuffer) > 0 && el.deliveryBuffer[0].Frame == el.frameToDeliver {
		deliveredAtropoi = append(deliveredAtropoi, heap.Pop(&el.deliveryBuffer).(*AtroposDecision))
		el.frameToDeliver++
	}
	return deliveredAtropoi
}
