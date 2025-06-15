package parser

import (
	"context"
	"io"

	"github.com/opengs/file2llm/ocr"
)

type ImageParserResult struct {
	FullPath string `json:"path"`
	Text     string `json:"text"`
	Err      error  `json:"error"`
}

func (r *ImageParserResult) Path() string {
	return r.FullPath
}

func (r *ImageParserResult) String() string {
	return r.Text
}

func (r *ImageParserResult) Error() error {
	return r.Err
}

func (r *ImageParserResult) Subfiles() []Result {
	return nil
}

type ImageParserStreamResult struct {
	FullPath        string             `json:"path"`
	Text            string             `json:"text"`
	CurrentStage    ParseProgressStage `json:"stage"`
	CurrentProgress uint8              `json:"progress"`
	Err             error              `json:"error"`
}

func (r *ImageParserStreamResult) Path() string {
	return r.FullPath
}

func (r *ImageParserStreamResult) Stage() ParseProgressStage {
	return r.CurrentStage
}

func (r *ImageParserStreamResult) Progress() uint8 {
	return r.CurrentProgress
}

func (r *ImageParserStreamResult) SubResult() StreamResult {
	return nil
}

func (r *ImageParserStreamResult) String() string {
	return r.Text
}

func (r *ImageParserStreamResult) Error() error {
	return r.Err
}

type ImageStreamResultIterator struct {
	path                  string
	ocrProvider           ocr.Provider
	file                  io.Reader
	imagePreparation      func(file io.Reader) (io.Reader, error)
	imagePreparationError error
	imagePrepared         bool

	baseContext context.Context
	ocrContext  context.Context
	ocrCancel   context.CancelFunc
	ocrProgress ocr.OCRProgress

	completed bool

	current StreamResult
}

func (i *ImageStreamResultIterator) Next(ctx context.Context) bool {
	if i.completed {
		i.current = nil
		return false
	}

	if i.imagePreparationError != nil {
		i.completed = true
		i.current = &ImageParserStreamResult{
			FullPath:     i.path,
			CurrentStage: ProgressCompleted,
			Err:          i.imagePreparationError,
		}
		return true
	}

	if !i.imagePrepared && i.imagePreparation != nil {
		i.imagePrepared = true
		i.file, i.imagePreparationError = i.imagePreparation(i.file)
		i.current = &ImageParserStreamResult{
			FullPath:     i.path,
			CurrentStage: ProgressNew,
		}
		if i.imagePreparationError != nil {
			return true
		}
	}

	if i.ocrProgress == nil {
		i.ocrContext, i.ocrCancel = context.WithCancel(i.baseContext)
		i.ocrProgress = i.ocrProvider.OCRWithProgress(i.ocrContext, i.file)
		i.current = &ImageParserStreamResult{
			FullPath:     i.path,
			CurrentStage: ProgressNew,
		}
		return true
	}

	select {
	case progress, ok := <-i.ocrProgress.CompletionUpdates():
		if ok {
			i.current = &ImageParserStreamResult{
				FullPath:        i.path,
				CurrentStage:    ProgressUpdate,
				CurrentProgress: progress,
			}
			return true
		} else {
			i.completed = true
			text, err := i.ocrProgress.Text()
			i.current = &ImageParserStreamResult{
				FullPath:     i.path,
				CurrentStage: ProgressCompleted,
				Text:         text,
				Err:          err,
			}
			return true
		}
	case <-ctx.Done():
		i.current = nil
		return false
	}
}

func (i *ImageStreamResultIterator) Current() StreamResult {
	return i.current
}

func (i *ImageStreamResultIterator) Close() {
	if i.ocrCancel != nil {
		i.ocrCancel()
		i.ocrCancel = nil
		i.ocrProgress.Text() // just wait for the end
		i.completed = true
	}
}
