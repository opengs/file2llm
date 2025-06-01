package basic

import (
	"testing"
)

func TestSplitStringInChunks(t *testing.T) {
	tests := []struct {
		name          string
		chunker       Chunker
		input         string
		wantCount     int
		reconstructed string
	}{
		{
			name: "Exact one chunk",
			chunker: Chunker{
				maxTokens: 2, // 2 * 4 = 8 bytes per chunk
				slide:     0,
			},
			input:         "12345678",
			wantCount:     1,
			reconstructed: "12345678",
		},
		{
			name: "Two chunks no overlap",
			chunker: Chunker{
				maxTokens: 2,
				slide:     0,
			},
			input:         "1234567890abcdef", // 16 bytes, 2 chunks
			wantCount:     2,
			reconstructed: "1234567890abcdef",
		},
		{
			name: "Three chunks with slide",
			chunker: Chunker{
				maxTokens: 2,
				slide:     1, // 4 bytes overlap
			},
			input:         "1234567890abcdef", // should overlap
			wantCount:     3,
			reconstructed: "12345678567890ab90abcdef",
		},
		{
			name: "Input shorter than chunk",
			chunker: Chunker{
				maxTokens: 5, // 20 bytes
				slide:     0,
			},
			input:         "short",
			wantCount:     1,
			reconstructed: "short",
		},
		{
			name: "Empty input",
			chunker: Chunker{
				maxTokens: 3,
				slide:     1,
			},
			input:         "",
			wantCount:     0,
			reconstructed: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotChunks := tt.chunker.splitStringInChunks(tt.input)

			if len(gotChunks) != tt.wantCount {
				t.Errorf("got %d chunks, want %d", len(gotChunks), tt.wantCount)
			}

			reconstructed := ""
			for _, ch := range gotChunks {
				reconstructed += ch
			}

			if tt.reconstructed != reconstructed {
				t.Errorf("reconstruction failed")
			}
		})
	}
}
