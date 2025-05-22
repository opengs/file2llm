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
		return &PNGParserResult{Err: errors.Join(errors.New("errors while running OCR"), err), FullPath: path}
	}

	return &PNGParserResult{Text: text, FullPath: path}
}

func (p *PNGParser) ParseStream(ctx context.Context, file io.Reader, path string) chan StreamResult {
	resultChan := make(chan StreamResult)
	go func() {
		resultChan <- &PNGParserStreamResult{
			FullPath:     path,
			CurrentStage: ProgressNew,
		}

		ocrProgress := p.ocrProvider.OCRWithProgress(ctx, file)
		for update := range ocrProgress.CompletionUpdates() {
			resultChan <- &PNGParserStreamResult{
				FullPath:        path,
				CurrentStage:    ProgressUpdate,
				CurrentProgress: update,
			}
		}
		text, err := ocrProgress.Text()
		if err != nil {
			resultChan <- &PNGParserStreamResult{
				FullPath:     path,
				CurrentStage: ProgressCompleted,
				Err:          err,
			}
		}

		resultChan <- &PNGParserStreamResult{
			FullPath:     path,
			Text:         text,
			CurrentStage: ProgressCompleted,
		}
		close(resultChan)
	}()
	return resultChan
}

type PNGParserResult struct {
	FullPath string `json:"path"`
	Text     string `json:"text"`
	Err      error  `json:"error"`
}

func (r *PNGParserResult) Path() string {
	return r.FullPath
}

func (r *PNGParserResult) String() string {
	return r.Text
}

func (r *PNGParserResult) Error() error {
	return r.Err
}

func (r *PNGParserResult) Subfiles() []Result {
	return nil
}

type PNGParserStreamResult struct {
	FullPath        string             `json:"path"`
	Text            string             `json:"text"`
	CurrentStage    ParseProgressStage `json:"stage"`
	CurrentProgress uint8              `json:"progress"`
	Err             error              `json:"error"`
}

func (r *PNGParserStreamResult) Path() string {
	return r.FullPath
}

func (r *PNGParserStreamResult) Stage() ParseProgressStage {
	return r.CurrentStage
}

func (r *PNGParserStreamResult) Progress() uint8 {
	return r.CurrentProgress
}

func (r *PNGParserStreamResult) SubResult() StreamResult {
	return nil
}

func (r *PNGParserStreamResult) String() string {
	return r.Text
}

func (r *PNGParserStreamResult) Error() error {
	return r.Err
}
