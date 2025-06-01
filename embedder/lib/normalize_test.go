package lib

import (
	"math"
	"testing"
)

func floatsEqual(a, b float32, tol float64) bool {
	return math.Abs(float64(a)-float64(b)) < tol
}

func TestIsNormalizedGO(t *testing.T) {
	vec := []float32{0.6, 0.8}
	if !isNormalizedGO(vec) {
		t.Errorf("Expected vector to be normalized (GO)")
	}
}

func TestIsNormalizedAVX(t *testing.T) {
	vec := []float32{0.6, 0.8}
	if !isNormalizedAVX(vec) {
		t.Errorf("Expected vector to be normalized (AVX)")
	}
}

func TestNormalizeVectorInPlaceGO(t *testing.T) {
	vec := []float32{3, 4}
	normalizeVectorInPlaceGO(vec)

	expected := []float32{0.6, 0.8}
	for i := range vec {
		if !floatsEqual(vec[i], expected[i], 1e-6) {
			t.Errorf("Expected %v, got %v", expected[i], vec[i])
		}
	}
}

func TestNormalizeVectorInPlaceAVX(t *testing.T) {
	vec := []float32{3, 4}
	normalizeVectorInPlaceAVX(vec)

	expected := []float32{0.6, 0.8}
	for i := range vec {
		if !floatsEqual(vec[i], expected[i], 1e-6) {
			t.Errorf("Expected %v, got %v", expected[i], vec[i])
		}
	}
}

func TestNormalizeVectorInPlaceSwitch(t *testing.T) {
	input := []float32{5, 12}
	expected := []float32{0.3846154, 0.9230769} // normalized form

	vec := make([]float32, len(input))
	copy(vec, input)
	useAVX2 = false
	NormalizeVectorInPlace(vec)
	for i := range vec {
		if !floatsEqual(vec[i], expected[i], 1e-6) {
			t.Errorf("[GO] Expected %v, got %v", expected[i], vec[i])
		}
	}

	copy(vec, input)
	useAVX2 = true
	NormalizeVectorInPlace(vec)
	for i := range vec {
		if !floatsEqual(vec[i], expected[i], 1e-6) {
			t.Errorf("[AVX] Expected %v, got %v", expected[i], vec[i])
		}
	}
}

func TestIsNormalizedSwitch(t *testing.T) {
	vec := []float32{1.0 / float32(math.Sqrt2), 1.0 / float32(math.Sqrt2)}
	useAVX2 = true
	if !IsNormalized(vec) {
		t.Errorf("Expected IsNormalized to return true (AVX)")
	}
	useAVX2 = false
	if !IsNormalized(vec) {
		t.Errorf("Expected IsNormalized to return true (GO)")
	}
}
