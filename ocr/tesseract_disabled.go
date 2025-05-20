//go:build !file2llm_feature_ocr_tesseract && !test

package ocr

import (
	"context"
	"errors"
	"io"
)

var errTesseractProviderNotCompiled = errors.New("OCR is not possible because binary wasnt compiled with internal tesseract OCR provider")

const FeatureTesseractEnabled = false

type Tesseract struct {
}

func NewTesseract(config TesseractConfig) *Tesseract {
	return &Tesseract{}
}

func (p *Tesseract) OCR(ctx context.Context, image io.Reader) (string, error) {
	return "", errTesseractProviderNotCompiled
}

func (p *Tesseract) OCRWithProgress(ctx context.Context, image io.Reader) OCRProgress {
	return &tesseractOCRProgress{}
}

func (p *Tesseract) Init() error {
	return errTesseractProviderNotCompiled
}
func (p *Tesseract) Destroy() error {
	return errTesseractProviderNotCompiled
}

func (p *Tesseract) IsMimeTypeSupported(mimeType string) bool {
	return false
}

type tesseractOCRProgress struct {
}

func (t *tesseractOCRProgress) Completion() uint8 {
	return 0
}

func (t *tesseractOCRProgress) CompletionUpdates() chan uint8 {
	ch := make(chan uint8)
	close(ch)
	return ch
}

func (t *tesseractOCRProgress) Text() (string, error) {
	return "", errTesseractProviderNotCompiled
}
