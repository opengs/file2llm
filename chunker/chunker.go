package chunker

import (
	"context"

	"github.com/opengs/file2llm/parser"
)

type Chunk struct {
	Start *StartChunk
	Data  *DataChunk
	End   *EndChunk
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

type Chunker interface {
	GenerateChunks(ctx context.Context, parseResult chan parser.StreamResult) chan Chunk
}
