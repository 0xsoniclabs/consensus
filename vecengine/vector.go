// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: <TBD>
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package vecengine

import (
	"github.com/0xsoniclabs/consensus/inter/dag"
	"github.com/0xsoniclabs/consensus/inter/idx"
)

type LowestAfterI interface {
	InitWithEvent(i idx.Validator, e dag.Event)
	Visit(i idx.Validator, e dag.Event) bool
}

type HighestBeforeI interface {
	InitWithEvent(i idx.Validator, e dag.Event)
	IsEmpty(i idx.Validator) bool
	IsForkDetected(i idx.Validator) bool
	Seq(i idx.Validator) idx.Event
	MinSeq(i idx.Validator) idx.Event
	SetForkDetected(i idx.Validator)
	CollectFrom(other HighestBeforeI, branches idx.Validator)
	GatherFrom(to idx.Validator, other HighestBeforeI, from []idx.Validator)
}

type allVecs struct {
	after  LowestAfterI
	before HighestBeforeI
}
