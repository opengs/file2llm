package ollama

import "net/http"

type Config func(o *Ollama)

func WithBaseURL(baseURL string) Config {
	return func(o *Ollama) {
		o.baseURL = baseURL
	}
}

func WithHTTPClient(httpClient *http.Client) Config {
	return func(o *Ollama) {
		o.httpClient = httpClient
	}
}

func WithDimensions(dimensions uint32) Config {
	return func(o *Ollama) {
		o.dimensions = dimensions
	}
}
