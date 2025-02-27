package election

import (
	"math"
	"slices"
	"testing"
)

func TestSum_Limit(t *testing.T) {
	testSum(
		t,
		[]int32{math.MaxInt32 / 2, -math.MaxInt32 / 2, math.MaxInt32},
		[]int32{math.MaxInt32/2 + 1, -math.MaxInt32/2 - 1, -math.MaxInt32},
		[]int32{math.MaxInt32, -math.MaxInt32, 0},
	)
}

func TestSum_Empty(t *testing.T) {
	testSum(t, []int32{}, []int32{}, []int32{})
}

func testSum(t *testing.T, a, b, expected []int32) {
	res := make([]int32, len(a))
	addInt32Vecs(res, a, b)
	if !slices.Equal(res, expected) {
		t.Errorf("incorrect sum for vectors %v and %v, expected: %v, got: %v", a, b, expected, res)
	}
}

func TestMul(t *testing.T) {
	a := []int32{-1, 1, -1}
	num := int32(math.MaxInt32)
	res := make([]int32, len(a))
	expected := []int32{-math.MaxInt32, math.MaxInt32, -math.MaxInt32}
	mulInt32VecWithConst(res, a, num)
	if !slices.Equal(res, expected) {
		t.Errorf("incorrect mul for vector %v and const %d, expected: %v, got: %v", a, num, expected, res)
	}
}

func TestBoolMask(t *testing.T) {
	vec := []int32{math.MaxInt32/2 - 1, -math.MaxInt32/2 + 1, math.MaxInt32 / 2, -math.MaxInt32 / 2, 0, -math.MaxInt32, math.MaxInt32}
	Q := int32(math.MaxInt32 / 2)
	posRes := boolMaskInt32Vec(vec, func(x int32) bool { return x >= Q })
	posExpected := []bool{false, false, true, false, false, false, true}
	if !slices.Equal(posRes, posExpected) {
		t.Errorf("incorrect bool mask for vector %v and const %d, expected: %v, got: %v", vec, Q, posExpected, posRes)
	}
	negRes := boolMaskInt32Vec(vec, func(x int32) bool { return x <= -Q })
	negExpected := []bool{false, false, false, true, false, true, false}
	if !slices.Equal(negRes, negExpected) {
		t.Errorf("incorrect bool mask for vector %v and const %d, expected: %v, got: %v", vec, Q, negExpected, negRes)
	}
}
