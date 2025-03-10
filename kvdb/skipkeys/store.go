// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package skipkeys

import (
	"bytes"

	"github.com/0xsoniclabs/consensus/kvdb"
)

type Store struct {
	kvdb.Store
	skipPrefix []byte
}

func Wrap(store kvdb.Store, skipPrefix []byte) *Store {
	return &Store{store, skipPrefix}
}

// Has retrieves if a key is present in the key-value data store.
func (s *Store) Has(key []byte) (bool, error) {
	if bytes.HasPrefix(key, s.skipPrefix) {
		return false, nil
	}
	return s.Store.Has(key)
}

// Get retrieves the given key if it's present in the key-value data store.
func (s *Store) Get(key []byte) ([]byte, error) {
	if bytes.HasPrefix(key, s.skipPrefix) {
		return nil, nil
	}
	return s.Store.Get(key)
}

// NewIterator creates a binary-alphabetical iterator over a subset
// of database content with a particular key prefix, starting at a particular
// initial key (or after, if it does not exist).
func (s *Store) NewIterator(prefix []byte, start []byte) kvdb.Iterator {
	return iterator{s.Store.NewIterator(prefix, start), s.skipPrefix}
}

type iterator struct {
	kvdb.Iterator
	skipPrefix []byte
}

func (it iterator) Next() bool {
	first := true
	for first || bytes.HasPrefix(it.Key(), it.skipPrefix) {
		if !it.Iterator.Next() {
			return false
		}
		first = false
	}
	return true
}
