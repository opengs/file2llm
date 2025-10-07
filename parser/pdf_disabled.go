//go:build !file2llm_feature_pdf && !test

package parser

import (
	"context"
	"io"
)

const FeaturePDFEnabled = false

// Parses `application/pdf` files
type PDFParser struct {
}

func NewPDFParser(innerParser Parser, dpi uint32) *PDFParser {
	return &PDFParser{}
}

func (p *PDFParser) SupportedMimeTypes() []string {
	return []string{}
}

func (p *PDFParser) Parse(ctx context.Context, file io.Reader, path string) Result {
	return &PDFParserResult{Err: ErrParserDisabled}
}

func (p *PDFParser) ParseStream(ctx context.Context, file io.Reader, path string) StreamResultIterator {
	return &PDFStreamResultIterator{
		path: path,
	}
}

type PDFStreamResultIterator struct {
	path      string
	completed bool
	startSend bool
	current   StreamResult
}

func (i *PDFStreamResultIterator) Current() StreamResult {
	return i.current
}

func (i *PDFStreamResultIterator) Next(ctx context.Context) bool {
	if i.completed {
		return false
	}

	if !i.startSend {
		i.startSend = true
		i.current = &PDFParserStreamResult{FullPath: i.path}
		return true
	}

	i.completed = true
	i.current = &PDFParserStreamResult{
		FullPath: i.path,
		Err:      ErrParserDisabled,
	}
	return true
}

func (i *PDFStreamResultIterator) Close() {
}
