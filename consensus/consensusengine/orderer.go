// Copyright (c) 2026 Sonic Operations Ltd

package consensusengine

import (
	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensusstore"
	"github.com/0xsoniclabs/consensus/consensus/dagindexer"
)

type OrdererCallbacks struct {
	ApplyLeader func(certifiedFrame consensus.Frame, leader consensus.EventHash) (sealEpoch *consensus.Validators)

	EpochDBLoaded func(consensus.Epoch)
}

// Orderer processes events to reach finality on their order.
// Unlike abft.Lachesis, this raw level of abstraction doesn't track cheaters detection
type Orderer struct {
	config Config
	crit   func(error)
	store  *consensusstore.Store
	Input  EventSource

	election *election
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
