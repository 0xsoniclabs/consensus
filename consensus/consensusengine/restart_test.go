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
	"errors"
	"math"
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensusstore"
	"github.com/0xsoniclabs/consensus/consensus/consensustest"
	"github.com/0xsoniclabs/consensus/consensus/dagindexer"

	"github.com/stretchr/testify/assert"

	"github.com/0xsoniclabs/kvdb"
	"github.com/0xsoniclabs/kvdb/memorydb"
)

func TestRestart_1(t *testing.T) {
	testRestart(t, []consensus.Weight{1}, 0)
}

func TestRestart_big1(t *testing.T) {
	testRestart(t, []consensus.Weight{math.MaxUint32 / 2}, 0)
}

func TestRestart_big2(t *testing.T) {
	testRestart(t, []consensus.Weight{math.MaxUint32 / 4, math.MaxUint32 / 4}, 0)
}

func TestRestart_big3(t *testing.T) {
	testRestart(t, []consensus.Weight{math.MaxUint32 / 8, math.MaxUint32 / 8, math.MaxUint32 / 4}, 0)
}

func TestRestart_4(t *testing.T) {
	testRestart(t, []consensus.Weight{1, 2, 3, 4}, 0)
}

func TestRestart_3_1(t *testing.T) {
	testRestart(t, []consensus.Weight{1, 1, 1, 1}, 1)
}

func TestRestart_67_33(t *testing.T) {
	testRestart(t, []consensus.Weight{33, 67}, 1)
}

func TestRestart_67_33_4(t *testing.T) {
	testRestart(t, []consensus.Weight{11, 11, 11, 67}, 3)
}

func TestRestart_67_33_5(t *testing.T) {
	testRestart(t, []consensus.Weight{11, 11, 11, 33, 34}, 3)
}

func TestRestart_2_8_10(t *testing.T) {
	testRestart(t, []consensus.Weight{1, 2, 1, 2, 1, 2, 1, 2, 1, 2}, 3)
}

func testRestart(t *testing.T, weights []consensus.Weight, cheatersCount int) {
	t.Helper()
	testRestartAndReset(t, weights, false, cheatersCount, false)
	testRestartAndReset(t, weights, false, cheatersCount, true)
	testRestartAndReset(t, weights, true, 0, false)
	testRestartAndReset(t, weights, true, 0, true)
}

func testRestartAndReset(t *testing.T, weights []consensus.Weight, mutateWeights bool, cheatersCount int, resets bool) {
	t.Helper()
	assertar := assert.New(t)

	const (
		COUNT     = 3 // 3 abft instances
		GENERATOR = 0 // event generator
		EXPECTED  = 1 // sample
		RESTORED  = 2 // compare with sample
	)
	nodes := consensustest.GenNodes(len(weights))

	lchs := make([]*CoreLachesis, 0, COUNT)
	inputs := make([]*consensustest.TestEventSource, 0, COUNT)
	for i := 0; i < COUNT; i++ {
		lch, _, input, _ := NewBootstrappedCoreConsensus(nodes, weights)
		lchs = append(lchs, lch)
		inputs = append(inputs, input)
	}

	eventCount := TestMaxEpochEvents
	const epochs = 5
	// maxEpochBlocks should be much smaller than eventCount so that there would be enough events to seal epoch
	var maxEpochBlocks = eventCount / 4

	// seal epoch on decided frame == maxEpochBlocks
	for _, _lch := range lchs {
		lch := _lch // capture
		lch.applyBlock = func(block *consensus.Block) *consensus.Validators {
			if lch.store.GetLastDecidedFrame()+1 == consensus.Frame(maxEpochBlocks) {
				// seal epoch
				if mutateWeights {
					return mutateValidators(lch.store.GetValidators())
				}
				return lch.store.GetValidators()
			}
			return nil
		}
	}

	var ordered consensus.Events
	parentCount := 5
	if parentCount > len(nodes) {
		parentCount = len(nodes)
	}
	epochStates := map[consensus.Epoch]*consensusstore.EpochState{}
	r := consensustest.NewIntSeededRandGenerator(uint64(len(nodes) + cheatersCount))
	for epoch := consensus.Epoch(1); epoch <= consensus.Epoch(epochs); epoch++ {
		consensustest.ForEachRandFork(nodes, nodes[:cheatersCount], eventCount, parentCount, 10, r, consensustest.ForEachEvent{
			Process: func(e consensus.Event, name string) {
				inputs[GENERATOR].SetEvent(e)
				assertar.NoError(
					lchs[GENERATOR].Process(e))

				ordered = append(ordered, e)
				epochStates[lchs[GENERATOR].store.GetEpoch()] = lchs[GENERATOR].store.GetEpochState()
			},
			Build: func(e consensus.MutableEvent, name string) error {
				if epoch != lchs[GENERATOR].store.GetEpoch() {
					return errors.New("epoch already sealed, skip")
				}
				e.SetEpoch(epoch)
				return lchs[GENERATOR].Build(e)
			},
		})
	}
	if !assertar.Equal(maxEpochBlocks*epochs, len(lchs[GENERATOR].blocks)) {
		return
	}

	resetEpoch := consensus.Epoch(0)

	// use pre-ordered events, call consensus(es) directly
	for _, e := range ordered {
		if e.Epoch() < resetEpoch {
			continue
		}
		if resets && epochStates[e.Epoch()+2] != nil && r.IntN(30) == 0 {
			// never reset last epoch to be able to compare latest state
			resetEpoch = e.Epoch() + 1
			err := lchs[EXPECTED].Reset(resetEpoch, epochStates[resetEpoch].Validators)
			assertar.NoError(err)
			err = lchs[RESTORED].Reset(resetEpoch, epochStates[resetEpoch].Validators)
			assertar.NoError(err)
		}
		if e.Epoch() < resetEpoch {
			continue
		}
		if r.IntN(10) == 0 {
			prev := lchs[RESTORED]

			store := consensusstore.NewMemStore()
			// copy prev DB into new one
			{
				it := prev.store.MainDB.NewIterator(nil, nil)
				for it.Next() {
					assertar.NoError(store.MainDB.Put(it.Key(), it.Value()))
				}
				it.Release()
			}
			restartEpochDB := memorydb.New()
			{
				it := prev.store.EpochDB.NewIterator(nil, nil)
				for it.Next() {
					assertar.NoError(restartEpochDB.Put(it.Key(), it.Value()))
				}
				it.Release()
			}
			restartEpoch := prev.store.GetEpoch()
			store.GetEpochDB = func(epoch consensus.Epoch) kvdb.Store {
				if epoch == restartEpoch {
					return restartEpochDB
				}
				return memorydb.New()
			}

			restored := NewIndexedLachesis(store, prev.Input, dagindexer.NewIndex(prev.crit, dagindexer.LiteConfig()), prev.crit, prev.config)
			assertar.NoError(restored.Bootstrap(prev.callback))

			lchs[RESTORED].IndexedLachesis = restored
		}

		if !assertar.Equal(e.Epoch(), lchs[EXPECTED].store.GetEpoch()) {
			break
		}
		inputs[EXPECTED].SetEvent(e)
		assertar.NoError(
			lchs[EXPECTED].Process(e))

		inputs[RESTORED].SetEvent(e)
		assertar.NoError(
			lchs[RESTORED].Process(e))

		compareStates(assertar, lchs[EXPECTED], lchs[RESTORED])
		if t.Failed() {
			return
		}
	}

	compareStates(assertar, lchs[GENERATOR], lchs[RESTORED])
	compareBlocks(assertar, lchs[EXPECTED], lchs[RESTORED])
}

func compareStates(assertar *assert.Assertions, expected, restored *CoreLachesis) {
	assertar.Equal(
		*(expected.store.GetLastDecidedState()), *(restored.store.GetLastDecidedState()))
	assertar.Equal(
		expected.store.GetEpochState().String(), restored.store.GetEpochState().String())
	// check last block
	if len(expected.blocks) != 0 {
		assertar.Equal(expected.lastBlock, restored.lastBlock)
		assertar.Equal(
			expected.blocks[expected.lastBlock],
			restored.blocks[restored.lastBlock],
			"block doesn't match")
	}
}

func compareBlocks(assertar *assert.Assertions, expected, restored *CoreLachesis) {
	assertar.Equal(expected.lastBlock, restored.lastBlock)
	for e := consensus.Epoch(1); e <= expected.lastBlock.Epoch; e++ {
		assertar.Equal(expected.epochBlocks[e], restored.epochBlocks[e])
		for f := consensus.Frame(1); f < expected.epochBlocks[e]; f++ {
			key := BlockKey{e, f}
			if !assertar.NotNil(restored.blocks[key]) ||
				!assertar.Equal(expected.blocks[key], restored.blocks[key]) {
				return
			}
		}
	}
}
