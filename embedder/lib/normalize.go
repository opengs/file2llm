package lib

import "math"

const isNormalizedPrecisionTolerance = 1e-6

func isNormalizedAVX(v []float32) bool {
	magnitude := float64(norm_AVX2_F32(v))
	return math.Abs(magnitude-1) < isNormalizedPrecisionTolerance
}

func isNormalizedGO(v []float32) bool {
	var sqSum float64
	for _, val := range v {
		sqSum += float64(val) * float64(val)
	}
	magnitude := math.Sqrt(sqSum)
	return math.Abs(magnitude-1) < isNormalizedPrecisionTolerance
}

func IsNormalized(v []float32) bool {
	if useAVX2 {
		return isNormalizedAVX(v)
	} else {
		return isNormalizedGO(v)
	}
}

func normalizeVectorInPlaceAVX(v []float32) {
	norm := norm_AVX2_F32(v)
	divNumber_AVX2_F32(v, norm)
}

func normalizeVectorInPlaceGO(v []float32) {
	var norm float32
	for _, val := range v {
		norm += val * val
	}
	norm = float32(math.Sqrt(float64(norm)))

	for i, val := range v {
		v[i] = val / norm
	}
}

func NormalizeVectorInPlace(v []float32) {
	if useAVX2 {
		normalizeVectorInPlaceAVX(v)
	} else {
		normalizeVectorInPlaceGO(v)
	}
}
