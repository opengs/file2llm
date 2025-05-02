//go:build !file2llm_feature_pdf && !test

package parser

import (
	"context"
	"io"
)

// Parses `application/pdf` files
type PDFParser struct {
}

func NewPDFParser(innerParser Parser) *PDFParser {
	return &PDFParser{}
}

func (p *PDFParser) SupportedMimeTypes() []string {
	return []string{}
}

func (p *PDFParser) Parse(ctx context.Context, file io.Reader) Result {
	return &PDFParserResult{Err: ErrParserDisabled}
}

type PDFParserResultPage struct {
	Text   string   `json:"text"`
	Images []Result `json:"images"`
}

type PDFParserResult struct {
	Metadata string                `json:"metadata"`
	Pages    []PDFParserResultPage `json:"pages"`
	Err      error                 `json:"error"`
}

func (r *PDFParserResult) String() string {
	return ""
}

func (r *PDFParserResult) Error() error {
	return ErrParserDisabled
}

func (r *PDFParserResult) Componets() []Result {
	return nil
}
