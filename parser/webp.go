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
		return &ImageParserResult{Err: errors.Join(errors.New("failed to prepare image data"), err), FullPath: path}
	}

	text, err := p.ocrProvider.OCR(ctx, imageData)
	if err != nil {
		return &ImageParserResult{Err: errors.Join(errors.New("errors while running OCR"), err), FullPath: path}
	}

	return &ImageParserResult{Text: text, FullPath: path}
}

func (p *WebPParser) ParseStream(ctx context.Context, file io.Reader, path string) StreamResultIterator {
	return &ImageStreamResultIterator{
		path:             path,
		file:             file,
		imagePreparation: p.prepareData,
		ocrProvider:      p.ocrProvider,
		baseContext:      ctx,
	}
}
