package election

import "github.com/kelindar/simd"

// addInt32Vecs is to be used only by consensus election:
// src1[i] in [-ValidatorXWeight, ValidatorXWeight] and src2[i] in [-ValidatorYWeight, ValidatorYWeight]
// => |src1[i] + src2[i]| <= ValidatorXWeight + ValidatorYWeight <= TotalValidatorWeight <= max(int32)
func addInt32Vecs(dst []int32, src1 []int32, src2 []int32) {
	if len(src1) == 0 {
		return
	}
	simd.AddInt32s(dst, src1, src2)
}

// mulInt32VecWithConst is to be used only by consensus election:
// src[i] in [-1, 1] and num in [0, ValidatorXWeight]
// => |src[i] * num| <= ValidatorXWeight <= TotalValidatorWeight <= max(int32)
func mulInt32VecWithConst(dst []int32, src []int32, num int32) {
	for i := range len(src) {
		dst[i] = src[i] * num
	}
}

// normalize scales the values to the [-1, 1] range
func normalizeInt32Vec(dst []int32, src []int32) {
	for i := range len(src) {
		if src[i] >= 0 {
			dst[i] = 1
		} else {
			dst[i] = -1
		}
	}
}

func initInt32WithConst(num int32, length int) []int32 {
	vec := make([]int32, length)
	for i := range len(vec) {
		vec[i] = num
	}
	return vec
}

func boolMaskInt32Vec(src []int32, predicate func(x int32) bool) []bool {
	vec := make([]bool, len(src))
	for i := range len(src) {
		vec[i] = predicate(src[i])
	}
	return vec
}
