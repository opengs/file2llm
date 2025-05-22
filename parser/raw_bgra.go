package parser

import (
	"bytes"
	"context"
	"errors"
	"image/png"
	"io"

	"github.com/opengs/file2llm/ocr"
	"github.com/opengs/file2llm/parser/bgra"
)

// Parses internal `image/file2llm-raw-bgra` streams
type RAWBGRAParser struct {
	ocrProvider  ocr.Provider
	convertToPNG bool
}

func NewRAWBGRAParser(ocrProvider ocr.Provider) *RAWBGRAParser {
	return &RAWBGRAParser{
		ocrProvider:  ocrProvider,
		convertToPNG: !ocrProvider.IsMimeTypeSupported("image/file2llm-raw-bgra"),
	}
}

func (p *RAWBGRAParser) SupportedMimeTypes() []string {
	return []string{"image/file2llm-raw-bgra"}
}

func (p *RAWBGRAParser) Parse(ctx context.Context, file io.Reader, path string) Result {
	if p.convertToPNG {
		img, err := bgra.ReadRAWBGRAImageFromReader(file)
		if err != nil {
			return &RAWBGRAParserResult{Err: errors.Join(errors.New("failed to read raw BGRA image"), err), FullPath: path}
		}

		rgbaIMG := img.ConvertBGRAtoRGBAInplace()

		var outPNGImgBuf bytes.Buffer
		if err := png.Encode(&outPNGImgBuf, rgbaIMG); err != nil {
			return &RAWBGRAParserResult{Err: errors.Join(errors.New("failed to convert image to PNG"), err), FullPath: path}
		}

		text, err := p.ocrProvider.OCR(ctx, &outPNGImgBuf)
		if err != nil {
			return &RAWBGRAParserResult{Err: errors.Join(errors.New("errors while running OCR"), err), FullPath: path}
		}

		return &RAWBGRAParserResult{Text: text, FullPath: path}
	}

	text, err := p.ocrProvider.OCR(ctx, file)
	if err != nil {
		return &RAWBGRAParserResult{Err: errors.Join(errors.New("errors while running OCR"), err), FullPath: path}
	}

	return &RAWBGRAParserResult{Text: text, FullPath: path}
}

func (p *RAWBGRAParser) ParseStream(ctx context.Context, file io.Reader, path string) chan StreamResult {
	resultChan := make(chan StreamResult)
	go func() {
		defer close(resultChan)
		resultChan <- &RAWBGRAParserStreamResult{FullPath: path, CurrentStage: ProgressNew}

		if p.convertToPNG {
			img, err := bgra.ReadRAWBGRAImageFromReader(file)
			if err != nil {
				resultChan <- &RAWBGRAParserStreamResult{
					Err:          errors.Join(errors.New("failed to read raw BGRA image"), err),
					CurrentStage: ProgressCompleted,
					FullPath:     path}
				return
			}

			rgbaIMG := img.ConvertBGRAtoRGBAInplace()

			var outPNGImgBuf bytes.Buffer
			if err := png.Encode(&outPNGImgBuf, rgbaIMG); err != nil {
				resultChan <- &RAWBGRAParserStreamResult{
					Err:          errors.Join(errors.New("failed to convert image to PNG"), err),
					CurrentStage: ProgressCompleted,
					FullPath:     path,
				}
				return
			}

			ocrProgress := p.ocrProvider.OCRWithProgress(ctx, &outPNGImgBuf)
			for update := range ocrProgress.CompletionUpdates() {
				resultChan <- &RAWBGRAParserStreamResult{
					FullPath:        path,
					CurrentStage:    ProgressUpdate,
					CurrentProgress: update,
				}
			}
			text, err := ocrProgress.Text()
			if err != nil {
				resultChan <- &RAWBGRAParserStreamResult{
					FullPath:     path,
					CurrentStage: ProgressCompleted,
					Err:          err,
				}
			}

			resultChan <- &RAWBGRAParserStreamResult{
				FullPath:     path,
				Text:         text,
				CurrentStage: ProgressCompleted,
			}
			return
		}

		ocrProgress := p.ocrProvider.OCRWithProgress(ctx, file)
		for update := range ocrProgress.CompletionUpdates() {
			resultChan <- &RAWBGRAParserStreamResult{
				FullPath:        path,
				CurrentStage:    ProgressUpdate,
				CurrentProgress: update,
			}
		}
		text, err := ocrProgress.Text()
		if err != nil {
			resultChan <- &RAWBGRAParserStreamResult{
				FullPath:     path,
				CurrentStage: ProgressCompleted,
				Err:          err,
			}
		}

		resultChan <- &RAWBGRAParserStreamResult{
			FullPath:     path,
			Text:         text,
			CurrentStage: ProgressCompleted,
		}
	}()
	return resultChan
}

type RAWBGRAParserResult struct {
	FullPath string `json:"path"`
	Text     string `json:"text"`
	Err      error  `json:"error"`
}

func (r *RAWBGRAParserResult) Path() string {
	return r.FullPath
}

func (r *RAWBGRAParserResult) String() string {
	return r.Text
}

func (r *RAWBGRAParserResult) Error() error {
	return r.Err
}

func (r *RAWBGRAParserResult) Subfiles() []Result {
	return nil
}

type RAWBGRAParserStreamResult struct {
	FullPath        string             `json:"path"`
	Text            string             `json:"text"`
	CurrentStage    ParseProgressStage `json:"stage"`
	CurrentProgress uint8              `json:"progress"`
	Err             error              `json:"error"`
}

func (r *RAWBGRAParserStreamResult) Path() string {
	return r.FullPath
}

func (r *RAWBGRAParserStreamResult) Stage() ParseProgressStage {
	return r.CurrentStage
}

func (r *RAWBGRAParserStreamResult) Progress() uint8 {
	return r.CurrentProgress
}

func (r *RAWBGRAParserStreamResult) SubResult() StreamResult {
	return nil
}

func (r *RAWBGRAParserStreamResult) String() string {
	return r.Text
}

func (r *RAWBGRAParserStreamResult) Error() error {
	return r.Err
}
