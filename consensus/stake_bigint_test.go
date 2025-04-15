// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensus

import (
	"math/big"
	"reflect"
	"testing"
)

func TestNewBigBuilder_CreatesEmptyBuilder(t *testing.T) {
	builder := NewBigBuilder()

	if len(builder) != 0 {
		t.Errorf("Expected empty builder, got %v items", len(builder))
	}
}

func TestValidatorsBigBuilder_SetAddsValidatorWeight(t *testing.T) {
	builder := NewBigBuilder()
	id := ValidatorID(1)
	weight := big.NewInt(100)

	builder.Set(id, weight)

	if len(builder) != 1 {
		t.Errorf("Expected 1 validator, got %v", len(builder))
	}

	if builder[id].Cmp(weight) != 0 {
		t.Errorf("Expected weight %v, got %v", weight, builder[id])
	}
}

func TestValidatorsBigBuilder_SetUpdatesExistingValidator(t *testing.T) {
	builder := NewBigBuilder()
	id := ValidatorID(1)

	builder.Set(id, big.NewInt(100))

	newWeight := big.NewInt(200)
	builder.Set(id, newWeight)

	if len(builder) != 1 {
		t.Errorf("Expected 1 validator, got %v", len(builder))
	}

	if builder[id].Cmp(newWeight) != 0 {
		t.Errorf("Expected updated weight %v, got %v", newWeight, builder[id])
	}
}

func TestValidatorsBigBuilder_SetRemovesValidatorWithZeroWeight(t *testing.T) {
	builder := NewBigBuilder()
	id := ValidatorID(1)

	builder.Set(id, big.NewInt(100))
	builder.Set(id, big.NewInt(0))

	if len(builder) != 0 {
		t.Errorf("Expected validator to be removed, but got %v validators", len(builder))
	}

	if _, exists := builder[id]; exists {
		t.Errorf("Expected validator %v to be removed", id)
	}
}

func TestValidatorsBigBuilder_SetRemovesValidatorWithNilWeight(t *testing.T) {
	builder := NewBigBuilder()
	id := ValidatorID(1)

	builder.Set(id, big.NewInt(100))
	builder.Set(id, nil)

	if len(builder) != 0 {
		t.Errorf("Expected validator to be removed, but got %v validators", len(builder))
	}

	if _, exists := builder[id]; exists {
		t.Errorf("Expected validator %v to be removed", id)
	}
}

func TestValidatorsBigBuilder_TotalWeightCalculatesCorrectSum(t *testing.T) {
	builder := NewBigBuilder()

	builder.Set(ValidatorID(1), big.NewInt(100))
	builder.Set(ValidatorID(2), big.NewInt(200))
	builder.Set(ValidatorID(3), big.NewInt(300))

	total := builder.TotalWeight()
	expected := big.NewInt(600)

	if total.Cmp(expected) != 0 {
		t.Errorf("Expected total weight %v, got %v", expected, total)
	}
}

func TestValidatorsBigBuilder_BuildCreatesValidValidatorsObject(t *testing.T) {
	builder := NewBigBuilder()

	builder.Set(ValidatorID(1), big.NewInt(100))
	builder.Set(ValidatorID(2), big.NewInt(200))

	validators := builder.Build()

	if validators == nil {
		t.Fatal("Expected non-nil Validators object")
	}

	expectedValues := map[ValidatorID]Weight{
		ValidatorID(1): Weight(100),
		ValidatorID(2): Weight(200),
	}

	if !reflect.DeepEqual(validators.values, expectedValues) {
		t.Errorf("Expected validators values %v, got %v", expectedValues, validators.values)
	}

	if validators.cache.totalWeight != Weight(300) {
		t.Errorf("Expected total weight %v, got %v", Weight(300), validators.cache.totalWeight)
	}

	expectedIDs := []ValidatorID{ValidatorID(1), ValidatorID(2)}
	expectedWeights := []Weight{Weight(100), Weight(200)}

	if len(validators.cache.ids) != len(expectedIDs) {
		t.Errorf("Expected %d IDs in cache, got %d", len(expectedIDs), len(validators.cache.ids))
	}

	for _, id := range expectedIDs {
		found := false
		for _, cacheID := range validators.cache.ids {
			if id == cacheID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected ID %v in cache, but not found", id)
		}
	}

	if len(validators.cache.weights) != len(expectedWeights) {
		t.Errorf("Expected %d weights in cache, got %d", len(expectedWeights), len(validators.cache.weights))
	}

	if validators.cache.totalWeight != Weight(300) {
		t.Errorf("Expected total weight %v, got %v", Weight(300), validators.cache.totalWeight)
	}

	for id, index := range validators.cache.indexes {
		if validators.cache.ids[index] != id {
			t.Errorf("Inconsistency in cache: index %v for ID %v points to ID %v",
				index, id, validators.cache.ids[index])
		}

		expectedWeight := expectedValues[id]
		if validators.cache.weights[index] != expectedWeight {
			t.Errorf("Inconsistency in cache: index %v for ID %v has weight %v, expected %v",
				index, id, validators.cache.weights[index], expectedWeight)
		}
	}
}

func TestValidatorsBigBuilder_BuildHandlesLargeWeights(t *testing.T) {
	builder := NewBigBuilder()

	// Create weights that exceed 31 bits
	largeWeight1 := new(big.Int).Lsh(big.NewInt(1), 32) // 2^32
	largeWeight2 := new(big.Int).Lsh(big.NewInt(1), 33) // 2^33

	builder.Set(ValidatorID(1), largeWeight1)
	builder.Set(ValidatorID(2), largeWeight2)

	validators := builder.Build()

	if validators == nil {
		t.Fatal("Expected non-nil Validators object")
	}

	// After right shift by (totalBits - 31), weights should be scaled down
	// totalBits = largeWeight1 + largeWeight2 = 2^32 + 2^33 = 3*2^32, which has BitLen of 34
	// shift = 34 - 31 = 3
	// largeWeight1 >> 3 = 2^32 >> 3 = 2^29 = 536,870,912
	// largeWeight2 >> 3 = 2^33 >> 3 = 2^30 = 1,073,741,824
	expectedValues := map[ValidatorID]Weight{
		ValidatorID(1): Weight(536870912),  // 2^32 >> 3
		ValidatorID(2): Weight(1073741824), // 2^33 >> 3
	}

	if !reflect.DeepEqual(validators.values, expectedValues) {
		t.Errorf("Expected validators values %v, got %v", expectedValues, validators.values)
	}

	expectedTotal := Weight(536870912 + 1073741824)
	if validators.cache.totalWeight != expectedTotal {
		t.Errorf("Expected total weight %v, got %v", expectedTotal, validators.cache.totalWeight)
	}
}
