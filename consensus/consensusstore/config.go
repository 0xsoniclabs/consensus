package consensusstore

import "github.com/0xsoniclabs/cacheutils/cachescale"

// StoreCacheConfig is a cache config for store db.
type StoreCacheConfig struct {
	// Cache size for Bases.
	BasesNum    uint
	BasesFrames int
}

// StoreConfig is a config for store db.
type StoreConfig struct {
	Cache StoreCacheConfig
}

// DefaultStoreConfig for livenet.
func DefaultStoreConfig(scale cachescale.Func) StoreConfig {
	return StoreConfig{
		StoreCacheConfig{
			BasesNum:    scale.U(1000),
			BasesFrames: scale.I(100),
		},
	}
}

// LiteStoreConfig is for tests or inmemory.
func LiteStoreConfig() StoreConfig {
	return DefaultStoreConfig(cachescale.Ratio{Base: 8 * 1024, Target: 16 * 1024})
}
