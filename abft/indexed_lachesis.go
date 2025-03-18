// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package abft

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xsoniclabs/consensus/abft/dagidx"
	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/dag"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/inter/pos"
	"github.com/0xsoniclabs/consensus/lachesis"
	"github.com/0xsoniclabs/kvdb"
	"github.com/0xsoniclabs/kvdb/flushable"
)

var _ lachesis.Consensus = (*IndexedLachesis)(nil)

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

	Add(dag.Event) error
	Flush()
	DropNotFlushed()

	Reset(validators *pos.Validators, db kvdb.FlushableKVStore, getEvent func(hash.Event) dag.Event)
}

// NewIndexedLachesis creates IndexedLachesis instance.
func NewIndexedLachesis(store *Store, input EventSource, dagIndexer DagIndexer, crit func(error), config Config) *IndexedLachesis {
	p := &IndexedLachesis{
		Lachesis:      NewLachesis(store, input, dagIndexer, crit, config),
		DagIndexer:    dagIndexer,
		uniqueDirtyID: uniqueID{new(big.Int)},
	}

	return p
}

// Build fills consensus-related fields: Frame, IsRoot
// returns error if event should be dropped
func (p *IndexedLachesis) Build(e dag.MutableEvent) error {
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
func (p *IndexedLachesis) Process(e dag.Event) (err error) {
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

func (p *IndexedLachesis) Bootstrap(callback lachesis.ConsensusCallbacks) error {
	base := p.Lachesis.OrdererCallbacks()
	ordererCallbacks := OrdererCallbacks{
		ApplyAtropos: base.ApplyAtropos,
		EpochDBLoaded: func(epoch idx.Epoch) {
			if base.EpochDBLoaded != nil {
				base.EpochDBLoaded(epoch)
			}
			p.DagIndexer.Reset(p.store.GetValidators(), flushable.Wrap(p.store.epochTable.VectorIndex), p.Input.GetEvent)
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
