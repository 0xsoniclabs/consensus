package abft

import (
	"bytes"
	"fmt"

	"github.com/0xsoniclabs/consensus/abft/election"
	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/dag"
	"github.com/0xsoniclabs/consensus/inter/idx"
)

func rootRecordKey(frame idx.Frame, root *election.EventDescriptor) []byte {
	key := bytes.Buffer{}
	key.Write(frame.Bytes())
	key.Write(root.ValidatorID.Bytes())
	key.Write(root.EventID.Bytes())
	return key.Bytes()
}

// AddRoot stores the new root
// Not safe for concurrent use due to the complex mutable cache!
func (s *Store) AddRoot(root dag.Event) {
	s.addRoot(root, root.Frame())
}

func (s *Store) addRoot(root dag.Event, frame idx.Frame) {
	r := election.EventDescriptor{
		ValidatorID: root.Creator(),
		EventID:     root.ID(),
	}

	if err := s.epochTable.Roots.Put(rootRecordKey(frame, &r), []byte{}); err != nil {
		s.crit(err)
	}

	// Add to cache.
	if c, ok := s.cache.FrameRoots.Get(frame); ok {
		rr := c.([]election.EventDescriptor)
		rr = append(rr, r)
		s.cache.FrameRoots.Add(frame, rr, uint(len(rr)))
	}
}

const (
	frameSize       = 4
	validatorIDSize = 4
	eventIDSize     = 32
)

// GetFrameRoots returns all the roots in the specified frame
// Not safe for concurrent use due to the complex mutable cache!
func (s *Store) GetFrameRoots(frame idx.Frame) []election.EventDescriptor {
	if rr, ok := s.cache.FrameRoots.Get(frame); ok {
		return rr.([]election.EventDescriptor)
	}
	roots := make([]election.EventDescriptor, 0, 100)
	it := s.epochTable.Roots.NewIterator(frame.Bytes(), nil)
	defer it.Release()
	for it.Next() {
		key := it.Key()
		if len(key) != frameSize+validatorIDSize+eventIDSize {
			s.crit(fmt.Errorf("roots table: incorrect key len=%d", len(key)))
		}

		r := election.EventDescriptor{
			EventID:     hash.BytesToEvent(key[frameSize+validatorIDSize:]),
			ValidatorID: idx.BytesToValidatorID(key[frameSize : frameSize+validatorIDSize]),
		}
		roots = append(roots, r)
	}
	if it.Error() != nil {
		s.crit(it.Error())
	}

	s.cache.FrameRoots.Add(frame, roots, uint(len(roots)))

	return roots
}
