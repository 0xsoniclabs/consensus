// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensustest

import (
	"math/rand"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/utils/byteutils"
	"github.com/ethereum/go-ethereum/common"
)

type TestEvent struct {
	consensus.MutableBaseEvent
	Name string
}

func (e *TestEvent) AddParent(id consensus.EventHash) {
	parents := e.Parents()
	parents.Add(id)
	e.SetParents(parents)
}

/*
 * Utils:
 */

// FakePeer generates random fake peer id for testing purpose.
func FakePeer() consensus.ValidatorID {
	return consensus.BytesToValidatorID(FakeHash().Bytes()[:4])
}

// FakeEpoch gives fixed value of fake epoch for testing purpose.
func FakeEpoch() consensus.Epoch {
	return 123456
}

// FakeEvent generates random fake event hash with the same epoch for testing purpose.
func FakeEvent() (h consensus.EventHash) {
	_, err := rand.Read(h[:]) // nolint:gosec
	if err != nil {
		panic(err)
	}
	copy(h[0:4], byteutils.Uint32ToBigEndian(uint32(FakeEpoch())))
	return
}

// FakeEvents generates random hashes of fake event with the same epoch for testing purpose.
func FakeEvents(n int) consensus.EventHashes {
	res := consensus.EventHashes{}
	for i := 0; i < n; i++ {
		res.Add(FakeEvent())
	}
	return res
}

// FakeHash generates random fake hash for testing purpose.
func FakeHash(seed ...int64) (h common.Hash) {
	randRead := rand.Read

	if len(seed) > 0 {
		src := rand.NewSource(seed[0])
		rnd := rand.New(src) // nolint:gosec
		randRead = rnd.Read
	}

	_, err := randRead(h[:])
	if err != nil {
		panic(err)
	}
	return
}
