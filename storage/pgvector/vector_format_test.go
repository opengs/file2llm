package pgvector

import (
	"reflect"
	"testing"
)

func TestEmbeddingToPgvectorFormat(t *testing.T) {
	tests := []struct {
		input    []float32
		expected string
	}{
		{[]float32{1.0, 2.5, 3.14}, "[1, 2.5, 3.14]"},
		{[]float32{}, "[]"},
		{[]float32{0.0}, "[0]"},
		{[]float32{123456.000}, "[123456]"}, // %g rounds large float
	}

	for _, test := range tests {
		output := embeddingToPgvectorFormat(test.input)
		if output != test.expected {
			t.Errorf("embeddingToPgvectorFormat(%v) = %q; expected %q", test.input, output, test.expected)
		}
	}
}

func TestPgvectorFormatToEmbedding(t *testing.T) {
	tests := []struct {
		input       string
		expected    []float32
		expectError bool
	}{
		{"[1, 2.5, 3.14]", []float32{1, 2.5, 3.14}, false},
		{"[]", []float32{}, false},
		{"[0]", []float32{0}, false},
		{"[  1 ,  2  , 3  ]", []float32{1, 2, 3}, false},
		{"{1, 2.5, 3.14}", nil, true}, // wrong brackets
		{"[1, abc]", nil, true},       // invalid float
		{"", nil, true},               // empty string
		{"[1,]", nil, true},           // trailing comma
		{"[1 2]", nil, true},          // missing comma
	}

	for _, test := range tests {
		output, err := pgvectorFormatToEmbedding(test.input)
		if (err != nil) != test.expectError {
			t.Errorf("pgvectorFormatToEmbedding(%q) error = %v; expected error = %v", test.input, err, test.expectError)
		}
		if !test.expectError && !reflect.DeepEqual(output, test.expected) {
			t.Errorf("pgvectorFormatToEmbedding(%q) = %v; expected %v", test.input, output, test.expected)
		}
	}
}
