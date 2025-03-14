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
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/0xsoniclabs/consensus/consensustypes"
	"github.com/0xsoniclabs/consensus/lachesis"
)

func TestConfirmBlocks_1(t *testing.T) {
	testConfirmBlocks(t, []consensustypes.Weight{1}, 0)
}

func TestConfirmBlocks_big1(t *testing.T) {
	testConfirmBlocks(t, []consensustypes.Weight{math.MaxUint32 / 2}, 0)
}

func TestConfirmBlocks_big2(t *testing.T) {
	testConfirmBlocks(t, []consensustypes.Weight{math.MaxUint32 / 4, math.MaxUint32 / 4}, 0)
}

func TestConfirmBlocks_big3(t *testing.T) {
	testConfirmBlocks(t, []consensustypes.Weight{math.MaxUint32 / 8, math.MaxUint32 / 8, math.MaxUint32 / 4}, 0)
}

func TestConfirmBlocks_4(t *testing.T) {
	testConfirmBlocks(t, []consensustypes.Weight{1, 2, 3, 4}, 0)
}

func TestConfirmBlocks_3_1(t *testing.T) {
	testConfirmBlocks(t, []consensustypes.Weight{1, 1, 1, 1}, 1)
}

func TestConfirmBlocks_67_33(t *testing.T) {
	testConfirmBlocks(t, []consensustypes.Weight{33, 67}, 1)
}

func TestConfirmBlocks_67_33_4(t *testing.T) {
	testConfirmBlocks(t, []consensustypes.Weight{11, 11, 11, 67}, 3)
}

func TestConfirmBlocks_67_33_5(t *testing.T) {
	testConfirmBlocks(t, []consensustypes.Weight{11, 11, 11, 33, 34}, 3)
}

func TestConfirmBlocks_2_8_10(t *testing.T) {
	testConfirmBlocks(t, []consensustypes.Weight{1, 2, 1, 2, 1, 2, 1, 2, 1, 2}, 3)
}

func testConfirmBlocks(t *testing.T, weights []consensustypes.Weight, cheatersCount int) {
	t.Helper()
	assertar := assert.New(t)

	nodes := consensustypes.GenNodes(len(weights))
	lch, _, input, _ := NewCoreLachesis(nodes, weights)

	var (
		frames []consensustypes.Frame
		blocks []*lachesis.Block
	)
	lch.applyBlock = func(block *lachesis.Block) *consensustypes.Validators {
		frames = append(frames, lch.store.GetLastDecidedFrame()+1)
		blocks = append(blocks, block)

		return nil
	}

	eventCount := int(TestMaxEpochEvents)
	parentCount := 5
	if parentCount > len(nodes) {
		parentCount = len(nodes)
	}
	r := rand.New(rand.NewSource(int64(len(nodes) + cheatersCount))) // nolint:gosec
	consensustypes.ForEachRandFork(nodes, nodes[:cheatersCount], eventCount, parentCount, 10, r, consensustypes.ForEachEvent{
		Process: func(e consensustypes.Event, name string) {
			input.SetEvent(e)
			assertar.NoError(
				lch.Process(e))

		},
		Build: func(e consensustypes.MutableEvent, name string) error {
			e.SetEpoch(FirstEpoch)
			return lch.Build(e)
		},
	})

	// unconfirm all events
	it := lch.store.epochTable.ConfirmedEvent.NewIterator(nil, nil)
	batch := lch.store.epochTable.ConfirmedEvent.NewBatch()
	for it.Next() {
		assertar.NoError(batch.Delete(it.Key()))
	}
	assertar.NoError(batch.Write())
	it.Release()

	for i, block := range blocks {
		frame := frames[i]
		atropos := blocks[i].Atropos

		// call confirmBlock again
		_, err := lch.onFrameDecided(frame, atropos)
		gotBlock := lch.blocks[lch.lastBlock]

		if !assertar.NoError(err) {
			break
		}
		if !assertar.LessOrEqual(gotBlock.Cheaters.Len(), cheatersCount) {
			break
		}
		if !assertar.Equal(block.Cheaters, gotBlock.Cheaters) {
			break
		}
		if !assertar.Equal(block.Atropos, gotBlock.Atropos) {
			break
		}
	}
	assertar.GreaterOrEqual(len(blocks), TestMaxEpochEvents/5)
}
