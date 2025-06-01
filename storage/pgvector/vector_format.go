package pgvector

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func embeddingToPgvectorFormat(vec []float32) string {
	var b strings.Builder
	b.WriteByte('[')

	for i, val := range vec {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%g", val) // %g trims unnecessary zeroes
	}

	b.WriteByte(']')
	return b.String()
}

func pgvectorFormatToEmbedding(s string) ([]float32, error) {
	s = strings.TrimSpace(s)
	if len(s) < 2 || s[0] != '[' || s[len(s)-1] != ']' {
		return nil, errors.New("invalid pgvector format")
	}

	s = s[1 : len(s)-1] // Strip brackets
	if len(s) == 0 {
		return []float32{}, nil
	}

	parts := strings.Split(s, ",")
	vec := make([]float32, len(parts))
	for i, part := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(part), 32)
		if err != nil {
			return nil, err
		}
		vec[i] = float32(f)
	}
	return vec, nil
}
