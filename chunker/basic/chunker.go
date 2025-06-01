package basic

import (
	"context"

	"github.com/opengs/file2llm/chunker"
	"github.com/opengs/file2llm/parser"
)

type Chunker struct {
	maxTokens uint32
	slide     uint32
}

func (c *Chunker) splitStringInChunks(data string) []string {
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

func (c *Chunker) GenerateChunks(ctx context.Context, parseResult chan parser.StreamResult) chan chunker.Chunk {
	chunksChan := make(chan chunker.Chunk)

	go func() {
		defer close(chunksChan)

		for result := range parseResult {
			deepest := result
			for deepest.SubResult() != nil {
				deepest = deepest.SubResult()
			}

			if deepest.Stage() == parser.ProgressNew {
				select {
				case chunksChan <- chunker.Chunk{
					Start: &chunker.StartChunk{
						FilePath: deepest.Path(),
					},
				}:
				case <-ctx.Done():
					return
				}

			}

			if deepest.Stage() == parser.ProgressCompleted {
				if deepest.Error() == nil {
					dataChunks := c.splitStringInChunks(deepest.String())
					for _, dataChunk := range dataChunks {
						select {
						case chunksChan <- chunker.Chunk{
							Data: &chunker.DataChunk{
								FilePath: deepest.Path(),
								Data:     dataChunk,
							},
						}:
						case <-ctx.Done():
							return
						}
					}
				}

				select {
				case chunksChan <- chunker.Chunk{

					End: &chunker.EndChunk{
						FilePath: deepest.Path(),
						Error:    deepest.Error(),
					},
				}:
				case <-ctx.Done():
					return
				}

			}
		}
	}()

	return chunksChan
}
