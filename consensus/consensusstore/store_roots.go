// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensusstore

import (
	"bytes"
	"fmt"

	"github.com/0xsoniclabs/consensus/consensus"
)

const (
	frameSize       = 4
	validatorIDSize = 4
	eventIDSize     = 32
)

// RootDescriptor wraps the root context retrieved from the ConsensusStore
type RootDescriptor struct {
	ValidatorID consensus.ValidatorID
	RootHash    consensus.EventHash
}

func rootRecordKey(frame consensus.Frame, root *RootDescriptor) []byte {
	key := bytes.Buffer{}
	key.Write(frame.Bytes())
	key.Write(root.ValidatorID.Bytes())
	key.Write(root.RootHash.Bytes())
	return key.Bytes()
}

// AddRoot stores the new root
// Not safe for concurrent use due to the complex mutable cache!
func (s *Store) AddRoot(root consensus.Event) {
	s.addRoot(root, root.Frame())
}

func (s *Store) addRoot(root consensus.Event, frame consensus.Frame) {
	r := RootDescriptor{
		ValidatorID: root.Creator(),
		RootHash:    root.ID(),
	}

	if err := s.EpochTable.Roots.Put(rootRecordKey(frame, &r), []byte{}); err != nil {
		s.crit(err)
	}

	// Add to cache.
	if c, ok := s.cache.FrameRoots.Get(frame); ok {
		rr := c.([]RootDescriptor)
		rr = append(rr, r)
		s.cache.FrameRoots.Add(frame, rr, uint(len(rr)))
	}
}

// GetFrameRoots returns all the roots in the specified frame
// Not safe for concurrent use due to the complex mutable cache!
func (s *Store) GetFrameRoots(frame consensus.Frame) []RootDescriptor {
	if rr, ok := s.cache.FrameRoots.Get(frame); ok {
		return rr.([]RootDescriptor)
	}
	roots := make([]RootDescriptor, 0, 100)
	it := s.EpochTable.Roots.NewIterator(frame.Bytes(), nil)
	defer it.Release()
	for it.Next() {
		key := it.Key()
		if len(key) != frameSize+validatorIDSize+eventIDSize {
			s.crit(fmt.Errorf("roots table: incorrect key len=%d", len(key)))
		}

		r := RootDescriptor{
			RootHash:    consensus.BytesToEvent(key[frameSize+validatorIDSize:]),
			ValidatorID: consensus.BytesToValidatorID(key[frameSize : frameSize+validatorIDSize]),
		}
		roots = append(roots, r)
	}
	if it.Error() != nil {
		s.crit(it.Error())
	}

	s.cache.FrameRoots.Add(frame, roots, uint(len(roots)))

	return roots
}
