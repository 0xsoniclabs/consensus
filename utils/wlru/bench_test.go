// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package wlru

import (
	"testing"

	lru "github.com/hashicorp/golang-lru"
)

func BenchmarkWeightedCache_Add(b *testing.B) {
	cache, _ := New(5000, 1000)
	data := make([]int, 1000)
	b.ResetTimer()
	for i := 1; i < b.N; i += len(data) {
		for j, d := range data {
			cache.Add(i*j, d, 5)
		}
	}
}

func BenchmarkCache_Add(b *testing.B) {
	cache, _ := lru.New(1000)
	data := make([]int, 1000)
	b.ResetTimer()
	for i := 1; i < b.N; i += len(data) {
		for j, d := range data {
			cache.Add(i*j, d)
		}
	}
}

func BenchmarkWeightedCache_Get(b *testing.B) {
	cache, _ := New(5000, 1000)
	data := make([]int, 1000)
	for j, d := range data {
		cache.Add(j, d, 5)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(i % (len(data) * 2))
	}
}

func BenchmarkCache_Get(b *testing.B) {
	cache, _ := lru.New(1000)
	data := make([]int, 1000)
	for j, d := range data {
		cache.Add(j, d)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(i % (len(data) * 2))
	}
}
