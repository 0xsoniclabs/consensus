// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package vecflushable

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/0xsoniclabs/consensus/utils/byteutils"
	"github.com/0xsoniclabs/kvdb"
	"github.com/0xsoniclabs/kvdb/devnulldb"
	"github.com/0xsoniclabs/kvdb/leveldb"
)

// TestVecflushableNoBackup tests normal operation of vecflushable, before and after
// flush, while the size remains under the limit.
func TestVecflushableNoBackup(t *testing.T) {
	// we set the limit at 100000 bytes so
	// the underlying cache should not be unloaded to leveldb

	backupDB, _ := tempLevelDB()
	vecflushable := Wrap(backupDB, 100000)

	putOp := func(key []byte, value []byte) {
		err := vecflushable.Put(key, value)
		if err != nil {
			t.Error(err)
		}
	}

	getOp := func(key []byte, val []byte) {
		v, err := vecflushable.Get(key)
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(v, val) {
			t.Errorf("retrieved value does not match expected value")
		}
	}

	totalItems := 10
	keySize := 8
	valSize := 8
	expectedNotFlushedSize := totalItems * mapMemEst(keySize, valSize)

	loopOp(putOp, totalItems)

	assert.Equal(t, totalItems, vecflushable.NotFlushedPairs())
	assert.Equal(t, expectedNotFlushedSize, vecflushable.NotFlushedSizeEst())
	assert.Equal(t, 0, vecflushable.underlying.memSize)

	loopOp(getOp, totalItems)

	err := vecflushable.Flush()
	assert.NoError(t, err)

	expectedUnderlyingCacheSize := totalItems * mapMemEst(keySize, valSize)

	assert.Equal(t, 0, vecflushable.NotFlushedPairs())
	assert.Equal(t, 0, vecflushable.NotFlushedSizeEst())
	assert.Equal(t, expectedUnderlyingCacheSize, vecflushable.underlying.memSize)

	loopOp(getOp, totalItems)
}

// TestVecflushableBackup tests that the native map is unloaded to persistent
// storage when size exceeds the limit, respecting the eviction threshold.
func TestVecflushableBackup(t *testing.T) {
	// we set the limit at 144 bytes and insert 1160 bytes [10 * (100 + 8 + 8)]
	// the eviction threshold is 72 bytes
	//
	// unfolding:
	//
	// - the sizeLimit is first hit after inserting 6 items (6*116 = 696).
	//		* the first 3 items (3*16=48) are unloaded from the map and copied to level db
	//   => | cache = 3 | memSize = 348 | levelDB = 3
	//
	// - after inserting 3 more items, the cache limit is hit again.
	// 		* the next 3 items are unloaded from the map and copied to level db
	// 	 => | cache = 3 | memSize = 72 | levelDB = 6
	//
	// - after inserting the last item, the size of cache is still under the limit
	//   => | cache = 4 | memSize = 96 | levelDB = 6

	backupDB, _ := tempLevelDB()
	vecflushable := wrap(backupDB, 696-1, 48)

	putOp := func(key []byte, value []byte) {
		if err := vecflushable.Put(key, value); err != nil {
			t.Error(err)
		}
		if err := vecflushable.Flush(); err != nil {
			t.Error(err)
		}
	}

	totalItems := 10
	expectedUnderlyingCacheCount := 4
	expectedUnderlyingCacheSize := expectedUnderlyingCacheCount * 116

	loopOp(putOp, totalItems)

	assert.Equal(t, 0, vecflushable.NotFlushedPairs())
	assert.Equal(t, 0, vecflushable.NotFlushedSizeEst())
	assert.Equal(t, expectedUnderlyingCacheCount, len(vecflushable.underlying.cache))
	assert.Equal(t, expectedUnderlyingCacheSize, vecflushable.underlying.memSize)

	getOp := func(key []byte, val []byte) {
		v, err := vecflushable.Get(key)
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(v, val) {
			t.Errorf("retrieved value does not match expected value")
		}
	}

	// check that we can retrieve items from the backup store
	loopOp(getOp, totalItems)
}

// TestVecflushableUpdateValue tests that updating a value (as opposed to inserting
// a new value) does not increase the size of the cache.
func TestVecflushableUpdateValue(t *testing.T) {
	backupDB, _ := tempLevelDB()
	vecflushable := Wrap(backupDB, 1000)

	key0 := byteutils.Uint64ToBigEndian(uint64(0))
	bigVal := make([]byte, 70)
	for i := 0; i < 70; i++ {
		bigVal[i] = 0xff
	}

	for i := 0; i < 2; i++ {
		if err := vecflushable.Put(key0, bigVal); err != nil {
			t.Error(err)
		}
		if err := vecflushable.Flush(); err != nil {
			t.Error(err)
		}
	}

	assert.Equal(t, 0, vecflushable.NotFlushedPairs())
	assert.Equal(t, 0, vecflushable.NotFlushedSizeEst())
	assert.Equal(t, 1, len(vecflushable.underlying.cache))
	assert.Equal(t, 178, vecflushable.underlying.memSize)

	key1 := byteutils.Uint64ToBigEndian(uint64(1))
	for i := 0; i < 2; i++ {
		if err := vecflushable.Put(key1, bigVal); err != nil {
			t.Error(err)
		}
	}
	if err := vecflushable.Flush(); err != nil {
		t.Error(err)
	}

	assert.Equal(t, 0, vecflushable.NotFlushedPairs())
	assert.Equal(t, 0, vecflushable.NotFlushedSizeEst())
	assert.Equal(t, 2, len(vecflushable.underlying.cache))
	assert.Equal(t, 356, vecflushable.underlying.memSize)
}

func TestSizeBenchmark(t *testing.T) {
	for _, numItems := range []int{10, 100, 1000, 10000, 100000, 1000000, 10000000} {
		t.Run(strconv.Itoa(numItems), func(t *testing.T) {
			res := testing.Benchmark(func(b *testing.B) {
				b.ReportAllocs()
				vecflushable := Wrap(devnulldb.New(), 1_000_000_000)
				loopOp(
					func(key []byte, value []byte) {
						err := vecflushable.Put(key, value)
						if err != nil {
							t.Error(err)
						}
						err = vecflushable.Flush()
						if err != nil {
							t.Error(err)
						}
					},
					numItems,
				)
				runtime.KeepAlive(vecflushable) // prevent GC
			})
			s := res.MemBytes / uint64(numItems)
			fmt.Printf("items: %d, avg bytes/item: %v\n", numItems, s)
		})
	}
}

func BenchmarkPutAndFlush(b *testing.B) {
	b.ReportAllocs()
	vecflushable := Wrap(devnulldb.New(), 1_000_000_000)
	for op := 0; op < b.N; op++ {
		step := op & 0xff
		key := byteutils.Uint64ToBigEndian(uint64(step << 48))
		val := byteutils.Uint64ToBigEndian(uint64(step))
		err := vecflushable.Put(key, val)
		if err != nil {
			b.Error(err)
		}
		err = vecflushable.Flush()
		if err != nil {
			b.Error(err)
		}
	}
}

func loopOp(operation func(key []byte, val []byte), iterations int) {
	for op := 0; op < iterations; op++ {
		step := op & 0xff
		key := byteutils.Uint64ToBigEndian(uint64(step << 48))
		val := byteutils.Uint64ToBigEndian(uint64(step))
		operation(key, val)
	}
}

func tempLevelDB() (kvdb.Store, error) {
	cache16mb := func(string) (int, int) {
		return 16 * opt.MiB, 64
	}
	dir, err := os.MkdirTemp("", "bench")
	if err != nil {
		panic(fmt.Sprintf("can't create temporary directory %s: %v", dir, err))
	}
	disk := leveldb.NewProducer(dir, cache16mb)
	ldb, _ := disk.OpenDB("0")
	return ldb, nil
}

func TestNilParentWrap(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, got nil")
		}
	}()
	_ = Wrap(nil, 1000)
}

func TestNilPut(t *testing.T) {
	backupDB, _ := tempLevelDB()
	vecflushable := Wrap(backupDB, 1000)

	err := vecflushable.Put([]byte("a"), nil)
	if err == nil {
		t.Error("expected error, got nil")
	}

	err = vecflushable.Put(nil, []byte("a"))
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestHasInModified(t *testing.T) {
	backupDB, _ := tempLevelDB()
	vecflushable := Wrap(backupDB, 1000)

	key := byteutils.Uint64ToBigEndian(uint64(0))
	val := byteutils.Uint64ToBigEndian(uint64(1))

	if err := vecflushable.Put(key, val); err != nil {
		t.Error(err)
	}

	has, err := vecflushable.Has(key)
	if err != nil {
		t.Error(err)
	}
	assert.True(t, has)

	has, err = vecflushable.Has(byteutils.Uint64ToBigEndian(uint64(2)))
	if err != nil {
		t.Error(err)
	}
	assert.False(t, has)
}

func TestHasInBacked(t *testing.T) {
	backupDB, _ := tempLevelDB()
	vecflushable := Wrap(backupDB, 1000)

	key := byteutils.Uint64ToBigEndian(uint64(0))
	val := byteutils.Uint64ToBigEndian(uint64(1))

	if err := vecflushable.Put(key, val); err != nil {
		t.Error(err)
	}
	// Force contents to underlying
	if err := vecflushable.Flush(); err != nil {
		t.Fatal(err)
	}

	has, err := vecflushable.Has(key)
	if err != nil {
		t.Error(err)
	}
	assert.True(t, has)

	has, err = vecflushable.Has(byteutils.Uint64ToBigEndian(uint64(2)))
	if err != nil {
		t.Error(err)
	}
	assert.False(t, has)
}

func TestHasClosed(t *testing.T) {
	backupDB, _ := tempLevelDB()
	vecflushable := Wrap(backupDB, 1000)

	key := byteutils.Uint64ToBigEndian(uint64(0))
	val := byteutils.Uint64ToBigEndian(uint64(1))

	if err := vecflushable.Put(key, val); err != nil {
		t.Error(err)
	}

	if err := vecflushable.Close(); err != nil {
		t.Fatal(err)
	}

	has, err := vecflushable.Has(key)
	if err == nil {
		t.Error("expected error, got nil")
	}
	assert.False(t, has)
}

func TestGetClosed(t *testing.T) {
	backupDB, _ := tempLevelDB()
	vecflushable := Wrap(backupDB, 1000)

	key := byteutils.Uint64ToBigEndian(uint64(0))
	val := byteutils.Uint64ToBigEndian(uint64(1))

	if err := vecflushable.Put(key, val); err != nil {
		t.Error(err)
	}

	if err := vecflushable.Close(); err != nil {
		t.Fatal(err)
	}

	v, err := vecflushable.Get(key)
	if err == nil {
		t.Error("expected error, got nil")
	}
	assert.Nil(t, v)
}

func TestFlushClosed(t *testing.T) {
	backupDB, _ := tempLevelDB()
	vecflushable := Wrap(backupDB, 1000)

	key := byteutils.Uint64ToBigEndian(uint64(0))
	val := byteutils.Uint64ToBigEndian(uint64(1))

	if err := vecflushable.Put(key, val); err != nil {
		t.Error(err)
	}

	if err := vecflushable.Close(); err != nil {
		t.Fatal(err)
	}

	err := vecflushable.Flush()
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestCloseClosed(t *testing.T) {
	backupDB, _ := tempLevelDB()
	vecflushable := Wrap(backupDB, 1000)

	if err := vecflushable.Close(); err != nil {
		t.Fatal(err)
	}

	err := vecflushable.Close()
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestUnimplementedDrop(t *testing.T) {
	backupDB, _ := tempLevelDB()
	vecflushable := Wrap(backupDB, 1000)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, got nil")
		}
	}()

	vecflushable.Drop()

}

func TestUnimplementedAncientDatadir(t *testing.T) {
	backupDB, err := tempLevelDB()
	if err != nil {
		t.Fatal(err)
	}
	vecflushable := Wrap(backupDB, 1000)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, got nil")
		}
	}()

	if _, err := vecflushable.AncientDatadir(); err != nil {
		t.Fatal(err)
	}
}

func TestUnimplementedDelete(t *testing.T) {
	backupDB, _ := tempLevelDB()
	vecflushable := Wrap(backupDB, 1000)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, got nil")
		}
	}()

	if err := vecflushable.Delete([]byte("a")); err != nil {
		t.Fatal(err)
	}
}

func TestUnimplementedGetSnapshot(t *testing.T) {
	backupDB, _ := tempLevelDB()
	vecflushable := Wrap(backupDB, 1000)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, got nil")
		}
	}()

	if _, err := vecflushable.GetSnapshot(); err != nil {
		t.Fatal(err)
	}
}

func TestUnimplementedNewIterator(t *testing.T) {
	backupDB, _ := tempLevelDB()
	vecflushable := Wrap(backupDB, 1000)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, got nil")
		}
	}()

	vecflushable.NewIterator([]byte("a"), []byte("b"))
}

func TestUnimplementedStat(t *testing.T) {
	backupDB, _ := tempLevelDB()
	vecflushable := Wrap(backupDB, 1000)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, got nil")
		}
	}()

	if _, err := vecflushable.Stat(); err != nil {
		t.Fatal(err)
	}
}

func TestUnimplementedCompact(t *testing.T) {
	backupDB, _ := tempLevelDB()
	vecflushable := Wrap(backupDB, 1000)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, got nil")
		}
	}()

	if err := vecflushable.Compact([]byte("a"), []byte("b")); err != nil {
		t.Fatal(err)
	}
}

func TestUnimplementedNewBatch(t *testing.T) {
	backupDB, _ := tempLevelDB()
	vecflushable := Wrap(backupDB, 1000)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, got nil")
		}
	}()

	vecflushable.NewBatch()
}
