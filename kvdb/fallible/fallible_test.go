// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package fallible

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/consensus/kvdb"
	"github.com/0xsoniclabs/consensus/kvdb/memorydb"
)

func TestFallible(t *testing.T) {
	require := require.New(t)

	var (
		key  = []byte("test-key")
		key2 = []byte("test-key-2")
		val  = []byte("test-value")
		db   kvdb.Store
		err  error
	)

	mem := memorydb.New()
	w := Wrap(mem)
	db = w

	var v []byte
	v, err = db.Get(key)
	require.Nil(v)
	require.NoError(err)

	require.Panics(func() {
		_ = db.Put(key, val)
	})

	w.SetWriteCount(1)

	err = db.Put(key, val)
	require.NoError(err)

	require.Panics(func() {
		_ = db.Put(key, val)
	})

	err = db.Delete(key)
	require.Nil(err)

	count := w.GetWriteCount()
	require.Equal(-1, count)

	require.Panics(func() {
		db.Close()
	})

	require.Panics(func() {
		db.Drop()
	})

	w.SetWriteCount(2)
	count = w.GetWriteCount()
	require.Equal(2, count)

	err = db.Put(key, val)
	require.NoError(err)

	err = db.Put(key2, val)
	require.NoError(err)

	iterator := db.NewIterator([]byte("test"), nil)

	iterator.Next()
	require.Equal(key, iterator.Key())

	iterator.Next()
	require.Equal(key2, iterator.Key())
}
