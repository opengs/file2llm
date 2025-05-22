package parser

import (
	"bytes"
	"context"
	"errors"
	"io"

	"image/png"

	"github.com/opengs/file2llm/ocr"
	"golang.org/x/image/webp"
)

// Parses `image/webp` files
type WebPParser struct {
	ocrProvider ocr.Provider
}

func NewWebPParser(ocrProvider ocr.Provider) *WebPParser {
	return &WebPParser{
		ocrProvider: ocrProvider,
	}
}

func (p *WebPParser) SupportedMimeTypes() []string {
	return []string{"image/webp"}
}

func (p *WebPParser) prepareData(file io.Reader) (io.Reader, error) {
	var imageData io.Reader

	if p.ocrProvider.IsMimeTypeSupported("image/webp") {
		imageData = file
	} else {
		img, err := webp.Decode(file)
		if err != nil {
			return nil, errors.Join(ErrBadFile, errors.New("failed to decode webp image for transcoding"), err)
		}

		var outBuf bytes.Buffer
		if err := png.Encode(&outBuf, img); err != nil {
			return nil, errors.Join(errors.New("failed to transcode image to PNG"), err)
		}
		imageData = &outBuf
	}

	return imageData, nil
}

func (p *WebPParser) Parse(ctx context.Context, file io.Reader, path string) Result {
	imageData, err := p.prepareData(file)
	if err != nil {
		return &JPEGParserResult{Err: errors.Join(errors.New("failed to prepare image data"), err), FullPath: path}
	}

	text, err := p.ocrProvider.OCR(ctx, imageData)
	if err != nil {
		return &WebPParserResult{Err: errors.Join(errors.New("errors while running OCR"), err), FullPath: path}
	}

	return &WebPParserResult{Text: text, FullPath: path}
}

func (p *WebPParser) ParseStream(ctx context.Context, file io.Reader, path string) chan StreamResult {
	resultChan := make(chan StreamResult)
	go func() {
		defer close(resultChan)
		resultChan <- &WEBPParserStreamResult{FullPath: path, CurrentStage: ProgressNew}

		imageData, err := p.prepareData(file)
		if err != nil {
			resultChan <- &WEBPParserStreamResult{
				Err:          errors.Join(errors.New("failed to prepare image data"), err),
				FullPath:     path,
				CurrentStage: ProgressCompleted,
			}
		}

		ocrProgress := p.ocrProvider.OCRWithProgress(ctx, imageData)
		for update := range ocrProgress.CompletionUpdates() {
			resultChan <- &WEBPParserStreamResult{
				FullPath:        path,
				CurrentStage:    ProgressUpdate,
				CurrentProgress: update,
			}
		}
		text, err := ocrProgress.Text()
		if err != nil {
			resultChan <- &WEBPParserStreamResult{
				FullPath:     path,
				CurrentStage: ProgressCompleted,
				Err:          err,
			}
		}

		resultChan <- &WEBPParserStreamResult{
			FullPath:     path,
			Text:         text,
			CurrentStage: ProgressCompleted,
		}
	}()
	return resultChan
}

type WebPParserResult struct {
	FullPath string `json:"path"`
	Text     string `json:"text"`
	Err      error  `json:"error"`
}

func (r *WebPParserResult) Path() string {
	return r.FullPath
}

func (r *WebPParserResult) String() string {
	return r.Text
}

func (r *WebPParserResult) Error() error {
	return r.Err
}

func (r *WebPParserResult) Subfiles() []Result {
	return nil
}

type WEBPParserStreamResult struct {
	FullPath        string             `json:"path"`
	Text            string             `json:"text"`
	CurrentStage    ParseProgressStage `json:"stage"`
	CurrentProgress uint8              `json:"progress"`
	Err             error              `json:"error"`
}

func (r *WEBPParserStreamResult) Path() string {
	return r.FullPath
}

func (r *WEBPParserStreamResult) Stage() ParseProgressStage {
	return r.CurrentStage
}

func (r *WEBPParserStreamResult) Progress() uint8 {
	return r.CurrentProgress
}

func (r *WEBPParserStreamResult) SubResult() StreamResult {
	return nil
}

func (r *WEBPParserStreamResult) String() string {
	return r.Text
}

func (r *WEBPParserStreamResult) Error() error {
	return r.Err
}
