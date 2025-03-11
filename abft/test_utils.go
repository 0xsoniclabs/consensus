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
	"fmt"
	"math/rand"

	"github.com/0xsoniclabs/consensus/ctype"
	"github.com/0xsoniclabs/consensus/vecengine"

	"github.com/0xsoniclabs/consensus/kvdb/memorydb"
	"github.com/0xsoniclabs/consensus/lachesis"
	"github.com/0xsoniclabs/consensus/utils/adapters"
)

type dbEvent struct {
	hash        ctype.EventHash
	validatorId ctype.ValidatorID
	seq         ctype.Seq
	frame       ctype.Frame
	lamportTs   ctype.Lamport
	parents     []ctype.EventHash
}

func (e *dbEvent) String() string {
	return fmt.Sprintf("{Epoch:%d Validator:%d Frame:%d Seq:%d Lamport:%d}", e.hash.Epoch(), e.validatorId, e.frame, e.seq, e.lamportTs)
}

type applyBlockFn func(block *lachesis.Block) *ctype.Validators

type BlockKey struct {
	Epoch ctype.Epoch
	Frame ctype.Frame
}

type BlockResult struct {
	Atropos    ctype.EventHash
	Cheaters   lachesis.Cheaters
	Validators *ctype.Validators
}

// CoreLachesis extends Indexed Orderer for tests.
type CoreLachesis struct {
	*IndexedLachesis

	blocks      map[BlockKey]*BlockResult
	lastBlock   BlockKey
	epochBlocks map[ctype.Epoch]ctype.Frame

	applyBlock applyBlockFn
}

// NewCoreLachesis creates empty abft consensus with mem store and optional node weights w.o. some callbacks usually instantiated by Client
func NewCoreLachesis(nodes []ctype.ValidatorID, weights []ctype.Weight, mods ...memorydb.Mod) (*CoreLachesis, *Store, *EventStore, *adapters.VectorToDagIndexer) {
	validators := make(ctype.ValidatorsBuilder, len(nodes))
	for i, v := range nodes {
		if weights == nil {
			validators[v] = 1
		} else {
			validators[v] = weights[i]
		}
	}
	store := NewMemStore()

	err := store.ApplyGenesis(&Genesis{
		Validators: validators.Build(),
		Epoch:      FirstEpoch,
	})
	if err != nil {
		panic(err)
	}

	input := NewEventStore()

	config := LiteConfig()
	crit := func(err error) {
		panic(err)
	}
	dagIndexer := &adapters.VectorToDagIndexer{Engine: vecengine.NewIndex(crit, vecengine.LiteConfig(), vecengine.GetEngineCallbacks)}
	lch := NewIndexedLachesis(store, input, dagIndexer, crit, config)

	extended := &CoreLachesis{
		IndexedLachesis: lch,
		blocks:          map[BlockKey]*BlockResult{},
		epochBlocks:     map[ctype.Epoch]ctype.Frame{},
	}

	err = extended.Bootstrap(lachesis.ConsensusCallbacks{
		BeginBlock: func(block *lachesis.Block) lachesis.BlockCallbacks {
			return lachesis.BlockCallbacks{
				EndBlock: func() (sealEpoch *ctype.Validators) {
					// track blocks
					key := BlockKey{
						Epoch: extended.store.GetEpoch(),
						Frame: extended.store.GetLastDecidedFrame() + 1,
					}
					extended.blocks[key] = &BlockResult{
						Atropos:    block.Atropos,
						Cheaters:   block.Cheaters,
						Validators: extended.store.GetValidators(),
					}
					// check that prev block exists
					if extended.lastBlock.Epoch != key.Epoch && key.Frame != 1 {
						panic("first frame must be 1")
					}
					extended.epochBlocks[key.Epoch]++
					extended.lastBlock = key
					if extended.applyBlock != nil {
						return extended.applyBlock(block)
					}
					return nil
				},
			}
		},
	})
	if err != nil {
		panic(err)
	}

	return extended, store, input, dagIndexer
}

func mutateValidators(validators *ctype.Validators) *ctype.Validators {
	r := rand.New(rand.NewSource(int64(validators.TotalWeight()))) // nolint:gosec
	builder := ctype.NewBuilder()
	for _, vid := range validators.IDs() {
		stake := uint64(validators.Get(vid))*uint64(500+r.Intn(500))/1000 + 1
		builder.Set(vid, ctype.Weight(stake))
	}
	return builder.Build()
}

// EventStore is a abft event storage for test purpose.
// It implements EventSource interface.
type EventStore struct {
	db map[ctype.EventHash]ctype.Event
}

// NewEventStore creates store over memory map.
func NewEventStore() *EventStore {
	return &EventStore{
		db: map[ctype.EventHash]ctype.Event{},
	}
}

// Close leaves underlying database.
func (s *EventStore) Close() {
	s.db = nil
}

// SetEvent stores event.
func (s *EventStore) SetEvent(e ctype.Event) {
	s.db[e.ID()] = e
}

// GetEvent returns stored event.
func (s *EventStore) GetEvent(h ctype.EventHash) ctype.Event {
	return s.db[h]
}

// HasEvent returns true if event exists.
func (s *EventStore) HasEvent(h ctype.EventHash) bool {
	_, ok := s.db[h]
	return ok
}
