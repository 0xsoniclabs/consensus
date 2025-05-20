// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensusengine

import (
	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensusstore"
	"github.com/0xsoniclabs/consensus/consensus/dagindexer"
)

type OrdererCallbacks struct {
	ApplyLeader           func(certifiedFrame consensus.Frame, leader consensus.EventHash) (sealEpoch *consensus.Validators)
	RegisterElectingEvent func(electingHash consensus.EventHash)
	RegisterElectedEvent  func(electingHash consensus.EventHash, electedHash consensus.EventHash)
	EpochDBLoaded         func(consensus.Epoch)
	LatestElectingHash    consensus.EventHash
}

// Orderer processes events to reach finality on their order.
// Unlike abft.Lachesis, this raw level of abstraction doesn't track cheaters detection
type Orderer struct {
	config Config
	crit   func(error)
	store  *consensusstore.Store
	Input  EventSource

	election *electionB
	dagIndex *dagindexer.Index

	callback OrdererCallbacks
}

// NewOrderer creates Orderer instance.
// Unlike Lachesis, Orderer doesn't updates DAG indexes for events, and doesn't detect cheaters
// It has only one purpose - reaching consensus on events order.
func NewOrderer(store *consensusstore.Store, input EventSource, dagIndex *dagindexer.Index, crit func(error), config Config) *Orderer {
	p := &Orderer{
		config:   config,
		store:    store,
		Input:    input,
		crit:     crit,
		dagIndex: dagIndex,
	}

	return p
}
