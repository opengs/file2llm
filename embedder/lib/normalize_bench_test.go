package lib

import (
	"math/rand"
	"testing"
)

func generateRandomVector(n int) []float32 {
	vec := make([]float32, n)
	for i := range vec {
		vec[i] = rand.Float32()
	}
	return vec
}

func BenchmarkIsNormalizedAVX(b *testing.B) {
	vec := generateRandomVector(1024)
	NormalizeVectorInPlace(vec) // normalize beforehand

	useAVX2 = true
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = isNormalizedAVX(vec)
	}
}

func BenchmarkIsNormalizedGO(b *testing.B) {
	vec := generateRandomVector(1024)
	NormalizeVectorInPlace(vec) // normalize beforehand

	useAVX2 = false
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = isNormalizedGO(vec)
	}
}

func BenchmarkNormalizeVectorInPlaceAVX(b *testing.B) {
	useAVX2 = true
	vec := generateRandomVector(1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copyVec := make([]float32, len(vec))
		copy(copyVec, vec)
		normalizeVectorInPlaceAVX(copyVec)
	}
}

func BenchmarkNormalizeVectorInPlaceGO(b *testing.B) {
	useAVX2 = false
	vec := generateRandomVector(1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copyVec := make([]float32, len(vec))
		copy(copyVec, vec)
		normalizeVectorInPlaceGO(copyVec)
	}
}
