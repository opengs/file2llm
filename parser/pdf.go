package parser

import (
	"strings"
)

type PDFParserResult struct {
	FullPath string   `json:"path"`
	Metadata string   `json:"metadata"`
	Pages    []string `json:"pages"`
	Err      error    `json:"error"`
}

func (r *PDFParserResult) Path() string {
	return r.FullPath
}

func (r *PDFParserResult) String() string {
	var result strings.Builder

	if r.Metadata != "" {
		result.WriteString("------ Metadata ------\n\n")
		result.WriteString(r.Metadata)
		result.WriteString("\n\n")
	}

	result.WriteString("------ Pages ------\n\n")

	for _, page := range r.Pages {
		result.WriteString(page)
		result.WriteString("\n")
	}

	return result.String()
}

func (r *PDFParserResult) Error() error {
	return r.Err
}

func (r *PDFParserResult) Subfiles() []Result {
	return nil
}

type PDFParserStreamResult struct {
	FullPath        string             `json:"path"`
	CurrentStage    ParseProgressStage `json:"stage"`
	CurrentProgress uint8              `json:"progress"`
	Text            string             `json:"text"`
	Err             error              `json:"error"`
}

func (r *PDFParserStreamResult) Path() string {
	return r.FullPath
}

func (r *PDFParserStreamResult) Stage() ParseProgressStage {
	return r.CurrentStage
}

func (r *PDFParserStreamResult) Progress() uint8 {
	return r.CurrentProgress
}

func (r *PDFParserStreamResult) SubResult() StreamResult {
	return nil
}

func (r *PDFParserStreamResult) String() string {
	return r.Text
}

func (r *PDFParserStreamResult) Error() error {
	return r.Err
}
