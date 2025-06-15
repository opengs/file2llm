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
		return &ImageParserResult{Err: errors.Join(errors.New("failed to prepare image data"), err), FullPath: path}
	}

	text, err := p.ocrProvider.OCR(ctx, imageData)
	if err != nil {
		return &ImageParserResult{Err: errors.Join(errors.New("errors while running OCR"), err), FullPath: path}
	}

	return &ImageParserResult{Text: text, FullPath: path}
}

func (p *JPEGParser) ParseStream(ctx context.Context, file io.Reader, path string) StreamResultIterator {
	return &ImageStreamResultIterator{
		path:             path,
		file:             file,
		imagePreparation: p.prepareData,
		ocrProvider:      p.ocrProvider,
		baseContext:      ctx,
	}
}
