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
	"bytes"
	"fmt"
	"math"
	"math/big"
	"testing"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidators(t *testing.T) {
	b := NewValidatorsBuilder()

	assert.NotNil(t, b)
	assert.NotNil(t, b.Build())

	assert.Equal(t, ValidatorIndex(0), b.Build().Len())
}

func TestValidators_Set(t *testing.T) {
	b := NewValidatorsBuilder()

	b.Set(1, 1)
	b.Set(2, 2)
	b.Set(3, 3)
	b.Set(4, 4)
	b.Set(5, 5)

	v := b.Build()

	assert.Equal(t, ValidatorIndex(5), v.Len())
	assert.Equal(t, Weight(15), v.TotalWeight())

	b.Set(1, 10)
	b.Set(3, 30)

	v = b.Build()

	assert.Equal(t, ValidatorIndex(5), v.Len())
	assert.Equal(t, Weight(51), v.TotalWeight())

	b.Set(2, 0)
	b.Set(5, 0)

	v = b.Build()

	assert.Equal(t, ValidatorIndex(3), v.Len())
	assert.Equal(t, Weight(44), v.TotalWeight())

	b.Set(4, 0)
	b.Set(3, 0)
	b.Set(1, 0)

	v = b.Build()

	assert.Equal(t, ValidatorIndex(0), v.Len())
	assert.Equal(t, Weight(0), v.TotalWeight())
}

func TestValidators_Get(t *testing.T) {
	b := NewValidatorsBuilder()

	b.Set(0, 1)
	b.Set(2, 2)
	b.Set(3, 3)
	b.Set(4, 4)
	b.Set(7, 5)

	v := b.Build()

	assert.Equal(t, Weight(1), v.Get(0))
	assert.Equal(t, Weight(0), v.Get(1))
	assert.Equal(t, Weight(2), v.Get(2))
	assert.Equal(t, Weight(3), v.Get(3))
	assert.Equal(t, Weight(4), v.Get(4))
	assert.Equal(t, Weight(0), v.Get(5))
	assert.Equal(t, Weight(0), v.Get(6))
	assert.Equal(t, Weight(5), v.Get(7))
}

func TestValidators_Iterate(t *testing.T) {
	b := NewValidatorsBuilder()

	b.Set(1, 1)
	b.Set(2, 2)
	b.Set(3, 3)
	b.Set(4, 4)
	b.Set(5, 5)

	v := b.Build()

	count := 0
	sum := 0

	for _, id := range v.IDs() {
		count++
		sum += int(v.Get(id))
	}

	assert.Equal(t, 5, count)
	assert.Equal(t, 15, sum)
}

func TestValidators_Copy(t *testing.T) {
	b := NewValidatorsBuilder()

	b.Set(1, 1)
	b.Set(2, 2)
	b.Set(3, 3)
	b.Set(4, 4)
	b.Set(5, 5)

	v := b.Build()
	vv := v.Copy()

	assert.Equal(t, v.values, vv.values)

	assert.NotEqual(t, unsafe.Pointer(&v.values), unsafe.Pointer(&vv.values))
	assert.NotEqual(t, unsafe.Pointer(&v.cache.indexes), unsafe.Pointer(&vv.cache.indexes))
	assert.NotEqual(t, unsafe.Pointer(&v.cache.ids), unsafe.Pointer(&vv.cache.ids))
	assert.NotEqual(t, unsafe.Pointer(&v.cache.weights), unsafe.Pointer(&vv.cache.weights))
}

func maxBig(n uint) *big.Int {
	max := new(big.Int).Lsh(common.Big1, n)
	max.Sub(max, common.Big1)
	return max
}

func TestValidators_Big(t *testing.T) {
	max := Weight(math.MaxUint32 >> 1)

	b := NewBigBuilder()

	b.Set(1, big.NewInt(1))
	v := b.Build()
	assert.Equal(t, Weight(1), v.TotalWeight())
	assert.Equal(t, Weight(1), v.Get(1))

	b.Set(2, big.NewInt(int64(max)-1))
	v = b.Build()
	assert.Equal(t, max, v.TotalWeight())
	assert.Equal(t, Weight(1), v.Get(1))
	assert.Equal(t, max-1, v.Get(2))

	b.Set(3, big.NewInt(1))
	v = b.Build()
	assert.Equal(t, max/2, v.TotalWeight())
	assert.Equal(t, Weight(0), v.Get(1))
	assert.Equal(t, max/2, v.Get(2))
	assert.Equal(t, Weight(0), v.Get(3))

	b.Set(4, big.NewInt(2))
	v = b.Build()
	assert.Equal(t, max/2+1, v.TotalWeight())
	assert.Equal(t, Weight(0), v.Get(1))
	assert.Equal(t, max/2, v.Get(2))
	assert.Equal(t, Weight(0), v.Get(3))
	assert.Equal(t, Weight(1), v.Get(4))

	b.Set(5, maxBig(60))
	v = b.Build()
	assert.Equal(t, Weight(0x40000000), v.TotalWeight())
	assert.Equal(t, Weight(0), v.Get(1))
	assert.Equal(t, Weight(0x1), v.Get(2))
	assert.Equal(t, Weight(0), v.Get(3))
	assert.Equal(t, Weight(0), v.Get(4))
	assert.Equal(t, max/2, v.Get(5))

	b.Set(1, maxBig(501))
	b.Set(2, maxBig(502))
	b.Set(3, maxBig(503))
	b.Set(4, maxBig(504))
	b.Set(5, maxBig(515))
	v = b.Build()
	assert.Equal(t, Weight(0x400efffb), v.TotalWeight())
	assert.Equal(t, Weight(0xffff), v.Get(1))
	assert.Equal(t, Weight(0x1ffff), v.Get(2))
	assert.Equal(t, Weight(0x3ffff), v.Get(3))
	assert.Equal(t, Weight(0x7ffff), v.Get(4))
	assert.Equal(t, Weight(0x3fffffff), v.Get(5))

	for v := ValidatorID(1); v <= 5000; v++ {
		b.Set(v, new(big.Int).Mul(big.NewInt(int64(v)), maxBig(400)))
	}
	v = b.Build()
	assert.Equal(t, Weight(0x5f62de78), v.TotalWeight())
	assert.Equal(t, Weight(0x7f), v.Get(1))
	assert.Equal(t, Weight(0xff), v.Get(2))
	assert.Equal(t, Weight(0x17f), v.Get(3))
	assert.Equal(t, Weight(0x4e1ff), v.Get(2500))
	assert.Equal(t, Weight(0x9c37f), v.Get(4999))
	assert.Equal(t, Weight(0x9c3ff), v.Get(5000))
}

func TestArrayToValidators(t *testing.T) {
	ids := []ValidatorID{1, 2, 3, 4, 5}
	weights := []Weight{10, 20, 30, 40, 50}
	v := ArrayToValidators(ids, weights)
	assert.Equal(t, ValidatorIndex(5), v.Len())
	assert.Equal(t, Weight(150), v.TotalWeight())
	for i, id := range ids {
		assert.Equal(t, weights[i], v.Get(id))
	}

	v = ArrayToValidators([]ValidatorID{}, []Weight{})
	assert.Equal(t, ValidatorIndex(0), v.Len())
	assert.Equal(t, Weight(0), v.TotalWeight())

	v = ArrayToValidators([]ValidatorID{42}, []Weight{100})
	assert.Equal(t, ValidatorIndex(1), v.Len())
	assert.Equal(t, Weight(100), v.TotalWeight())
	assert.Equal(t, Weight(100), v.Get(42))

	ids = []ValidatorID{5, 3, 1, 4, 2}
	weights = []Weight{5, 3, 1, 4, 2}
	v = ArrayToValidators(ids, weights)
	expectedSortedIDs := []ValidatorID{5, 4, 3, 2, 1}
	assert.Equal(t, expectedSortedIDs, v.SortedIDs())
	expectedSortedWeights := []Weight{5, 4, 3, 2, 1}
	assert.Equal(t, expectedSortedWeights, v.SortedWeights())
}

func TestValidators_GetIdx(t *testing.T) {
	ids := []ValidatorID{5, 3, 1, 4, 2}
	weights := []Weight{50, 30, 10, 40, 20}
	v := ArrayToValidators(ids, weights)

	// IDs should be sorted by weight (descending) then by ID (ascending)
	assert.Equal(t, ValidatorIndex(0), v.GetIdx(5))
	assert.Equal(t, ValidatorIndex(1), v.GetIdx(4))
	assert.Equal(t, ValidatorIndex(2), v.GetIdx(3))
	assert.Equal(t, ValidatorIndex(3), v.GetIdx(2))
	assert.Equal(t, ValidatorIndex(4), v.GetIdx(1))

	// Non-existent ID should return 0 index
	assert.Equal(t, ValidatorIndex(0), v.GetIdx(99))
}

func TestValidators_GetID(t *testing.T) {
	ids := []ValidatorID{5, 3, 1, 4, 2}
	weights := []Weight{50, 30, 10, 40, 20}
	v := ArrayToValidators(ids, weights)

	// Expected order after sorting by weight then ID
	assert.Equal(t, ValidatorID(5), v.GetID(0))
	assert.Equal(t, ValidatorID(4), v.GetID(1))
	assert.Equal(t, ValidatorID(3), v.GetID(2))
	assert.Equal(t, ValidatorID(2), v.GetID(3))
	assert.Equal(t, ValidatorID(1), v.GetID(4))
}

func TestValidators_GetWeightByIdx(t *testing.T) {
	ids := []ValidatorID{5, 3, 1, 4, 2}
	weights := []Weight{50, 30, 10, 40, 20}
	v := ArrayToValidators(ids, weights)

	// Weights should be sorted in descending order
	assert.Equal(t, Weight(50), v.GetWeightByIdx(0))
	assert.Equal(t, Weight(40), v.GetWeightByIdx(1))
	assert.Equal(t, Weight(30), v.GetWeightByIdx(2))
	assert.Equal(t, Weight(20), v.GetWeightByIdx(3))
	assert.Equal(t, Weight(10), v.GetWeightByIdx(4))
}

func TestValidators_Exists(t *testing.T) {
	b := NewValidatorsBuilder()
	b.Set(1, 10)
	b.Set(3, 30)
	b.Set(5, 50)

	v := b.Build()

	assert.True(t, v.Exists(1))
	assert.True(t, v.Exists(3))
	assert.True(t, v.Exists(5))
	assert.False(t, v.Exists(2))
	assert.False(t, v.Exists(4))
	assert.False(t, v.Exists(0))
	assert.False(t, v.Exists(99))
}

func TestValidators_IDs(t *testing.T) {
	ids := []ValidatorID{5, 3, 1, 4, 2}
	weights := []Weight{50, 30, 10, 40, 20}
	v := ArrayToValidators(ids, weights)

	// IDs should match the sorted order (by weight then ID)
	expectedIDs := []ValidatorID{5, 4, 3, 2, 1}
	assert.Equal(t, expectedIDs, v.IDs())
}

func TestValidators_Idxs(t *testing.T) {
	ids := []ValidatorID{5, 3, 1, 4, 2}
	weights := []Weight{50, 30, 10, 40, 20}
	v := ArrayToValidators(ids, weights)

	idxs := v.Idxs()
	assert.Equal(t, ValidatorIndex(0), idxs[5])
	assert.Equal(t, ValidatorIndex(1), idxs[4])
	assert.Equal(t, ValidatorIndex(2), idxs[3])
	assert.Equal(t, ValidatorIndex(3), idxs[2])
	assert.Equal(t, ValidatorIndex(4), idxs[1])
}

func TestValidators_Builder(t *testing.T) {
	b := NewValidatorsBuilder()
	b.Set(1, 10)
	b.Set(3, 30)
	b.Set(5, 50)

	v := b.Build()

	newBuilder := v.Builder()

	assert.Equal(t, Weight(10), newBuilder[1])
	assert.Equal(t, Weight(30), newBuilder[3])
	assert.Equal(t, Weight(50), newBuilder[5])

	// Modify the builder and check that it doesn't affect the original validators
	newBuilder.Set(1, 100)
	newBuilder.Set(7, 70)

	assert.Equal(t, Weight(10), v.Get(1))
	assert.Equal(t, Weight(0), v.Get(7))

	// Build a new validator set from the modified builder
	newV := newBuilder.Build()
	assert.Equal(t, Weight(100), newV.Get(1))
	assert.Equal(t, Weight(70), newV.Get(7))
}

func TestValidators_Quorum(t *testing.T) {
	testCases := []struct {
		ids            []ValidatorID
		weights        []Weight
		expectedQuorum Weight
	}{
		{[]ValidatorID{1}, []Weight{3}, 3},
		{[]ValidatorID{1, 2}, []Weight{3, 3}, 5},
		{[]ValidatorID{1, 2, 3}, []Weight{3, 3, 3}, 7},
		{[]ValidatorID{1, 2, 3, 4}, []Weight{1, 2, 3, 4}, 7},              // 2/3 * 10 + 1 = 7.67 -> 7
		{[]ValidatorID{1, 2, 3, 4, 5}, []Weight{10, 20, 30, 40, 50}, 101}, // 2/3 * 150 + 1 = 101
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			v := ArrayToValidators(tc.ids, tc.weights)
			assert.Equal(t, tc.expectedQuorum, v.Quorum())
		})
	}
}

func TestEqualWeightValidators(t *testing.T) {
	ids := []ValidatorID{1, 2, 3, 4, 5}
	v := EqualWeightValidators(ids, 10)

	for _, id := range ids {
		assert.Equal(t, Weight(10), v.Get(id))
	}

	assert.Equal(t, Weight(50), v.TotalWeight())

	assert.Equal(t, ValidatorIndex(5), v.Len())

	v = EqualWeightValidators([]ValidatorID{}, 10)
	assert.Equal(t, ValidatorIndex(0), v.Len())
	assert.Equal(t, Weight(0), v.TotalWeight())
}

func TestValidators_RLPEncodeDecode(t *testing.T) {
	b := NewValidatorsBuilder()
	b.Set(1, 10)
	b.Set(3, 30)
	b.Set(5, 50)
	original := b.Build()

	var buf bytes.Buffer
	err := rlp.Encode(&buf, original)
	require.NoError(t, err)

	decoded := &Validators{}
	err = rlp.DecodeBytes(buf.Bytes(), decoded)
	require.NoError(t, err)

	// Verify that the decoded validator set is the same as the original
	assert.Equal(t, original.Len(), decoded.Len())
	assert.Equal(t, original.TotalWeight(), decoded.TotalWeight())

	assert.Equal(t, Weight(10), decoded.Get(1))
	assert.Equal(t, Weight(30), decoded.Get(3))
	assert.Equal(t, Weight(50), decoded.Get(5))

	assert.Equal(t, original.SortedIDs(), decoded.SortedIDs())
	assert.Equal(t, original.SortedWeights(), decoded.SortedWeights())
}

func TestValidators_DecodeRLPError(t *testing.T) {
	invalidRLP := []byte{0xc1} // Invalid RLP data

	decoded := &Validators{}
	err := rlp.DecodeBytes(invalidRLP, decoded)

	// Should return an error from the stream decoder
	assert.Error(t, err)
}

func TestValidators_String(t *testing.T) {
	b := NewValidatorsBuilder()
	b.Set(1, 10)
	b.Set(3, 30)
	b.Set(5, 50)
	v := b.Build()

	// Expected string format is "[id:weight],[id:weight],..."
	// IDs are sorted by weight (descending) then by ID (ascending)
	expected := "[5:50],[3:30],[1:10]"
	assert.Equal(t, expected, v.String())

	emptyV := NewValidatorsBuilder().Build()
	assert.Equal(t, "", emptyV.String())

	singleV := ArrayToValidators([]ValidatorID{42}, []Weight{100})
	assert.Equal(t, "[42:100]", singleV.String())
}
func TestValidators_StepwiseWeightOverflow(t *testing.T) {
	b := NewValidatorsBuilder()

	// Set first validator with weight just below MaxUint32
	b.Set(1, math.MaxUint32-10)

	// This should cause a panic during the cache calculation when adding weight
	defer func() {
		r := recover()
		assert.NotNil(t, r)
		assert.Contains(t, r.(string), "validators weight overflow")
	}()

	// Add another validator with weight that will cause overflow when added to the first
	b.Set(2, 20) // This will cause total to exceed MaxUint32

	b.Build() // Should panic during cache calculation
}

func TestValidators_WeightOverflow(t *testing.T) {
	b := NewValidatorsBuilder()
	defer func() {
		r := recover()
		assert.NotNil(t, r)
		assert.Contains(t, r.(string), "validators weight overflow")
	}()

	// Set weights to cause overflow
	b.Set(1, math.MaxUint32/2)
	b.Set(2, math.MaxUint32/2+1) // This should cause panic on Build()

	b.Build() // Should panic
}

func TestValidators_SortedOrderIsStable(t *testing.T) {
	b := NewValidatorsBuilder()

	b.Set(5, 10)
	b.Set(3, 10)
	b.Set(1, 10)
	b.Set(4, 10)
	b.Set(2, 10)

	v := b.Build()

	expectedIDs := []ValidatorID{1, 2, 3, 4, 5}
	assert.Equal(t, expectedIDs, v.SortedIDs())
}
