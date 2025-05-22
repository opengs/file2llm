package parser

import (
	"bytes"
	"context"
	"errors"
	"io"

	"image/png"

	"github.com/opengs/file2llm/ocr"
	"golang.org/x/image/bmp"
)

// Parses `image/bmp` files
type BMPParser struct {
	ocrProvider ocr.Provider
}

func NewBMPParser(ocrProvider ocr.Provider) *BMPParser {
	return &BMPParser{
		ocrProvider: ocrProvider,
	}
}

func (p *BMPParser) SupportedMimeTypes() []string {
	return []string{"image/bmp"}
}

func (p *BMPParser) prepareData(file io.Reader) (io.Reader, error) {
	var imageData io.Reader

	if p.ocrProvider.IsMimeTypeSupported("image/bmp") {
		imageData = file
	} else {
		img, err := bmp.Decode(file)
		if err != nil {
			return nil, errors.Join(ErrBadFile, errors.New("failed to decode bmp image for transcoding"), err)
		}

		var outBuf bytes.Buffer
		if err := png.Encode(&outBuf, img); err != nil {
			return nil, errors.Join(errors.New("failed to transcode image to PNG"), err)
		}
		imageData = &outBuf
	}

	return imageData, nil
}

func (p *BMPParser) Parse(ctx context.Context, file io.Reader, path string) Result {
	imageData, err := p.prepareData(file)
	if err != nil {
		return &JPEGParserResult{Err: errors.Join(errors.New("failed to prepare image data"), err), FullPath: path}
	}

	text, err := p.ocrProvider.OCR(ctx, imageData)
	if err != nil {
		return &BMPParserResult{Err: errors.Join(errors.New("errors while running OCR"), err), FullPath: path}
	}

	return &BMPParserResult{Text: text, FullPath: path}
}

func (p *BMPParser) ParseStream(ctx context.Context, file io.Reader, path string) chan StreamResult {
	resultChan := make(chan StreamResult)
	go func() {
		defer close(resultChan)
		resultChan <- &BMPParserStreamResult{FullPath: path, CurrentStage: ProgressNew}

		imageData, err := p.prepareData(file)
		if err != nil {
			resultChan <- &BMPParserStreamResult{
				Err:          errors.Join(errors.New("failed to prepare image data"), err),
				FullPath:     path,
				CurrentStage: ProgressCompleted,
			}
		}

		ocrProgress := p.ocrProvider.OCRWithProgress(ctx, imageData)
		for update := range ocrProgress.CompletionUpdates() {
			resultChan <- &BMPParserStreamResult{
				FullPath:        path,
				CurrentStage:    ProgressUpdate,
				CurrentProgress: update,
			}
		}
		text, err := ocrProgress.Text()
		if err != nil {
			resultChan <- &BMPParserStreamResult{
				FullPath:     path,
				CurrentStage: ProgressCompleted,
				Err:          err,
			}
		}

		resultChan <- &BMPParserStreamResult{
			FullPath:     path,
			Text:         text,
			CurrentStage: ProgressCompleted,
		}
	}()
	return resultChan
}

type BMPParserResult struct {
	FullPath string `json:"path"`
	Text     string `json:"text"`
	Err      error  `json:"error"`
}

func (r *BMPParserResult) Path() string {
	return r.FullPath
}

func (r *BMPParserResult) String() string {
	return r.Text
}

func (r *BMPParserResult) Error() error {
	return r.Err
}

func (r *BMPParserResult) Subfiles() []Result {
	return nil
}

type BMPParserStreamResult struct {
	FullPath        string             `json:"path"`
	Text            string             `json:"text"`
	CurrentStage    ParseProgressStage `json:"stage"`
	CurrentProgress uint8              `json:"progress"`
	Err             error              `json:"error"`
}

func (r *BMPParserStreamResult) Path() string {
	return r.FullPath
}

func (r *BMPParserStreamResult) Stage() ParseProgressStage {
	return r.CurrentStage
}

func (r *BMPParserStreamResult) Progress() uint8 {
	return r.CurrentProgress
}

func (r *BMPParserStreamResult) SubResult() StreamResult {
	return nil
}

func (r *BMPParserStreamResult) String() string {
	return r.Text
}

func (r *BMPParserStreamResult) Error() error {
	return r.Err
}
