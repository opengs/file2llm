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

func (p *PNGParser) Parse(ctx context.Context, file io.Reader, path string) Result {
	text, err := p.ocrProvider.OCR(ctx, file)
	if err != nil {
		return &ImageParserResult{Err: errors.Join(errors.New("errors while running OCR"), err), FullPath: path}
	}

	return &ImageParserResult{Text: text, FullPath: path}
}

func (p *PNGParser) ParseStream(ctx context.Context, file io.Reader, path string) StreamResultIterator {
	return &ImageStreamResultIterator{
		path:        path,
		file:        file,
		ocrProvider: p.ocrProvider,
		baseContext: ctx,
	}
}
