// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package abft

import "github.com/0xsoniclabs/consensus/utils/cachescale"

type Config struct {
	// Suppresses the frame missmatch panic - used only for importing older historical event files, disabled by default
	SuppressFramePanic bool
}

// DefaultConfig for livenet.
func DefaultConfig() Config {
	return Config{
		SuppressFramePanic: false,
	}
}

// LiteConfig is for tests or inmemory.
func LiteConfig() Config {
	return Config{
		SuppressFramePanic: false,
	}
}

// StoreCacheConfig is a cache config for store db.
type StoreCacheConfig struct {
	// Cache size for Roots.
	RootsNum    uint
	RootsFrames int
}

// StoreConfig is a config for store db.
type StoreConfig struct {
	Cache StoreCacheConfig
}

// DefaultStoreConfig for livenet.
func DefaultStoreConfig(scale cachescale.Func) StoreConfig {
	return StoreConfig{
		StoreCacheConfig{
			RootsNum:    scale.U(1000),
			RootsFrames: scale.I(100),
		},
	}
}

// LiteStoreConfig is for tests or inmemory.
func LiteStoreConfig() StoreConfig {
	return DefaultStoreConfig(cachescale.Ratio{Base: 20, Target: 1})
}
