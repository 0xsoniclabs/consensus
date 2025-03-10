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
	"github.com/0xsoniclabs/kvdb"
)

func openDB(p kvdb.DBProducer, skipPrefix []byte, name string) (kvdb.Store, error) {
	store, err := p.OpenDB(name)
	if err != nil {
		return nil, err
	}
	return &Store{store, skipPrefix}, nil
}

type AllDBProducer struct {
	kvdb.FullDBProducer
	skipPrefix []byte
}

func WrapAllProducer(p kvdb.FullDBProducer, skipPrefix []byte) *AllDBProducer {
	return &AllDBProducer{
		FullDBProducer: p,
		skipPrefix:     skipPrefix,
	}
}

func (p *AllDBProducer) OpenDB(name string) (kvdb.Store, error) {
	return openDB(p.FullDBProducer, p.skipPrefix, name)
}

type DBProducer struct {
	kvdb.DBProducer
	skipPrefix []byte
}

func WrapProducer(p kvdb.DBProducer, skipPrefix []byte) *DBProducer {
	return &DBProducer{
		DBProducer: p,
		skipPrefix: skipPrefix,
	}
}

func (p *DBProducer) OpenDB(name string) (kvdb.Store, error) {
	return openDB(p.DBProducer, p.skipPrefix, name)
}
