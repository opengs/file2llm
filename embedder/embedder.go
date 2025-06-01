package embedder

import "context"

type Embedder interface {
	// Returned embeddings vector dimensions
	Dimensions() uint32
	// Unique model name used for generating embeddings
	ModelName() string
	// Generate embeddings from string. Returns normalized vector
	GenerateEmbeddings(ctx context.Context, data string) ([]float32, error)
}
