package testlib

import (
	"math/rand/v2"
	"testing"

	"github.com/opengs/file2llm/embedder"
	"github.com/opengs/file2llm/embedder/lib"
)

func RandNormalizedEmbedding(dimensions int) []float32 {
	vec := make([]float32, dimensions)
	for i := range vec {
		vec[i] = rand.Float32() // random float32 between 0.0 and 1.0
	}

	lib.NormalizeVectorInPlace(vec)

	return vec
}

func TestEmbedder(t *testing.T, emb embedder.Embedder) {
	emb1, err := emb.GenerateEmbeddings(t.Context(), "Hello, world 1!")
	if err != nil {
		t.Error(err.Error())
		return
	}

	emb2, err := emb.GenerateEmbeddings(t.Context(), "Hello, world 2!")
	if err != nil {
		t.Error(err.Error())
		return
	}

	emb3, err := emb.GenerateEmbeddings(t.Context(), "information technology")
	if err != nil {
		t.Error(err.Error())
		return
	}

	similarity, err := lib.DotProduct(emb1, emb2)
	if err != nil {
		t.Error(err.Error())
		return
	}
	if similarity < 0.5 {
		t.Error("Low similarity")
		return
	}

	similarity, err = lib.DotProduct(emb1, emb3)
	if err != nil {
		t.Error(err.Error())
		return
	}
	if similarity > 0.5 {
		t.Error("Hight similarity")
		return
	}
}
