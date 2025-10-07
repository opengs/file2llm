package slidechunk

import (
	"context"
	"strings"

	"github.com/opengs/file2llm/chunker"
	"github.com/opengs/file2llm/parser"
)

type SlideChunker struct {
	maxTokens uint32
	slide     uint32
}

// Creates new chunker with maximum tokens in one chunk `window` and `slide` overlap between chunks.
func New(window uint32, slide uint32) *SlideChunker {
	return &SlideChunker{
		maxTokens: window,
		slide:     slide,
	}
}

func (c *SlideChunker) GenerateChunks(ctx context.Context, streamIterator parser.StreamResultIterator) chunker.ChunkIterator {
	return &slideChunkIterator{
		ctx:            ctx,
		streamIterator: streamIterator,
		maxTokens:      c.maxTokens,
		slide:          c.slide,
		data:           make(map[string]*strings.Builder),
	}
}

type slideChunkIterator struct {
	ctx            context.Context
	streamIterator parser.StreamResultIterator
	maxTokens      uint32
	slide          uint32

	ready []chunker.Chunk
	data  map[string]*strings.Builder
}

func (c *slideChunkIterator) splitStringInChunks(data string) []string {
	var chunks []string
	for i := 0; i < len(data); i += (int(c.maxTokens)*4 - int(c.slide*4)) {
		end := i + int(c.maxTokens*4)
		if end > len(data) {
			end = len(data) // ensure no out-of-bounds
		}
		chunk := data[i:end]
		chunks = append(chunks, chunk)

		if end == len(data) {
			break // stop when end reaches the end of string
		}
	}

	return chunks
}

func (i *slideChunkIterator) processChunks(filePath string, stage parser.ParseProgressStage) {
	if len(i.data) == 0 {
		return
	}

	entireText := i.data[filePath].String()
	chunks := i.splitStringInChunks(entireText)
	if len(chunks) == 0 {
		return
	}
	if len(chunks) == 1 {
		if stage == parser.ProgressCompleted {
			i.ready = append(i.ready, chunker.Chunk{
				Data: &chunker.DataChunk{
					FilePath: filePath,
					Data:     chunks[0],
				},
			})
			i.data[filePath].Reset()
		}
	} else {
		for _, chunk := range chunks[:len(chunks)-1] {
			i.ready = append(i.ready, chunker.Chunk{
				Data: &chunker.DataChunk{
					FilePath: filePath,
					Data:     chunk,
				},
			})
		}
		i.data[filePath].Reset()
		i.data[filePath].WriteString(chunks[len(chunks)-1])
	}
}

func (i *slideChunkIterator) Next(ctx context.Context) bool {
	if len(i.ready) > 0 {
		i.ready = i.ready[1:]
		if len(i.ready) > 0 {
			return true
		}
	}

	for i.streamIterator.Next(ctx) {
		streamResult := i.streamIterator.Current()
		for streamResult.SubResult() != nil {
			streamResult = streamResult.SubResult()
		}

		if streamResult.Stage() == parser.ProgressNew {
			newBuilder := &strings.Builder{}
			newBuilder.WriteString(streamResult.String())
			i.data[streamResult.Path()] = newBuilder
			i.ready = append(i.ready, chunker.Chunk{
				Start: &chunker.StartChunk{
					FilePath: streamResult.Path(),
				},
			})
			i.processChunks(streamResult.Path(), parser.ProgressNew)
		}

		if streamResult.Stage() == parser.ProgressUpdate {
			i.data[streamResult.Path()].WriteString(streamResult.String())
			i.processChunks(streamResult.Path(), parser.ProgressUpdate)
		}

		if streamResult.Stage() == parser.ProgressCompleted {
			i.data[streamResult.Path()].WriteString(streamResult.String())
			i.processChunks(streamResult.Path(), parser.ProgressCompleted)
			delete(i.data, streamResult.Path())
			i.ready = append(i.ready, chunker.Chunk{
				End: &chunker.EndChunk{
					FilePath: streamResult.Path(),
					Error:    streamResult.Error(),
				},
			})
		}

		if len(i.ready) > 0 {
			return true
		}
	}

	return false
}

func (i *slideChunkIterator) Current() chunker.Chunk {
	return i.ready[0]
}
