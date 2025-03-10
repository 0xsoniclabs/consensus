// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package dagidx

import (
	"github.com/0xsoniclabs/consensus/consensus"
)

type Seq interface {
	Seq() consensus.Seq
	IsForkDetected() bool
}

type HighestBeforeSeq interface {
	Size() int
	Get(i consensus.ValidatorIndex) Seq
}

type ForklessCause interface {
	// ForklessCause calculates "sufficient coherence" between the events.
	// The A.HighestBefore array remembers the sequence number of the last
	// event by each validator that is an ancestor of A. The array for
	// B.LowestAfter remembers the sequence number of the earliest
	// event by each validator that is a descendant of B. Compare the two arrays,
	// and find how many elements in the A.HighestBefore array are greater
	// than or equal to the corresponding element of the B.LowestAfter
	// array. If there are more than 2n/3 such matches, then the A and B
	// have achieved sufficient coherency.
	//
	// If B1 and B2 are forks, then they cannot BOTH forkless-cause any specific event A,
	// unless more than 1/3W are Byzantine.
	// This great property is the reason why this function exists,
	// providing the base for the BFT algorithm.
	ForklessCause(aID, bID consensus.EventHash) bool
}

type VectorClock interface {
	GetMergedHighestBefore(id consensus.EventHash) HighestBeforeSeq
}
