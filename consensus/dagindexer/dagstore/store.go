// Copyright (c) 2026 Sonic Operations Ltd
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package dagstore

import (
	"errors"

	"github.com/0xsoniclabs/cacheutils/simplewlru"
	"github.com/0xsoniclabs/cacheutils/wlru"
	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/dagindexer/dagvec"
	"github.com/0xsoniclabs/kvdb"
	"github.com/0xsoniclabs/kvdb/table"
)

// CacheConfig holds cache size configuration for the store.
type CacheConfig struct {
	HighestBeforeTimeSize uint
	HighestBeforeSeqSize  uint
	LowestAfterSeqSize    uint
}

// Store provides persistent storage and caching for vector clocks and branch info.
type Store struct {
	crit func(error)

	Db kvdb.FlushableKVStore

	Table struct {
		HighestBeforeTime kvdb.Store `table:"T"`
		EventBranch       kvdb.Store `table:"b"`
		BranchesInfo      kvdb.Store `table:"B"`
		HighestBeforeSeq  kvdb.Store `table:"S"`
		LowestAfterSeq    kvdb.Store `table:"s"`
	}

	cache struct {
		HighestBeforeTime *wlru.Cache
		HighestBeforeSeq  *simplewlru.Cache
		LowestAfterSeq    *simplewlru.Cache
	}
}

// NewStore creates a Store with initialized caches.
func NewStore(crit func(error), cfg CacheConfig) *Store {
	s := &Store{
		crit: crit,
	}
	s.cache.HighestBeforeTime, _ = wlru.New(cfg.HighestBeforeTimeSize, int(cfg.HighestBeforeTimeSize))
	s.cache.HighestBeforeSeq, _ = simplewlru.New(cfg.HighestBeforeSeqSize, int(cfg.HighestBeforeSeqSize))
	s.cache.LowestAfterSeq, _ = simplewlru.New(cfg.LowestAfterSeqSize, int(cfg.HighestBeforeSeqSize))
	return s
}

// InitTables migrates the table layout onto the given database.
func (s *Store) InitTables(db kvdb.FlushableKVStore) {
	s.Db = db
	table.MigrateTables(&s.Table, s.Db)
}

func (s *Store) getBytes(t kvdb.Store, id consensus.EventHash) []byte {
	key := id.Bytes()
	b, err := t.Get(key)
	if err != nil {
		s.crit(err)
	}
	return b
}

func (s *Store) setBytes(t kvdb.Store, id consensus.EventHash, b []byte) {
	key := id.Bytes()
	err := t.Put(key, b)
	if err != nil {
		s.crit(err)
	}
}

// GetHighestBefore reads the vector from DB.
func (s *Store) GetHighestBefore(id consensus.EventHash) *dagvec.HighestBefore {
	var vSeq *dagvec.HighestBeforeSeq = nil
	if vSeqVal, ok := s.cache.HighestBeforeSeq.Get(id); ok {
		vSeq = vSeqVal.(*dagvec.HighestBeforeSeq)
	} else {
		vSeqVal := dagvec.HighestBeforeSeq(s.getBytes(s.Table.HighestBeforeSeq, id))
		if vSeqVal != nil {
			vSeq = &vSeqVal
			s.cache.HighestBeforeSeq.Add(id, vSeq, uint(len(*vSeq)))
		}
	}
	var vTime *dagvec.HighestBeforeTime = nil
	if vTimeVal, ok := s.cache.HighestBeforeTime.Get(id); ok {
		vTime = vTimeVal.(*dagvec.HighestBeforeTime)
	} else {
		vTimeVal := dagvec.HighestBeforeTime(s.getBytes(s.Table.HighestBeforeTime, id))
		if vTimeVal != nil {
			vTime = &vTimeVal
			s.cache.HighestBeforeTime.Add(id, vTime, uint(len(*vTime)))
		}
	}
	if vSeq != nil && vTime != nil {
		return &dagvec.HighestBefore{
			VSeq:  vSeq,
			VTime: vTime,
		}
	}
	return nil
}

// GetLowestAfter reads the vector from DB.
func (s *Store) GetLowestAfter(id consensus.EventHash) *dagvec.LowestAfter {
	if bVal, okGet := s.cache.LowestAfterSeq.Get(id); okGet {
		return bVal.(*dagvec.LowestAfter)
	}
	b := dagvec.LowestAfter(s.getBytes(s.Table.LowestAfterSeq, id))
	if b == nil {
		return nil
	}
	s.cache.LowestAfterSeq.Add(id, &b, uint(len(b)))
	return &b
}

// SetHighestBefore stores the vectors into DB.
func (s *Store) SetHighestBefore(id consensus.EventHash, vec *dagvec.HighestBefore) {
	s.setBytes(s.Table.HighestBeforeTime, id, *vec.VTime)
	s.cache.HighestBeforeTime.Add(id, vec.VTime, uint(len(*vec.VTime)))

	s.setBytes(s.Table.HighestBeforeSeq, id, *vec.VSeq)
	s.cache.HighestBeforeSeq.Add(id, vec.VSeq, uint(len(*vec.VSeq)))
}

// SetLowestAfter stores the vector into DB.
func (s *Store) SetLowestAfter(id consensus.EventHash, seq *dagvec.LowestAfterSeq) {
	s.setBytes(s.Table.LowestAfterSeq, id, *seq)
	s.cache.LowestAfterSeq.Add(id, seq, uint(len(*seq)))
}

// SetBranchesInfoBytes stores raw BranchesInfo bytes into DB.
func (s *Store) SetBranchesInfoBytes(data []byte) {
	key := []byte("c")
	if err := s.Table.BranchesInfo.Put(key, data); err != nil {
		s.crit(err)
	}
}

// GetBranchesInfoBytes reads raw BranchesInfo bytes from DB.
func (s *Store) GetBranchesInfoBytes() []byte {
	key := []byte("c")
	buf, err := s.Table.BranchesInfo.Get(key)
	if err != nil {
		s.crit(err)
	}
	return buf
}

// SetEventBranchID stores the event's global branch ID.
func (s *Store) SetEventBranchID(id consensus.EventHash, branchID consensus.ValidatorIndex) {
	s.setBytes(s.Table.EventBranch, id, branchID.Bytes())
}

// GetEventBranchID reads the event's global branch ID.
func (s *Store) GetEventBranchID(id consensus.EventHash) consensus.ValidatorIndex {
	b := s.getBytes(s.Table.EventBranch, id)
	if b == nil {
		s.crit(errors.New("failed to read event's branch ID (inconsistent DB)"))
		return 0
	}
	branchID := consensus.BytesToValidator(b)
	return branchID
}

// OnDropNotFlushed purges all caches.
func (s *Store) OnDropNotFlushed() {
	s.cache.HighestBeforeSeq.Purge()
	s.cache.LowestAfterSeq.Purge()
	s.cache.HighestBeforeTime.Purge()
}
