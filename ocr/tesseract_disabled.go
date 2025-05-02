//go:build !file2llm_feature_ocr_tesseract && !test

package ocr

import (
	"context"
	"errors"
)

var errTesseractProviderNotCompiled = errors.New("OCR is not possible because binary wasnt compiled with internal tesseract OCR provider")

const FeatureTesseractEnabled = false

type tesseractProvider struct {
}

func NewTesseractProvider(config *TesseractConfig) Provider {
	return &tesseractProvider{}
}

func (p *tesseractProvider) OCR(ctx context.Context, image []byte) (string, error) {
	return "", errTesseractProviderNotCompiled
}

func (p *tesseractProvider) Init() error {
	return errTesseractProviderNotCompiled
}
func (p *tesseractProvider) Destroy() error {
	return nil
}

func (p *tesseractProvider) Name() ProviderName {
	return ProviderNameTesseract
}

func (p *tesseractProvider) IsMimeTypeSupported(mimeType string) bool {
	return false
}
