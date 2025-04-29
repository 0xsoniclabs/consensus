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

// BaseDescriptor wraps the base context retrieved from the ConsensusStore
type BaseDescriptor struct {
	ValidatorID consensus.ValidatorID
	BaseHash    consensus.EventHash
}

func baseRecordKey(frame consensus.Frame, base *BaseDescriptor) []byte {
	key := bytes.Buffer{}
	key.Write(frame.Bytes())
	key.Write(base.ValidatorID.Bytes())
	key.Write(base.BaseHash.Bytes())
	return key.Bytes()
}

// AddBase stores the new base
// Not safe for concurrent use due to the complex mutable cache!
func (s *Store) AddBase(base consensus.Event) {
	s.addBase(base, base.Frame())
}

func (s *Store) addBase(base consensus.Event, frame consensus.Frame) {
	baseDescriptor := BaseDescriptor{
		ValidatorID: base.Creator(),
		BaseHash:    base.ID(),
	}

	if err := s.EpochTable.Bases.Put(baseRecordKey(frame, &baseDescriptor), []byte{}); err != nil {
		s.crit(err)
	}

	// Add to cache.
	if c, ok := s.cache.FrameBases.Get(frame); ok {
		baseDescriptors := c.([]BaseDescriptor)
		baseDescriptors = append(baseDescriptors, baseDescriptor)
		s.cache.FrameBases.Add(frame, baseDescriptors, uint(len(baseDescriptors)))
	}
}

// GetFrameBases returns all the bases in the specified frame
// Not safe for concurrent use due to the complex mutable cache!
func (s *Store) GetFrameBases(frame consensus.Frame) []BaseDescriptor {
	if rr, ok := s.cache.FrameBases.Get(frame); ok {
		return rr.([]BaseDescriptor)
	}
	bases := make([]BaseDescriptor, 0, 100)
	it := s.EpochTable.Bases.NewIterator(frame.Bytes(), nil)
	defer it.Release()
	for it.Next() {
		key := it.Key()
		if len(key) != frameSize+validatorIDSize+eventIDSize {
			s.crit(fmt.Errorf("bases table: incorrect key len=%d", len(key)))
		}

		r := BaseDescriptor{
			BaseHash:    consensus.BytesToEvent(key[frameSize+validatorIDSize:]),
			ValidatorID: consensus.BytesToValidatorID(key[frameSize : frameSize+validatorIDSize]),
		}
		bases = append(bases, r)
	}
	if it.Error() != nil {
		s.crit(it.Error())
	}

	s.cache.FrameBases.Add(frame, bases, uint(len(bases)))

	return bases
}
