package dagindexer

import (
	"github.com/0xsoniclabs/consensus/consensus"
)

type CreationTimer interface {
	CreationTimePortable() Timestamp
}

func (b *HighestBefore) InitWithEvent(i consensus.ValidatorIndex, e consensus.Event, uid consensus.Seq) {
	b.VSeq.Set(i, BranchSeq{Seq: e.Seq(), MinSeq: e.Seq(), uid: uid})
	if eCreationTimer, ok := e.(CreationTimer); ok { // Workaround for existing type-unsafe practices.
		b.VTime.Set(i, eCreationTimer.CreationTimePortable())
	}
}

func (b *LowestAfter) InitWithEvent(i consensus.ValidatorIndex, e consensus.Event) {
	b.Set(i, e.Seq())
}

func (b *LowestAfter) Visit(i consensus.ValidatorIndex, e consensus.Event) bool {
	if b.Get(i) != 0 {
		return false
	}

	b.Set(i, e.Seq())
	return true
}

func (b *HighestBefore) IsEmpty(i consensus.ValidatorIndex) bool {
	seq := b.VSeq.Get(i)
	return !seq.IsEquivocationDetected() && seq.Seq == 0
}

func (b *HighestBefore) IsEquivocationDetected(i consensus.ValidatorIndex) bool {
	return b.VSeq.Get(i).IsEquivocationDetected()
}

func (b *HighestBefore) Seq(i consensus.ValidatorIndex) consensus.Seq {
	return b.VSeq.Get(i).Seq
}

func (b *HighestBefore) MinSeq(i consensus.ValidatorIndex) consensus.Seq {
	return b.VSeq.Get(i).MinSeq
}

func (b *HighestBefore) SetEquivocationDetected(i consensus.ValidatorIndex) {
	b.VSeq.Set(i, equivocationDetectedSeq)
}

func (hb *HighestBefore) CollectFrom(other *HighestBefore, num consensus.ValidatorIndex, diff []consensus.Seq) {
	for branchID := consensus.ValidatorIndex(0); branchID < num; branchID++ {
		hisSeq := other.VSeq.Get(branchID)
		if hisSeq.Seq == 0 && !hisSeq.IsEquivocationDetected() {
			// hisSeq doesn't observe anything about this branchID
			continue
		}
		mySeq := hb.VSeq.Get(branchID)

		if mySeq.IsEquivocationDetected() {
			// mySeq reaches the maximum already
			continue
		}
		if hisSeq.IsEquivocationDetected() {
			// set equivocation detected
			hb.SetEquivocationDetected(branchID)
		} else {
			if mySeq.Seq == 0 || mySeq.MinSeq > hisSeq.MinSeq {
				// take hisSeq.MinSeq
				mySeq.MinSeq = hisSeq.MinSeq
				hb.VSeq.Set(branchID, mySeq)
			}
			if mySeq.Seq < hisSeq.Seq {
				// take hisSeq.Seq
				if diff != nil {
					diff[branchID] += hisSeq.Seq - mySeq.Seq
				}
				mySeq.Seq = hisSeq.Seq
				mySeq.uid = hisSeq.uid
				hb.VSeq.Set(branchID, mySeq)
				hb.VTime.Set(branchID, other.VTime.Get(branchID))
			}
		}
	}
}

func (hb *HighestBefore) GatherFrom(to consensus.ValidatorIndex, other *HighestBefore, from []consensus.ValidatorIndex) {
	// read all branches to find highest event
	highestBranchSeq := BranchSeq{}
	highestBranchTime := Timestamp(0)
	for _, branchID := range from {
		vseq := other.VSeq.Get(branchID)
		vtime := other.VTime.Get(branchID)
		if vseq.IsEquivocationDetected() {
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
