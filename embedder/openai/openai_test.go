package openai

import (
	"os"
	"testing"

	"github.com/opengs/file2llm/embedder/testlib"
)

func TestOllama(t *testing.T) {
	openaiAPIKey := os.Getenv("TEST_EMBEDDER_OPENAI_APIKEY")
	if openaiAPIKey == "" {
		t.Skip("TEST_EMBEDDER_OPENAI_APIKEY is not configured")
	}

	emb := New("text-embedding-3-small", openaiAPIKey)
	testlib.TestEmbedder(t, emb)
}
