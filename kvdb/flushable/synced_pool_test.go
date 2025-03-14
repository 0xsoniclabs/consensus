// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package flushable

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/consensus/kvdb"
	"github.com/0xsoniclabs/consensus/kvdb/table"
	"github.com/0xsoniclabs/consensus/utils/byteutils"
)

func TestSyncedPoolUnderlying(t *testing.T) {
	require := require.New(t)
	const (
		N       = 1000
		dbname1 = "db1"
		dbname2 = "db2"
		tbname  = "table"
	)

	dbs := dbProducer("")
	pool := NewSyncedPool(dbs, []byte("flushID"))
	defer func() {
		err := pool.Close()
		require.NoError(err)
	}()

	db1, err := pool.GetUnderlying(dbname1)
	require.NoError(err)
	r1 := table.New(db1, []byte(tbname))

	fdb1, err := pool.OpenDB(dbname1)
	require.NoError(err)
	w1 := table.New(fdb1, []byte(tbname))
	defer w1.Close()

	fdb2, err := pool.OpenDB(dbname2)
	require.NoError(err)
	w2 := table.New(fdb2, []byte(tbname))

	db2, err := pool.GetUnderlying(dbname2)
	require.NoError(err)
	r2 := table.New(db2, []byte(tbname))

	_, err = pool.Initialize([]string{dbname1, dbname2}, nil)
	require.NoError(err)

	pushData := func(n uint32, w kvdb.Store) {
		const size uint32 = 10
		for i := size; i > 0; i-- {
			key := byteutils.Uint32ToBigEndian(i + size*n)
			require.NoError(w.Put(key, key))
		}
	}

	checkConsistency := func() {
		it := r1.NewIterator(nil, nil)
		defer it.Release()
		var prev uint32 = 0
		for it.Next() {
			key1 := it.Key()
			i := byteutils.BigEndianToUint32(key1)
			require.Equal(prev+1, i)
			prev = i

			key2, err := r2.Get(key1)
			require.NoError(err)
			require.Equal(key1, key2)
		}
	}

	pushData(0, w1)
	checkConsistency()

	pushData(0, w2)
	pool.Flush(nil)
	checkConsistency()

	pushData(1, w1)
	pushData(1, w2)
	checkConsistency()
	pool.Flush(nil)
	checkConsistency()
}
