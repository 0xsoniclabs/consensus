// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package dagindexer

import (
	"fmt"
	"os"
	"testing"

	"github.com/0xsoniclabs/cacheutils/cachescale"
	"github.com/0xsoniclabs/kvdb/memorydb"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensustest"
	"github.com/0xsoniclabs/consensus/vecflushable"
	"github.com/0xsoniclabs/kvdb"
	"github.com/0xsoniclabs/kvdb/flushable"
	"github.com/0xsoniclabs/kvdb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

func TestDefaultConfigDummy(t *testing.T) {
	// DefaultConfig() is not used in this repo, but in order to not have it marked
	// as unused, we define a dummy function here.
	_ = DefaultConfig(cachescale.Ratio{Base: 100, Target: 1})
}

func BenchmarkIndex_Add_MemoryDB(b *testing.B) {
	dbProducer := func() kvdb.FlushableKVStore {
		return flushable.Wrap(memorydb.New())
	}
	benchmark_Index_Add(b, dbProducer)
}

func BenchmarkIndex_Add_vecflushable_NoBackup(b *testing.B) {
	// the total database produced by the test is roughly 2'000'000 bytes (checked
	// against multiple runs) so we set the limit to double that to ensure that
	// no offloading to levelDB occurs
	dbProducer := func() kvdb.FlushableKVStore {
		db, _ := tempLevelDB()
		return vecflushable.Wrap(db, 4000000)
	}
	benchmark_Index_Add(b, dbProducer)
}

func BenchmarkIndex_Add_vecflushable_Backup(b *testing.B) {
	// the total database produced by the test is roughly 2'000'000 bytes (checked
	// against multiple runs) so we set the limit to half of that to force the
	// database to unload the cache into leveldb halfway through.
	dbProducer := func() kvdb.FlushableKVStore {
		db, _ := tempLevelDB()
		return vecflushable.Wrap(db, 1000000)
	}
	benchmark_Index_Add(b, dbProducer)
}

func benchmark_Index_Add(b *testing.B, dbProducer func() kvdb.FlushableKVStore) {
	b.StopTimer()

	nodes := consensustest.GenNodes(70)
	ordered := make(consensus.Events, 0)
	consensustest.ForEachRandEvent(nodes, 10, 10, nil, consensustest.ForEachEvent{
		Process: func(e consensus.Event, name string) {
			ordered = append(ordered, e)
		},
	})

	validatorsBuilder := consensus.NewValidatorsBuilder()
	for _, peer := range nodes {
		validatorsBuilder.Set(peer, 1)
	}
	validators := validatorsBuilder.Build()
	events := make(map[consensus.EventHash]consensus.Event)
	getEvent := func(id consensus.EventHash) consensus.Event {
		return events[id]
	}
	for _, e := range ordered {
		events[e.ID()] = e
	}

	i := 0
	for {
		b.StopTimer()
		vecClock := NewIndex(func(err error) { panic(err) }, LiteConfig())
		vecClock.Reset(validators, dbProducer(), getEvent)
		b.StartTimer()
		for _, e := range ordered {
			err := vecClock.Add(e)
			if err != nil {
				panic(err)
			}
			vecClock.Flush()
			i++
			if i >= b.N {
				return
			}
		}
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

var (
	testASCIIScheme = `
a1.0   b1.0   c1.0   d1.0   e1.0
║      ║      ║      ║      ║
║      ╠──────╫───── d2.0   ║
║      ║      ║      ║      ║
║      b2.1 ──╫──────╣      e2.1
║      ║      ║      ║      ║
║      ╠──────╫───── d3.1   ║
a2.1 ──╣      ║      ║      ║
║      ║      ║      ║      ║
║      b3.2 ──╣      ║      ║
║      ║      ║      ║      ║
║      ╠──────╫───── d4.2   ║
║      ║      ║      ║      ║
║      ╠───── c2.2   ║      e3.2
║      ║      ║      ║      ║
`
)

type eventWithCreationTime struct {
	consensus.Event
	creationTime Timestamp
}

func (e *eventWithCreationTime) CreationTimePortable() Timestamp {
	return e.creationTime
}

func BenchmarkIndex_Add(b *testing.B) {
	b.StopTimer()
	ordered := make(consensus.Events, 0)
	nodes, _, _ := consensustest.ASCIIschemeForEach(testASCIIScheme, consensustest.ForEachEvent{
		Process: func(e consensus.Event, name string) {
			ordered = append(ordered, e)
		},
	})
	validatorsBuilder := consensus.NewValidatorsBuilder()
	for _, peer := range nodes {
		validatorsBuilder.Set(peer, 1)
	}
	validators := validatorsBuilder.Build()
	events := make(map[consensus.EventHash]consensus.Event)
	getEvent := func(id consensus.EventHash) consensus.Event {
		return events[id]
	}
	for _, e := range ordered {
		events[e.ID()] = e
	}

	vecClock := NewIndex(func(err error) { panic(err) }, LiteConfig())
	vecClock.Reset(validators, vecClock.WrapWithFlushable(memorydb.New()), getEvent)

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		vecClock.Reset(validators, vecClock.WrapWithFlushable(memorydb.New()), getEvent)
		b.StartTimer()
		for _, e := range ordered {
			err := vecClock.Add(&eventWithCreationTime{e, Timestamp(e.Seq())})
			if err != nil {
				panic(err)
			}
			i++
			if i >= b.N {
				break
			}
		}
	}
}
