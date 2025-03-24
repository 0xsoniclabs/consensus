// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensusstore

import (
	"errors"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/0xsoniclabs/cacheutils/simplewlru"
	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/kvdb"
	"github.com/0xsoniclabs/kvdb/memorydb"
	"github.com/0xsoniclabs/kvdb/table"
)

// Store is a abft persistent storage working over parent key-value database.
type Store struct {
	GetEpochDB EpochDBProducer
	cfg        StoreConfig
	crit       func(error)

	MainDB kvdb.Store
	table  struct {
		LastDecidedState kvdb.Store `table:"c"`
		EpochState       kvdb.Store `table:"e"`
	}

	cache struct {
		LastDecidedState *LastDecidedState
		EpochState       *EpochState
		FrameRoots       *simplewlru.Cache `cache:"-"` // store by pointer
	}

	EpochDB    kvdb.Store
	EpochTable struct {
		Roots          kvdb.Store `table:"r"`
		VectorIndex    kvdb.Store `table:"v"`
		ConfirmedEvent kvdb.Store `table:"C"`
	}
}

var (
	ErrNoGenesis = errors.New("genesis not applied")
)

type EpochDBProducer func(epoch consensus.Epoch) kvdb.Store

// NewStore creates store over key-value db.
func NewStore(mainDB kvdb.Store, getDB EpochDBProducer, crit func(error), cfg StoreConfig) *Store {
	s := &Store{
		GetEpochDB: getDB,
		cfg:        cfg,
		crit:       crit,
		MainDB:     mainDB,
	}

	table.MigrateTables(&s.table, s.MainDB)

	s.initCache()

	return s
}

func (s *Store) initCache() {
	s.cache.FrameRoots = s.makeCache(s.cfg.Cache.RootsNum, s.cfg.Cache.RootsFrames)
}

// NewMemStore creates store over memory map.
// Store is always blank.
func NewMemStore() *Store {
	getDb := func(epoch consensus.Epoch) kvdb.Store {
		return memorydb.New()
	}
	cfg := LiteStoreConfig()
	crit := func(err error) {
		panic(err)
	}
	return NewStore(memorydb.New(), getDb, crit, cfg)
}

// Close leaves underlying database.
func (s *Store) Close() error {
	setnil := func() interface{} {
		return nil
	}

	table.MigrateTables(&s.table, nil)
	table.MigrateCaches(&s.cache, setnil)
	table.MigrateTables(&s.EpochTable, nil)
	err := s.MainDB.Close()
	if err != nil {
		return err
	}

	if s.EpochDB != nil {
		err = s.EpochDB.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// DropEpochDB drops existing epoch DB
func (s *Store) DropEpochDB() error {
	prevDb := s.EpochDB
	if prevDb != nil {
		err := prevDb.Close()
		if err != nil {
			return err
		}
		prevDb.Drop()
	}
	return nil
}

// OpenEpochDB makes new epoch DB
func (s *Store) OpenEpochDB(n consensus.Epoch) error {
	// Clear full LRU cache.
	s.cache.FrameRoots.Purge()

	s.EpochDB = s.GetEpochDB(n)
	table.MigrateTables(&s.EpochTable, s.EpochDB)
	return nil
}

/*
 * Utils:
 */

// set RLP value
func (s *Store) set(table kvdb.Store, key []byte, val interface{}) {
	buf, err := rlp.EncodeToBytes(val)
	if err != nil {
		s.crit(err)
	}

	if err := table.Put(key, buf); err != nil {
		s.crit(err)
	}
}

// get RLP value
func (s *Store) get(table kvdb.Store, key []byte, to interface{}) interface{} {
	buf, err := table.Get(key)
	if err != nil {
		s.crit(err)
	}
	if buf == nil {
		return nil
	}

	err = rlp.DecodeBytes(buf, to)
	if err != nil {
		s.crit(err)
	}
	return to
}

func (s *Store) makeCache(weight uint, size int) *simplewlru.Cache {
	cache, err := simplewlru.New(weight, size)
	if err != nil {
		s.crit(err)
	}
	return cache
}
