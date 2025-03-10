// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package flaggedproducer

import (
	"sync/atomic"

	"github.com/0xsoniclabs/kvdb"
	"github.com/0xsoniclabs/kvdb/flushable"
)

type flaggedStore struct {
	kvdb.Store
	DropFn     func()
	Dirty      uint32
	flushIDKey []byte
}

type flaggedBatch struct {
	kvdb.Batch
	db *flaggedStore
}

func (s *flaggedStore) Close() error {
	return nil
}

func (s *flaggedStore) Drop() {
	s.DropFn()
}

func (s *flaggedStore) modified() error {
	if atomic.LoadUint32(&s.Dirty) == 0 {
		atomic.StoreUint32(&s.Dirty, 1)
		err := s.Store.Put(s.flushIDKey, []byte{flushable.DirtyPrefix})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *flaggedStore) Put(key []byte, value []byte) error {
	err := s.modified()
	if err != nil {
		return err
	}
	return s.Store.Put(key, value)
}

func (s *flaggedStore) Delete(key []byte) error {
	err := s.modified()
	if err != nil {
		return err
	}
	return s.Store.Delete(key)
}

func (s *flaggedStore) NewBatch() kvdb.Batch {
	return &flaggedBatch{
		Batch: s.Store.NewBatch(),
		db:    s,
	}
}

func (s *flaggedBatch) Write() error {
	err := s.db.modified()
	if err != nil {
		return err
	}
	return s.Batch.Write()
}
