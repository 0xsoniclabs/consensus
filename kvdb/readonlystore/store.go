// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package readonlystore

import "github.com/0xsoniclabs/consensus/kvdb"

type Store struct {
	kvdb.Store
}

func Wrap(s kvdb.Store) *Store {
	return &Store{s}
}

// Put inserts the given value into the key-value data store.
func (s *Store) Put(key []byte, value []byte) error {
	return kvdb.ErrUnsupportedOp
}

// Delete removes the key from the key-value data store.
func (s *Store) Delete(key []byte) error {
	return kvdb.ErrUnsupportedOp
}

type Batch struct {
	kvdb.Batch
}

func (s *Store) NewBatch() kvdb.Batch {
	return &Batch{s.Store.NewBatch()}
}

// Put inserts the given value into the key-value data store.
func (s *Batch) Put(key []byte, value []byte) error {
	return kvdb.ErrUnsupportedOp
}

// Delete removes the key from the key-value data store.
func (s *Batch) Delete(key []byte) error {
	return kvdb.ErrUnsupportedOp
}
