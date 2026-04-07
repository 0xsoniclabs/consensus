// Copyright (c) 2026 Sonic Operations Ltd
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
	"github.com/0xsoniclabs/cacheutils/cachescale"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// indexCacheConfig - config for cache sizes of Engine
type indexCacheConfig struct {
	HighestBeforeTimeSize uint
	DBCache               int
	HighestBeforeSeqSize  uint
	LowestAfterSeqSize    uint
}

// IndexConfig - Engine config (cache sizes)
type IndexConfig struct {
	Caches indexCacheConfig
}

// DefaultConfig returns default index config
func DefaultConfig(scale cachescale.Func) IndexConfig {
	return IndexConfig{
		Caches: indexCacheConfig{
			HighestBeforeTimeSize: scale.U(160 * 1024),
			DBCache:               scale.I(10 * opt.MiB),
			HighestBeforeSeqSize:  scale.U(160 * 1024),
			LowestAfterSeqSize:    scale.U(160 * 1024),
		},
	}
}

// LiteConfig returns default index config for tests
func LiteConfig() IndexConfig {
	scale := cachescale.Ratio{Base: 100, Target: 1}
	return IndexConfig{
		Caches: indexCacheConfig{
			HighestBeforeTimeSize: 4 * 1024,
			HighestBeforeSeqSize:  scale.U(160 * 1024),
			LowestAfterSeqSize:    scale.U(160 * 1024),
		},
	}
}
