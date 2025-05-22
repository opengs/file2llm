package parser

import (
	"bytes"
	"context"
	"errors"
	"image"
	"io"

	_ "image/jpeg"
	"image/png"

	"github.com/opengs/file2llm/ocr"
)

// Parses `image/jpeg` files
type JPEGParser struct {
	ocrProvider ocr.Provider
}

func NewJPEGParser(ocrProvider ocr.Provider) *JPEGParser {
	return &JPEGParser{
		ocrProvider: ocrProvider,
	}
}

func (p *JPEGParser) SupportedMimeTypes() []string {
	return []string{"image/jpeg"}
}

func (p *JPEGParser) prepareData(file io.Reader) (io.Reader, error) {
	var imageData io.Reader

	if p.ocrProvider.IsMimeTypeSupported("image/jpeg") {
		imageData = file
	} else {
		img, _, err := image.Decode(file)
		if err != nil {
			return nil, errors.Join(ErrBadFile, errors.New("failed to decode jpeg image for transcoding"), err)
		}

		var outBuf bytes.Buffer
		if err := png.Encode(&outBuf, img); err != nil {
			return nil, errors.Join(errors.New("failed to transcode image to PNG"), err)
		}
		imageData = &outBuf
	}

	return imageData, nil
}

func (p *JPEGParser) Parse(ctx context.Context, file io.Reader, path string) Result {
	imageData, err := p.prepareData(file)
	if err != nil {
		return &JPEGParserResult{Err: errors.Join(errors.New("failed to prepare image data"), err), FullPath: path}
	}

	text, err := p.ocrProvider.OCR(ctx, imageData)
	if err != nil {
		return &JPEGParserResult{Err: errors.Join(errors.New("errors while running OCR"), err), FullPath: path}
	}

	return &JPEGParserResult{Text: text, FullPath: path}
}

func (p *JPEGParser) ParseStream(ctx context.Context, file io.Reader, path string) chan StreamResult {
	resultChan := make(chan StreamResult)
	go func() {
		defer close(resultChan)
		resultChan <- &JPEGParserStreamResult{FullPath: path, CurrentStage: ProgressNew}

		imageData, err := p.prepareData(file)
		if err != nil {
			resultChan <- &JPEGParserStreamResult{
				Err:          errors.Join(errors.New("failed to prepare image data"), err),
				FullPath:     path,
				CurrentStage: ProgressCompleted,
			}
		}

		ocrProgress := p.ocrProvider.OCRWithProgress(ctx, imageData)
		for update := range ocrProgress.CompletionUpdates() {
			resultChan <- &JPEGParserStreamResult{
				FullPath:        path,
				CurrentStage:    ProgressUpdate,
				CurrentProgress: update,
			}
		}
		text, err := ocrProgress.Text()
		if err != nil {
			resultChan <- &JPEGParserStreamResult{
				FullPath:     path,
				CurrentStage: ProgressCompleted,
				Err:          err,
			}
		}

		resultChan <- &JPEGParserStreamResult{
			FullPath:     path,
			Text:         text,
			CurrentStage: ProgressCompleted,
		}
	}()
	return resultChan
}

type JPEGParserResult struct {
	FullPath string `json:"path"`
	Text     string `json:"text"`
	Err      error  `json:"error"`
}

func (r *JPEGParserResult) Path() string {
	return r.FullPath
}

func (r *JPEGParserResult) String() string {
	return r.Text
}

func (r *JPEGParserResult) Error() error {
	return r.Err
}

func (r *JPEGParserResult) Subfiles() []Result {
	return nil
}

type JPEGParserStreamResult struct {
	FullPath        string             `json:"path"`
	Text            string             `json:"text"`
	CurrentStage    ParseProgressStage `json:"stage"`
	CurrentProgress uint8              `json:"progress"`
	Err             error              `json:"error"`
}

func (r *JPEGParserStreamResult) Path() string {
	return r.FullPath
}

func (r *JPEGParserStreamResult) Stage() ParseProgressStage {
	return r.CurrentStage
}

func (r *JPEGParserStreamResult) Progress() uint8 {
	return r.CurrentProgress
}

func (r *JPEGParserStreamResult) SubResult() StreamResult {
	return nil
}

func (r *JPEGParserStreamResult) String() string {
	return r.Text
}

func (r *JPEGParserStreamResult) Error() error {
	return r.Err
}
