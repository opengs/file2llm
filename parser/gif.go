package parser

import (
	"bytes"
	"context"
	"errors"
	"io"

	"image/gif"
	"image/png"

	"github.com/opengs/file2llm/ocr"
)

// Parses `image/gif` files. Only decodes first frame
type GIFParser struct {
	ocrProvider ocr.Provider
}

func NewGIFParser(ocrProvider ocr.Provider) *GIFParser {
	return &GIFParser{
		ocrProvider: ocrProvider,
	}
}

func (p *GIFParser) SupportedMimeTypes() []string {
	return []string{"image/gif"}
}

func (p *GIFParser) prepareData(file io.Reader) (io.Reader, error) {
	var imageData io.Reader

	if p.ocrProvider.IsMimeTypeSupported("image/gif") {
		imageData = file
	} else {
		img, err := gif.Decode(file)
		if err != nil {
			return nil, errors.Join(ErrBadFile, errors.New("failed to decode gif image for transcoding"), err)
		}

		var outBuf bytes.Buffer
		if err := png.Encode(&outBuf, img); err != nil {
			return nil, errors.Join(errors.New("failed to transcode image to PNG"), err)
		}
		imageData = &outBuf
	}

	return imageData, nil
}

func (p *GIFParser) Parse(ctx context.Context, file io.Reader, path string) Result {
	imageData, err := p.prepareData(file)
	if err != nil {
		return &JPEGParserResult{Err: errors.Join(errors.New("failed to prepare image data"), err), FullPath: path}
	}

	text, err := p.ocrProvider.OCR(ctx, imageData)
	if err != nil {
		return &GIFParserResult{Err: errors.Join(errors.New("errors while running OCR"), err), FullPath: path}
	}

	return &GIFParserResult{Text: text, FullPath: path}
}

func (p *GIFParser) ParseStream(ctx context.Context, file io.Reader, path string) chan StreamResult {
	resultChan := make(chan StreamResult)
	go func() {
		defer close(resultChan)
		resultChan <- &GIFParserStreamResult{FullPath: path, CurrentStage: ProgressNew}

		imageData, err := p.prepareData(file)
		if err != nil {
			resultChan <- &GIFParserStreamResult{
				Err:          errors.Join(errors.New("failed to prepare image data"), err),
				FullPath:     path,
				CurrentStage: ProgressCompleted,
			}
		}

		ocrProgress := p.ocrProvider.OCRWithProgress(ctx, imageData)
		for update := range ocrProgress.CompletionUpdates() {
			resultChan <- &GIFParserStreamResult{
				FullPath:        path,
				CurrentStage:    ProgressUpdate,
				CurrentProgress: update,
			}
		}
		text, err := ocrProgress.Text()
		if err != nil {
			resultChan <- &GIFParserStreamResult{
				FullPath:     path,
				CurrentStage: ProgressCompleted,
				Err:          err,
			}
		}

		resultChan <- &GIFParserStreamResult{
			FullPath:     path,
			Text:         text,
			CurrentStage: ProgressCompleted,
		}
	}()
	return resultChan
}

type GIFParserResult struct {
	FullPath string `json:"path"`
	Text     string `json:"text"`
	Err      error  `json:"error"`
}

func (r *GIFParserResult) Path() string {
	return r.FullPath
}

func (r *GIFParserResult) String() string {
	return r.Text
}

func (r *GIFParserResult) Error() error {
	return r.Err
}

func (r *GIFParserResult) Subfiles() []Result {
	return nil
}

type GIFParserStreamResult struct {
	FullPath        string             `json:"path"`
	Text            string             `json:"text"`
	CurrentStage    ParseProgressStage `json:"stage"`
	CurrentProgress uint8              `json:"progress"`
	Err             error              `json:"error"`
}

func (r *GIFParserStreamResult) Path() string {
	return r.FullPath
}

func (r *GIFParserStreamResult) Stage() ParseProgressStage {
	return r.CurrentStage
}

func (r *GIFParserStreamResult) Progress() uint8 {
	return r.CurrentProgress
}

func (r *GIFParserStreamResult) SubResult() StreamResult {
	return nil
}

func (r *GIFParserStreamResult) String() string {
	return r.Text
}

func (r *GIFParserStreamResult) Error() error {
	return r.Err
}
