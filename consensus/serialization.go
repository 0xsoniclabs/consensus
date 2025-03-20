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
	"github.com/ethereum/go-ethereum/rlp"
)

type TestEventMarshaling struct {
	Epoch Epoch
	Seq   Seq

	Frame Frame

	Creator ValidatorID

	Parents EventHashes

	Lamport Lamport

	ID   EventHash
	Name string
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
