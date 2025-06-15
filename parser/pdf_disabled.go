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

func NewPDFParser(innerParser Parser) *PDFParser {
	return &PDFParser{}
}

func (p *PDFParser) SupportedMimeTypes() []string {
	return []string{}
}

func (p *PDFParser) Parse(ctx context.Context, file io.Reader, path string) Result {
	return &PDFParserResult{Err: ErrParserDisabled}
}

func (p *PDFParser) ParseStream(ctx context.Context, file io.Reader, path string) StreamResultIterator {
	resultChan := make(chan StreamResult)
	go func() {
		defer close(resultChan)
		resultChan <- &PDFParserStreamResult{FullPath: path, CurrentStage: ProgressNew}
		resultChan <- &PDFParserStreamResult{Err: ErrParserDisabled, FullPath: path, CurrentStage: ProgressCompleted}
	}()
	return resultChan
}
