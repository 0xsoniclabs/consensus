package main

import (
	"fmt"
	"io"

	"github.com/0xsoniclabs/consensus/abft"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/kvdb"
	"github.com/0xsoniclabs/consensus/kvdb/memorydb"
	"github.com/0xsoniclabs/consensus/utils/adapters"
	"github.com/0xsoniclabs/consensus/vecfc"
)

func main() {
	openEDB := func(epoch idx.Epoch) kvdb.Store {
		return memorydb.New()
	}

	crit := func(err error) {
		panic(err)
	}

	store := abft.NewStore(memorydb.New(), openEDB, crit, abft.LiteStoreConfig())
	restored := abft.NewIndexedLachesis(store, nil, &adapters.VectorToDagIndexer{Index: vecfc.NewIndex(crit, vecfc.LiteConfig())}, crit, abft.LiteConfig())

	// prevent compiler optimizations
	fmt.Fprint(io.Discard, restored == nil)
}
