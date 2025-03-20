// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensus

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
)

// GenNodes generates nodes.
// Result:
//   - nodes  is an array of node addresses;
func GenNodes(
	nodeCount int,
) (
	nodes []ValidatorID,
) {
	// init results
	nodes = make([]ValidatorID, nodeCount)
	// make and name nodes
	for i := 0; i < nodeCount; i++ {
		addr := FakePeer()
		nodes[i] = addr
		SetNodeName(addr, "node"+string('A'+rune(i)))
	}

	return
}

// ForEachRandFork generates random events with forks for test purpose.
// Result:
//   - callbacks are called for each new event;
//   - events maps node address to array of its events;
func ForEachRandFork(
	nodes []ValidatorID,
	cheatersArr []ValidatorID,
	eventCount int,
	parentCount int,
	forksCount int,
	r *rand.Rand,
	callback ForEachEvent,
) (
	events map[ValidatorID]Events,
) {
	if r == nil {
		// fixed seed
		r = rand.New(rand.NewSource(0)) // nolint:gosec
	}
	// init results
	nodeCount := len(nodes)
	events = make(map[ValidatorID]Events, nodeCount)
	cheaters := map[ValidatorID]int{}
	for _, cheater := range cheatersArr {
		cheaters[cheater] = 0
	}

	// make events
	for i := 0; i < nodeCount*eventCount; i++ {
		// seq parent
		self := i % nodeCount
		creator := nodes[self]
		parents := r.Perm(nodeCount)
		for j, n := range parents {
			if n == self {
				parents = append(parents[0:j], parents[j+1:]...)
				break
			}
		}
		parents = parents[:parentCount-1]
		// make
		e := &TestEvent{}
		e.SetCreator(creator)
		e.SetParents(EventHashes{})
		// first parent is a last creator's event or empty hash
		var parent Event
		if ee := events[creator]; len(ee) > 0 {
			parent = ee[len(ee)-1]

			// may insert fork
			forksAlready, isCheater := cheaters[creator]
			forkPossible := len(ee) > 1
			forkLimitOk := forksAlready < forksCount
			forkFlipped := r.Intn(eventCount) <= forksCount || i < (nodeCount-1)*eventCount
			if isCheater && forkPossible && forkLimitOk && forkFlipped {
				parent = ee[r.Intn(len(ee)-1)]
				if r.Intn(len(ee)) == 0 {
					parent = nil
				}
				cheaters[creator]++
			}
		}
		if parent == nil {
			e.SetSeq(1)
			e.SetLamport(1)
		} else {
			e.SetSeq(parent.Seq() + 1)
			e.AddParent(parent.ID())
			e.SetLamport(parent.Lamport() + 1)
			// other parents are the lasts other's events
			for _, other := range parents {
				if ee := events[nodes[other]]; len(ee) > 0 {
					parent := ee[len(ee)-1]
					e.AddParent(parent.ID())
					if e.Lamport() <= parent.Lamport() {
						e.SetLamport(parent.Lamport() + 1)
					}
				}
			}
		}
		e.Name = fmt.Sprintf("%s%03d", string('a'+rune(self)), len(events[creator]))
		// buildEvent callback
		if callback.Build != nil {
			err := callback.Build(e, e.Name)
			if err != nil {
				continue
			}
		}
		// save and name event
		hasher := sha256.New()
		hasher.Write(e.Bytes())
		var id [24]byte
		copy(id[:], hasher.Sum(nil)[:24])
		e.SetID(id)
		SetEventName(e.ID(), fmt.Sprintf("%s%03d", string('a'+rune(self)), len(events[creator])))
		events[creator] = append(events[creator], e)
		// callback
		if callback.Process != nil {
			callback.Process(e, e.Name)
		}
	}

	return
}

// ForEachRandEvent generates random events for test purpose.
// Result:
//   - callbacks are called for each new event;
//   - events maps node address to array of its events;
func ForEachRandEvent(
	nodes []ValidatorID,
	eventCount int,
	parentCount int,
	r *rand.Rand,
	callback ForEachEvent,
) (
	events map[ValidatorID]Events,
) {
	return ForEachRandFork(nodes, []ValidatorID{}, eventCount, parentCount, 0, r, callback)
}

// GenRandEvents generates random events for test purpose.
// Result:
//   - events maps node address to array of its events;
func GenRandEvents(
	nodes []ValidatorID,
	eventCount int,
	parentCount int,
	r *rand.Rand,
) (
	events map[ValidatorID]Events,
) {
	return ForEachRandEvent(nodes, eventCount, parentCount, r, ForEachEvent{})
}

func CalcHashForTestEvent(event *TestEvent) [24]byte {
	hasher := sha256.New()
	hasher.Write(event.Bytes())
	var id [24]byte
	copy(id[:], hasher.Sum(nil)[:24])
	return id
}

func delPeerIndex(events map[ValidatorID]Events) (res Events) {
	for _, ee := range events {
		res = append(res, ee...)
	}
	return
}
