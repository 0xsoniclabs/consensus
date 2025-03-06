// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package lachesis

import (
	"github.com/0xsoniclabs/consensus/inter/dag"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/inter/pos"
)

// Consensus is a consensus interface.
type Consensus interface {
	// Process takes event for processing.
	Process(e dag.Event) error
	// Build sets consensus fields. Returns an error if event should be dropped.
	Build(e dag.MutableEvent) error
	// Reset switches epoch state to a new empty epoch.
	Reset(epoch idx.Epoch, validators *pos.Validators) error
}

type ApplyEventFn func(event dag.Event)
type EndBlockFn func() (sealEpoch *pos.Validators)

type BlockCallbacks struct {
	// ApplyEvent is called on confirmation of each event during block processing.
	// Cannot be called twice for the same event.
	// The order in which ApplyBlock is called for events is deterministic but undefined. It's application's responsibility to sort events according to its needs.
	// It's application's responsibility to interpret this data (e.g. events may be related to batches of transactions or other ordered data).
	ApplyEvent ApplyEventFn
	// EndBlock indicates that ApplyEvent was called for all the events
	// Returns validators group for a new epoch, if epoch must be sealed after this bock
	// If epoch must not get sealed, then this callback must return nil
	EndBlock EndBlockFn
}

type BeginBlockFn func(block *Block) BlockCallbacks

// ConsensusCallbacks contains callbacks called during block processing by consensus engine
type ConsensusCallbacks struct {
	// BeginBlock returns further callbacks for processing of this block
	BeginBlock BeginBlockFn
}
