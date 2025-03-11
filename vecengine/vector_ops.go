// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package vecengine

import (
	"github.com/0xsoniclabs/consensus/ctype"
)

func (b *LowestAfterSeq) InitWithEvent(i ctype.ValidatorIdx, e ctype.Event) {
	b.Set(i, e.Seq())
}

func (b *LowestAfterSeq) Visit(i ctype.ValidatorIdx, e ctype.Event) bool {
	if b.Get(i) != 0 {
		return false
	}

	b.Set(i, e.Seq())
	return true
}

func (b *HighestBeforeSeq) InitWithEvent(i ctype.ValidatorIdx, e ctype.Event) {
	b.Set(i, BranchSeq{Seq: e.Seq(), MinSeq: e.Seq()})
}

func (b *HighestBeforeSeq) IsEmpty(i ctype.ValidatorIdx) bool {
	seq := b.Get(i)
	return !seq.IsForkDetected() && seq.Seq == 0
}

func (b *HighestBeforeSeq) IsForkDetected(i ctype.ValidatorIdx) bool {
	return b.Get(i).IsForkDetected()
}

func (b *HighestBeforeSeq) Seq(i ctype.ValidatorIdx) ctype.Seq {
	val := b.Get(i)
	return val.Seq
}

func (b *HighestBeforeSeq) MinSeq(i ctype.ValidatorIdx) ctype.Seq {
	val := b.Get(i)
	return val.MinSeq
}

func (b *HighestBeforeSeq) SetForkDetected(i ctype.ValidatorIdx) {
	b.Set(i, forkDetectedSeq)
}

func (self *HighestBeforeSeq) CollectFrom(_other HighestBeforeI, num ctype.ValidatorIdx) {
	other := _other.(*HighestBeforeSeq)
	for branchID := ctype.ValidatorIdx(0); branchID < num; branchID++ {
		hisSeq := other.Get(branchID)
		if hisSeq.Seq == 0 && !hisSeq.IsForkDetected() {
			// hisSeq doesn't observe anything about this branchID
			continue
		}
		mySeq := self.Get(branchID)

		if mySeq.IsForkDetected() {
			// mySeq observes the maximum already
			continue
		}
		if hisSeq.IsForkDetected() {
			// set fork detected
			self.SetForkDetected(branchID)
		} else {
			if mySeq.Seq == 0 || mySeq.MinSeq > hisSeq.MinSeq {
				// take hisSeq.MinSeq
				mySeq.MinSeq = hisSeq.MinSeq
				self.Set(branchID, mySeq)
			}
			if mySeq.Seq < hisSeq.Seq {
				// take hisSeq.Seq
				mySeq.Seq = hisSeq.Seq
				self.Set(branchID, mySeq)
			}
		}
	}
}

func (self *HighestBeforeSeq) GatherFrom(to ctype.ValidatorIdx, _other HighestBeforeI, from []ctype.ValidatorIdx) {
	other := _other.(*HighestBeforeSeq)
	// read all branches to find highest event
	highestBranchSeq := BranchSeq{}
	for _, branchID := range from {
		branch := other.Get(branchID)
		if branch.IsForkDetected() {
			highestBranchSeq = branch
			break
		}
		if branch.Seq > highestBranchSeq.Seq {
			highestBranchSeq = branch
		}
	}
	self.Set(to, highestBranchSeq)
}
