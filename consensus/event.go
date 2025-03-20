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
	"fmt"
)

type Event interface {
	Epoch() Epoch
	Seq() Seq
	Frame() Frame
	Creator() ValidatorID
	Lamport() Lamport

	Parents() EventHashes
	SelfParent() *EventHash
	IsSelfParent(hash EventHash) bool

	ID() EventHash

	String() string

	Size() int
}

type MutableEvent interface {
	Event
	SetEpoch(Epoch)
	SetSeq(Seq)
	SetFrame(Frame)
	SetCreator(ValidatorID)
	SetLamport(Lamport)

	SetParents(EventHashes)

	SetID(id [24]byte)
}

// BaseEvent is the consensus message in the Lachesis consensus algorithm
// The structure isn't supposed to be used as-is:
// Doesn't contain payload, it should be extended by an app
// Doesn't contain event signature, it should be extended by an app
type BaseEvent struct {
	epoch Epoch
	seq   Seq

	frame Frame

	creator ValidatorID

	parents EventHashes

	lamport Lamport

	id EventHash
}

type MutableBaseEvent struct {
	BaseEvent
}

// Build build immutable event
func (me *MutableBaseEvent) Build(rID [24]byte) *BaseEvent {
	e := me.BaseEvent
	copy(e.id[0:4], e.epoch.Bytes())
	copy(e.id[4:8], e.lamport.Bytes())
	copy(e.id[8:], rID[:])
	return &e
}

// String returns string representation.
func (e *BaseEvent) String() string {
	return fmt.Sprintf("{id=%s, p=%s, by=%d, frame=%d}", e.id.ShortID(3), e.parents.String(), e.creator, e.frame)
}

// SelfParent returns event's self-parent, if any
func (e *BaseEvent) SelfParent() *EventHash {
	if e.seq <= 1 || len(e.parents) == 0 {
		return nil
	}
	return &e.parents[0]
}

// IsSelfParent is true if specified ID is event's self-parent
func (e *BaseEvent) IsSelfParent(hash EventHash) bool {
	if e.SelfParent() == nil {
		return false
	}
	return *e.SelfParent() == hash
}

func (e *BaseEvent) Epoch() Epoch { return e.epoch }

func (e *BaseEvent) Seq() Seq { return e.seq }

func (e *BaseEvent) Frame() Frame { return e.frame }

func (e *BaseEvent) Creator() ValidatorID { return e.creator }

func (e *BaseEvent) Parents() EventHashes { return e.parents }

func (e *BaseEvent) Lamport() Lamport { return e.lamport }

func (e *BaseEvent) ID() EventHash { return e.id }

func (e *BaseEvent) Size() int { return 4 + 4 + 4 + 4 + len(e.parents)*32 + 4 + 32 }

func (e *MutableBaseEvent) SetEpoch(v Epoch) { e.epoch = v }

func (e *MutableBaseEvent) SetSeq(v Seq) { e.seq = v }

func (e *MutableBaseEvent) SetFrame(v Frame) { e.frame = v }

func (e *MutableBaseEvent) SetCreator(v ValidatorID) { e.creator = v }

func (e *MutableBaseEvent) SetParents(v EventHashes) { e.parents = v }

func (e *MutableBaseEvent) SetLamport(v Lamport) { e.lamport = v }

func (e *MutableBaseEvent) SetID(rID [24]byte) {
	copy(e.id[0:4], e.epoch.Bytes())
	copy(e.id[4:8], e.lamport.Bytes())
	copy(e.id[8:], rID[:])
}
