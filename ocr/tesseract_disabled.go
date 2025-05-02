//go:build !file2llm_feature_ocr_tesseract && !test

package ocr

import (
	"context"
	"errors"
)

var errTesseractProviderNotCompiled = errors.New("OCR is not possible because binary wasnt compiled with internal tesseract OCR provider")

const FeatureTesseractEnabled = false

type Tesseract struct {
}

func NewTesseractProvider(config *TesseractConfig) *Tesseract {
	return &Tesseract{}
}

func (p *Tesseract) OCR(ctx context.Context, image []byte) (string, error) {
	return "", errTesseractProviderNotCompiled
}

func (p *Tesseract) Init() error {
	return errTesseractProviderNotCompiled
}
func (p *Tesseract) Destroy() error {
	return nil
}

func (p *Tesseract) IsMimeTypeSupported(mimeType string) bool {
	return false
}
