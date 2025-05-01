package file2llm

import "github.com/opengs/file2llm/ocr"

type Config struct {
	OCRProvider ocr.Provider
}

type file2LLM struct{}

func New(config *Config) (*file2LLM, error) {
	return &file2LLM{}, nil
}

func (f2l *file2LLM) Destroy() error {
	return nil
}
