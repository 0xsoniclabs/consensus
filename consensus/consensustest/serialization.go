// Copyright (c) 2026 Sonic Operations Ltd

package consensustest

import (
	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/ethereum/go-ethereum/rlp"
)

type TestEventMarshaling struct {
	Epoch   consensus.Epoch
	Seq     consensus.Seq
	Frame   consensus.Frame
	Creator consensus.ValidatorID
	Parents consensus.EventHashes
	Lamport consensus.Lamport
	ID      consensus.EventHash
	Name    string
}

// EventToBytes serializes events
func (e *TestEvent) Bytes() []byte {
	b, _ := rlp.EncodeToBytes(&TestEventMarshaling{
		Epoch:   e.Epoch(),
		Seq:     e.Seq(),
		Frame:   e.Frame(),
		Creator: e.Creator(),
		Parents: e.Parents(),
		Lamport: e.Lamport(),
		ID:      e.ID(),
		Name:    e.Name,
	})
	return b
}
