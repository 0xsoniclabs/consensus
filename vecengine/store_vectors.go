// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package vecengine

import (
	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/kvdb"
)

func (vi *Engine) getBytes(table kvdb.Store, id hash.Event) []byte {
	key := id.Bytes()
	b, err := table.Get(key)
	if err != nil {
		vi.crit(err)
	}
	return b
}

func (vi *Engine) setBytes(table kvdb.Store, id hash.Event, b []byte) {
	key := id.Bytes()
	err := table.Put(key, b)
	if err != nil {
		vi.crit(err)
	}
}

// GetLowestAfter reads the vector from DB
func (vi *Engine) GetLowestAfter(id hash.Event) *LowestAfterSeq {
	if bVal, okGet := vi.cache.LowestAfterSeq.Get(id); okGet {
		return bVal.(*LowestAfterSeq)
	}

	b := LowestAfterSeq(vi.getBytes(vi.table.LowestAfterSeq, id))
	if b == nil {
		return nil
	}
	vi.cache.LowestAfterSeq.Add(id, &b, uint(len(b)))
	return &b
}

// GetHighestBefore reads the vector from DB
func (vi *Engine) GetHighestBefore(id hash.Event) *HighestBeforeSeq {
	if bVal, okGet := vi.cache.HighestBeforeSeq.Get(id); okGet {
		return bVal.(*HighestBeforeSeq)
	}

	b := HighestBeforeSeq(vi.getBytes(vi.table.HighestBeforeSeq, id))
	if b == nil {
		return nil
	}
	vi.cache.HighestBeforeSeq.Add(id, &b, uint(len(b)))
	return &b
}

// SetLowestAfter stores the vector into DB
func (vi *Engine) SetLowestAfter(id hash.Event, seq *LowestAfterSeq) {
	vi.setBytes(vi.table.LowestAfterSeq, id, *seq)

	vi.cache.LowestAfterSeq.Add(id, seq, uint(len(*seq)))
}

// SetHighestBefore stores the vectors into DB
func (vi *Engine) SetHighestBefore(id hash.Event, seq *HighestBeforeSeq) {
	vi.setBytes(vi.table.HighestBeforeSeq, id, *seq)

	vi.cache.HighestBeforeSeq.Add(id, seq, uint(len(*seq)))
}
