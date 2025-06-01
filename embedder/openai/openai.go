package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"

	"github.com/opengs/file2llm/embedder/lib"
)

type OpenAI struct {
	baseURL         string
	apiKey          string
	httpClient      *http.Client
	model           string
	checkNormalized sync.Once
	normalized      bool
	dimensions      uint32
}

func New(model string, apiKey string, config ...Config) *OpenAI {
	ollama := &OpenAI{
		baseURL:    "https://api.openai.com/v1",
		httpClient: http.DefaultClient,
		model:      model,
		apiKey:     apiKey,
		dimensions: 1536,
	}

	for _, cfg := range config {
		cfg(ollama)
	}

	return ollama
}

func (o *OpenAI) GenerateEmbeddings(ctx context.Context, data string) ([]float32, error) {
	bodyData := map[string]any{
		"input":      data,
		"model":      o.model,
		"dimensions": o.dimensions,
	}

	reqBody, err := json.Marshal(bodyData)
	if err != nil {
		return nil, errors.Join(errors.New("couldn't marshal request body"), err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/embeddings", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, errors.Join(errors.New("couldn't create request"), err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, errors.Join(errors.New("couldn't send request"), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("error response from the embedding API: " + resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Join(errors.New("couldn't read response bod"), err)
	}
	var openAIResponse struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	err = json.Unmarshal(body, &openAIResponse)
	if err != nil {
		return nil, errors.Join(errors.New("couldn't unmarshal response body"), err)
	}

	if len(openAIResponse.Data) == 0 || len(openAIResponse.Data[0].Embedding) == 0 {
		return nil, errors.New("no embeddings found in the response")
	}

	v := openAIResponse.Data[0].Embedding
	o.checkNormalized.Do(func() {
		o.normalized = lib.IsNormalized(v)
	})
	if !o.normalized {
		lib.NormalizeVectorInPlace(v)
	}

	return v, nil
}

func (o *OpenAI) Dimensions() uint32 {
	return o.dimensions
}

func (o *OpenAI) ModelName() string {
	return o.model
}
