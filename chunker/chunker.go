package chunker

import (
	"context"

	"github.com/opengs/file2llm/parser"
)

type Chunk struct {
	Start *StartChunk
	Data  *DataChunk
	End   *EndChunk

	// Internal chunker error.
	Error error
}

type StartChunk struct {
	FilePath string
}

type DataChunk struct {
	FilePath string
	Data     string
}

type EndChunk struct {
	FilePath string
	Error    error
}

type ChunkIterator interface {
	Next(ctx context.Context) bool
	Current() Chunk
}

type Chunker interface {
	GenerateChunks(ctx context.Context, parseStream parser.StreamResultIterator) ChunkIterator
}
