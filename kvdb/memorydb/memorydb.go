// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package memorydb

import (
	"github.com/0xsoniclabs/kvdb"
	"github.com/0xsoniclabs/kvdb/devnulldb"
	"github.com/0xsoniclabs/kvdb/flushable"
)

// Database is an ephemeral key-value store. Apart from basic data storage
// functionality it also supports batch writes and iterating over the keyspace in
// binary-alphabetical order.
type Database struct {
	kvdb.Store
}

// New returns a wrapped map with all the required database interface methods
// implemented.
func New() *Database {
	return &Database{
		Store: flushable.Wrap(devnulldb.New()),
	}
}

// NewWithDrop is the same as New, but defines onDrop callback.
func NewWithDrop(drop func()) *Database {
	return &Database{
		Store: flushable.WrapWithDrop(devnulldb.New(), drop),
	}
}
