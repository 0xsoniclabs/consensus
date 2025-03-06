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
	"github.com/0xsoniclabs/consensus/kvdb"
)

type Mod func(store kvdb.Store) kvdb.Store

type Producer struct {
	fs   *fakeFS
	mods []Mod
}

// NewProducer of memory db.
func NewProducer(namespace string, mods ...Mod) kvdb.IterableDBProducer {
	return &Producer{
		fs:   newFakeFS(namespace),
		mods: mods,
	}
}

// Names of existing databases.
func (p *Producer) Names() []string {
	return p.fs.ListFakeDBs()
}

// OpenDB or create db with name.
func (p *Producer) OpenDB(name string) (kvdb.Store, error) {
	db := p.fs.OpenFakeDB(name)

	for _, mod := range p.mods {
		db = mod(db)
	}

	return db, nil
}
