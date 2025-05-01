package parser

import (
	"context"
	"errors"
	"io"

	"github.com/opengs/file2llm/ocr"
)

// Parses `image/png` files
type PNGParser struct {
	ocrProvider ocr.Provider
}

func NewPNGParser(ocrProvider ocr.Provider) *PNGParser {
	return &PNGParser{
		ocrProvider: ocrProvider,
	}
}

func (p *PNGParser) SupportedMimeTypes() []string {
	return []string{"image/png"}
}

func (p *PNGParser) Parse(ctx context.Context, file io.Reader) Result {
	imageData, err := io.ReadAll(file)
	if err != nil {
		return &PNGParserResult{Err: errors.Join(errors.New("failed to read data to the bytes buffer"), err)}
	}

	text, err := p.ocrProvider.OCR(ctx, imageData)
	if err != nil {
		return &PNGParserResult{Err: errors.Join(errors.New("errors while running OCR"), err)}
	}

	return &PNGParserResult{Text: text}
}

type PNGParserResult struct {
	Text string `json:"text"`
	Err  error  `json:"error"`
}

func (r *PNGParserResult) String() string {
	return r.Text
}

func (r *PNGParserResult) Error() error {
	return r.Err
}

func (r *PNGParserResult) Componets() []Result {
	return nil
}
