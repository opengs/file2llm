package ollama

import (
	"os"
	"testing"

	"github.com/opengs/file2llm/embedder/testlib"
)

func TestOllama(t *testing.T) {
	ollamaBaseURL := os.Getenv("TEST_EMBEDDER_OLLAMA_BASEURL")
	if ollamaBaseURL == "" {
		t.Skip("TEST_EMBEDDER_OLLAMA_BASEURL is not configured")
	}

	emb := New("all-minilm", WithBaseURL(ollamaBaseURL), WithDimensions(384))
	if err := emb.PullModel(t.Context()); err != nil {
		t.Error(err.Error())
		return
	}
	testlib.TestEmbedder(t, emb)
}
