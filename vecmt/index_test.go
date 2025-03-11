package vecmt

import (
	"fmt"
	"github.com/0xsoniclabs/consensus/kvdb"
	"github.com/0xsoniclabs/consensus/kvdb/flushable"
	"github.com/0xsoniclabs/consensus/kvdb/leveldb"
	"github.com/0xsoniclabs/consensus/vecflushable"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"io/ioutil"
	"testing"

	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/dag"
	"github.com/0xsoniclabs/consensus/inter/dag/tdag"
	"github.com/0xsoniclabs/consensus/inter/pos"
	"github.com/0xsoniclabs/consensus/kvdb/memorydb"
)

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

	nodes := tdag.GenNodes(70)
	ordered := make(dag.Events, 0)
	tdag.ForEachRandEvent(nodes, 10, 10, nil, tdag.ForEachEvent{
		Process: func(e dag.Event, name string) {
			ordered = append(ordered, e)
		},
	})

	validatorsBuilder := pos.NewBuilder()
	for _, peer := range nodes {
		validatorsBuilder.Set(peer, 1)
	}
	validators := validatorsBuilder.Build()
	events := make(map[hash.Event]dag.Event)
	getEvent := func(id hash.Event) dag.Event {
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
	dir, err := ioutil.TempDir("", "bench")
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
	dag.Event
	creationTime Timestamp
}

func (e *eventWithCreationTime) CreationTime() Timestamp {
	return e.creationTime
}

func BenchmarkIndex_Add(b *testing.B) {
	b.StopTimer()
	ordered := make(dag.Events, 0)
	nodes, _, _ := tdag.ASCIIschemeForEach(testASCIIScheme, tdag.ForEachEvent{
		Process: func(e dag.Event, name string) {
			ordered = append(ordered, e)
		},
	})
	validatorsBuilder := pos.NewBuilder()
	for _, peer := range nodes {
		validatorsBuilder.Set(peer, 1)
	}
	validators := validatorsBuilder.Build()
	events := make(map[hash.Event]dag.Event)
	getEvent := func(id hash.Event) dag.Event {
		return events[id]
	}
	for _, e := range ordered {
		events[e.ID()] = e
	}

	vecClock := NewIndex(func(err error) { panic(err) }, LiteConfig())
	vecClock.Reset(validators, memorydb.New(), getEvent)

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		vecClock.Reset(validators, memorydb.New(), getEvent)
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
