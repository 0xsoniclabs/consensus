package dagindexer

import (
	"github.com/0xsoniclabs/consensus/inter/dag"
	"github.com/0xsoniclabs/consensus/inter/idx"
)

type CreationTimer interface {
	CreationTime() Timestamp
}

func (b *HighestBefore) InitWithEvent(i idx.Validator, e dag.Event) {
	b.VSeq.Set(i, BranchSeq{Seq: e.Seq(), MinSeq: e.Seq()})
	if eCreationTimer, ok := e.(CreationTimer); ok { // Workaround for existing type-unsafe practices.
		b.VTime.Set(i, eCreationTimer.CreationTime())
	}
}

func (b *LowestAfter) InitWithEvent(i idx.Validator, e dag.Event) {
	b.Set(i, e.Seq())
}

func (b *LowestAfter) Visit(i idx.Validator, e dag.Event) bool {
	if b.Get(i) != 0 {
		return false
	}

	b.Set(i, e.Seq())
	return true
}

func (b *HighestBefore) IsEmpty(i idx.Validator) bool {
	seq := b.VSeq.Get(i)
	return !seq.IsForkDetected() && seq.Seq == 0
}

func (b *HighestBefore) IsForkDetected(i idx.Validator) bool {
	return b.VSeq.Get(i).IsForkDetected()
}

func (b *HighestBefore) Seq(i idx.Validator) idx.Event {
	return b.VSeq.Get(i).Seq
}

func (b *HighestBefore) MinSeq(i idx.Validator) idx.Event {
	return b.VSeq.Get(i).MinSeq
}

func (b *HighestBefore) SetForkDetected(i idx.Validator) {
	b.VSeq.Set(i, forkDetectedSeq)
}

func (hb *HighestBefore) CollectFrom(other *HighestBefore, num idx.Validator) {
	for branchID := idx.Validator(0); branchID < num; branchID++ {
		hisSeq := other.VSeq.Get(branchID)
		if hisSeq.Seq == 0 && !hisSeq.IsForkDetected() {
			// hisSeq doesn't observe anything about this branchID
			continue
		}
		mySeq := hb.VSeq.Get(branchID)

		if mySeq.IsForkDetected() {
			// mySeq observes the maximum already
			continue
		}
		if hisSeq.IsForkDetected() {
			// set fork detected
			hb.SetForkDetected(branchID)
		} else {
			if mySeq.Seq == 0 || mySeq.MinSeq > hisSeq.MinSeq {
				// take hisSeq.MinSeq
				mySeq.MinSeq = hisSeq.MinSeq
				hb.VSeq.Set(branchID, mySeq)
			}
			if mySeq.Seq < hisSeq.Seq {
				// take hisSeq.Seq
				mySeq.Seq = hisSeq.Seq
				hb.VSeq.Set(branchID, mySeq)
				hb.VTime.Set(branchID, other.VTime.Get(branchID))
			}
		}
	}
}

func (hb *HighestBefore) GatherFrom(to idx.Validator, other *HighestBefore, from []idx.Validator) {
	// read all branches to find highest event
	highestBranchSeq := BranchSeq{}
	highestBranchTime := Timestamp(0)
	for _, branchID := range from {
		vseq := other.VSeq.Get(branchID)
		vtime := other.VTime.Get(branchID)
		if vseq.IsForkDetected() {
			highestBranchSeq = vseq
			break
		}
		if vseq.Seq > highestBranchSeq.Seq {
			highestBranchSeq = vseq
			highestBranchTime = vtime
		}
	}
	hb.VSeq.Set(to, highestBranchSeq)
	hb.VTime.Set(to, highestBranchTime)
}
