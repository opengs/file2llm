package openai

import "net/http"

type Config func(o *OpenAI)

func WithBaseURL(baseURL string) Config {
	return func(o *OpenAI) {
		o.baseURL = baseURL
	}
}

func WithHTTPClient(httpClient *http.Client) Config {
	return func(o *OpenAI) {
		o.httpClient = httpClient
	}
}

func WithDimensions(dimensions uint32) Config {
	return func(o *OpenAI) {
		o.dimensions = dimensions
	}
}
