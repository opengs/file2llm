package lib

import (
	"testing"
)

func TestDotProduct(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float32
		wantErr  bool
	}{
		{
			name:     "Equal length vectors",
			a:        []float32{1, 2, 3},
			b:        []float32{4, 5, 6},
			expected: 32, // 1*4 + 2*5 + 3*6 = 32
			wantErr:  false,
		},
		{
			name:     "Empty vectors",
			a:        []float32{},
			b:        []float32{},
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "Mismatched vector lengths",
			a:        []float32{1, 2},
			b:        []float32{3},
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "Vectors with negative numbers",
			a:        []float32{-1, 2, -3},
			b:        []float32{4, -5, 6},
			expected: -32, // -1*4 + 2*(-5) + -3*6 = -4 -10 -18 = -32
			wantErr:  false,
		},
		{
			name:     "Vectors with zeros",
			a:        []float32{0, 0, 0},
			b:        []float32{1, 2, 3},
			expected: 0,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DotProduct(tt.a, tt.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("DotProduct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("DotProduct() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
