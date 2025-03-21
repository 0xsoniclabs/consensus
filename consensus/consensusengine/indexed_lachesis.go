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
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensusstore"
	"github.com/0xsoniclabs/consensus/dagidx"
	"github.com/0xsoniclabs/kvdb"
	"github.com/0xsoniclabs/kvdb/flushable"
)

var _ consensus.Consensus = (*IndexedLachesis)(nil)

// IndexedLachesis performs events ordering and detects cheaters
// It's a wrapper around Orderer, which adds features which might potentially be application-specific:
// confirmed events traversal, DAG index updates and cheaters detection.
// Use this structure if need a general-purpose consensus. Instead, use lower-level abft.Orderer.
type IndexedLachesis struct {
	*Lachesis
	DagIndexer    DagIndexer
	uniqueDirtyID uniqueID
}

type DagIndexer interface {
	dagidx.VectorClock
	dagidx.ForklessCause

	Add(consensus.Event) error
	Flush()
	DropNotFlushed()

	Reset(validators *consensus.Validators, db kvdb.FlushableKVStore, getEvent func(consensus.EventHash) consensus.Event)
}

// NewIndexedLachesis creates IndexedLachesis instance.
func NewIndexedLachesis(store *consensusstore.Store, input EventSource, dagIndexer DagIndexer, crit func(error), config Config) *IndexedLachesis {
	p := &IndexedLachesis{
		Lachesis:      NewLachesis(store, input, dagIndexer, crit, config),
		DagIndexer:    dagIndexer,
		uniqueDirtyID: uniqueID{new(big.Int)},
	}

	return p
}

// Build fills consensus-related fields: Frame, IsRoot
// returns error if event should be dropped
func (p *IndexedLachesis) Build(e consensus.MutableEvent) error {
	e.SetID(p.uniqueDirtyID.sample())

	defer p.DagIndexer.DropNotFlushed()
	err := p.DagIndexer.Add(e)
	if err != nil {
		return err
	}

	return p.Lachesis.Build(e)
}

// Process takes event into processing.
// Event order matter: parents first.
// All the event checkers must be launched.
// Process is not safe for concurrent use.
func (p *IndexedLachesis) Process(e consensus.Event) (err error) {
	defer p.DagIndexer.DropNotFlushed()
	err = p.DagIndexer.Add(e)
	if err != nil {
		return err
	}

	err = p.Lachesis.Process(e)
	if err != nil {
		return err
	}
	p.DagIndexer.Flush()
	return nil
}

func (p *IndexedLachesis) Bootstrap(callback consensus.ConsensusCallbacks) error {
	base := p.Lachesis.OrdererCallbacks()
	ordererCallbacks := OrdererCallbacks{
		ApplyAtropos: base.ApplyAtropos,
		EpochDBLoaded: func(epoch consensus.Epoch) {
			if base.EpochDBLoaded != nil {
				base.EpochDBLoaded(epoch)
			}
			p.DagIndexer.Reset(p.store.GetValidators(), flushable.Wrap(p.store.EpochTable.VectorIndex), p.Input.GetEvent)
		},
	}
	return p.Lachesis.BootstrapWithOrderer(callback, ordererCallbacks)
}

type uniqueID struct {
	counter *big.Int
}

func (u *uniqueID) sample() [24]byte {
	u.counter = u.counter.Add(u.counter, common.Big1)
	var id [24]byte
	copy(id[:], u.counter.Bytes())
	return id
}
