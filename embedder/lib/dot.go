package lib

import "errors"

func DotProduct(a, b []float32) (float32, error) {
	// The vectors must have the same length
	if len(a) != len(b) {
		return 0, errors.New("vectors must have the same length")
	}

	var dotProduct float32
	for i := range a {
		dotProduct += a[i] * b[i]
	}

	return dotProduct, nil
}
