package parser

import (
	"bytes"
	"context"
	"errors"
	"io"

	"image/png"

	"github.com/opengs/file2llm/ocr"
	"golang.org/x/image/tiff"
)

// Parses `image/tiff` files
type TiffParser struct {
	ocrProvider ocr.Provider
}

func NewTiffParser(ocrProvider ocr.Provider) *TiffParser {
	return &TiffParser{
		ocrProvider: ocrProvider,
	}
}

func (p *TiffParser) SupportedMimeTypes() []string {
	return []string{"image/tiff"}
}

func (p *TiffParser) prepareData(file io.Reader) (io.Reader, error) {
	var imageData io.Reader

	if p.ocrProvider.IsMimeTypeSupported("image/tiff") {
		imageData = file
	} else {
		img, err := tiff.Decode(file)
		if err != nil {
			return nil, errors.Join(ErrBadFile, errors.New("failed to decode tiff image for transcoding"), err)
		}

		var outBuf bytes.Buffer
		if err := png.Encode(&outBuf, img); err != nil {
			return nil, errors.Join(errors.New("failed to transcode image to PNG"), err)
		}
		imageData = &outBuf
	}

	return imageData, nil
}

func (p *TiffParser) Parse(ctx context.Context, file io.Reader, path string) Result {
	imageData, err := p.prepareData(file)
	if err != nil {
		return &JPEGParserResult{Err: errors.Join(errors.New("failed to prepare image data"), err), FullPath: path}
	}

	text, err := p.ocrProvider.OCR(ctx, imageData)
	if err != nil {
		return &TiffParserResult{Err: errors.Join(errors.New("errors while running OCR"), err), FullPath: path}
	}

	return &TiffParserResult{Text: text, FullPath: path}
}

func (p *TiffParser) ParseStream(ctx context.Context, file io.Reader, path string) chan StreamResult {
	resultChan := make(chan StreamResult)
	go func() {
		defer close(resultChan)
		resultChan <- &TIFFParserStreamResult{FullPath: path, CurrentStage: ProgressNew}

		imageData, err := p.prepareData(file)
		if err != nil {
			resultChan <- &TIFFParserStreamResult{
				Err:          errors.Join(errors.New("failed to prepare image data"), err),
				FullPath:     path,
				CurrentStage: ProgressCompleted,
			}
		}

		ocrProgress := p.ocrProvider.OCRWithProgress(ctx, imageData)
		for update := range ocrProgress.CompletionUpdates() {
			resultChan <- &TIFFParserStreamResult{
				FullPath:        path,
				CurrentStage:    ProgressUpdate,
				CurrentProgress: update,
			}
		}
		text, err := ocrProgress.Text()
		if err != nil {
			resultChan <- &TIFFParserStreamResult{
				FullPath:     path,
				CurrentStage: ProgressCompleted,
				Err:          err,
			}
		}

		resultChan <- &TIFFParserStreamResult{
			FullPath:     path,
			Text:         text,
			CurrentStage: ProgressCompleted,
		}
	}()
	return resultChan
}

type TiffParserResult struct {
	FullPath string `json:"path"`
	Text     string `json:"text"`
	Err      error  `json:"error"`
}

func (r *TiffParserResult) Path() string {
	return r.FullPath
}

func (r *TiffParserResult) String() string {
	return r.Text
}

func (r *TiffParserResult) Error() error {
	return r.Err
}

func (r *TiffParserResult) Subfiles() []Result {
	return nil
}

type TIFFParserStreamResult struct {
	FullPath        string             `json:"path"`
	Text            string             `json:"text"`
	CurrentStage    ParseProgressStage `json:"stage"`
	CurrentProgress uint8              `json:"progress"`
	Err             error              `json:"error"`
}

func (r *TIFFParserStreamResult) Path() string {
	return r.FullPath
}

func (r *TIFFParserStreamResult) Stage() ParseProgressStage {
	return r.CurrentStage
}

func (r *TIFFParserStreamResult) Progress() uint8 {
	return r.CurrentProgress
}

func (r *TIFFParserStreamResult) SubResult() StreamResult {
	return nil
}

func (r *TIFFParserStreamResult) String() string {
	return r.Text
}

func (r *TIFFParserStreamResult) Error() error {
	return r.Err
}
