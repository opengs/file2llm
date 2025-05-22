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

func (p *PDFParser) ParseStream(ctx context.Context, file io.Reader, path string) chan StreamResult {
	resultChan := make(chan StreamResult)
	go func() {
		defer close(resultChan)
		resultChan <- &PDFParserStreamResult{FullPath: path, CurrentStage: ProgressNew}
		resultChan <- &PDFParserStreamResult{Err: ErrParserDisabled, FullPath: path, CurrentStage: ProgressCompleted}
	}()
	return resultChan
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

func (r *PDFParserResult) Path() string {
	return ""
}

func (r *PDFParserResult) String() string {
	return ""
}

func (r *PDFParserResult) Error() error {
	return ErrParserDisabled
}

func (r *PDFParserResult) Subfiles() []Result {
	return nil
}

type PDFParserStreamResultPage struct {
	Text   string         `json:"text"`
	Images []StreamResult `json:"images"`
}

type PDFParserStreamResult struct {
	FullPath        string                      `json:"path"`
	Text            string                      `json:"text"`
	CurrentStage    ParseProgressStage          `json:"stage"`
	CurrentProgress uint8                       `json:"progress"`
	Metadata        string                      `json:"metadata"`
	Pages           []PDFParserStreamResultPage `json:"pages"`
	Subfile         StreamResult                `json:"subfile"`
	Err             error                       `json:"error"`
}

func (r *PDFParserStreamResult) Path() string {
	return ""
}

func (r *PDFParserStreamResult) Stage() ParseProgressStage {
	return r.CurrentStage
}

func (r *PDFParserStreamResult) Progress() uint8 {
	return r.CurrentProgress
}

func (r *PDFParserStreamResult) SubResult() StreamResult {
	return r.Subfile
}

func (r *PDFParserStreamResult) String() string {
	return ""
}

func (r *PDFParserStreamResult) Error() error {
	return ErrParserDisabled
}
