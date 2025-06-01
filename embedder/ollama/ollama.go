package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/opengs/file2llm/embedder/lib"
)

type Ollama struct {
	baseURL         string
	httpClient      *http.Client
	model           string
	checkNormalized sync.Once
	normalized      bool
	dimensions      uint32
}

func New(model string, config ...Config) *Ollama {
	ollama := &Ollama{
		baseURL:    "http://localhost:11434/api",
		httpClient: http.DefaultClient,
		model:      model,
		dimensions: 768,
	}

	for _, cfg := range config {
		cfg(ollama)
	}

	return ollama
}

func (o *Ollama) PullModel(ctx context.Context) error {
	reqBody, err := json.Marshal(map[string]any{
		"name":   o.model,
		"stream": false,
	})
	if err != nil {
		return errors.Join(errors.New("couldn't marshal request body"), err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/pull", bytes.NewBuffer(reqBody))
	if err != nil {
		return errors.Join(errors.New("couldn't create request"), err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return errors.Join(errors.New("couldn't send request"), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		text, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("bad status code from ollama server: code [%d], body [%s]", resp.StatusCode, string(text))
	}

	var responseData struct {
		Status string `json:"status"`
	}
	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Join(errors.New("failed to read response body from ollama server"), err)
	}
	if err := json.Unmarshal(responseBytes, &responseData); err != nil {
		return errors.Join(errors.New("failed to unmarshall response body"), err)
	}

	if responseData.Status != "success" {
		return fmt.Errorf("bad response status: %s", responseData.Status)
	}

	return nil
}

func (o *Ollama) GenerateEmbeddings(ctx context.Context, data string) ([]float32, error) {
	reqBody, err := json.Marshal(map[string]string{
		"model":  o.model,
		"prompt": data,
	})
	if err != nil {
		return nil, errors.Join(errors.New("couldn't marshal request body"), err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/embeddings", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, errors.Join(errors.New("couldn't create request"), err)
	}
	req.Header.Set("Content-Type", "application/json")

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
		return nil, errors.Join(errors.New("couldn't read response body"), err)
	}
	var ollamaResponse struct {
		Embedding []float32 `json:"embedding"`
	}
	err = json.Unmarshal(body, &ollamaResponse)
	if err != nil {
		return nil, errors.Join(errors.New("couldn't unmarshal response body"), err)
	}

	if len(ollamaResponse.Embedding) == 0 {
		return nil, errors.New("no embeddings found in the response")
	}

	if len(ollamaResponse.Embedding) != int(o.dimensions) {
		return nil, fmt.Errorf("ollama returned embeddings vector of wrong size: wanted: %d, returned: %d", o.dimensions, len(ollamaResponse.Embedding))
	}

	v := ollamaResponse.Embedding
	o.checkNormalized.Do(func() {
		o.normalized = lib.IsNormalized(v)
	})
	if !o.normalized {
		lib.NormalizeVectorInPlace(v)
	}

	return v, nil
}

func (o *Ollama) Dimensions() uint32 {
	return o.dimensions
}

func (o *Ollama) ModelName() string {
	return o.model
}
