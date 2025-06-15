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
		return &ImageParserResult{Err: errors.Join(errors.New("failed to prepare image data"), err), FullPath: path}
	}

	text, err := p.ocrProvider.OCR(ctx, imageData)
	if err != nil {
		return &ImageParserResult{Err: errors.Join(errors.New("errors while running OCR"), err), FullPath: path}
	}

	return &ImageParserResult{Text: text, FullPath: path}
}

func (p *GIFParser) ParseStream(ctx context.Context, file io.Reader, path string) StreamResultIterator {
	return &ImageStreamResultIterator{
		path:             path,
		file:             file,
		imagePreparation: p.prepareData,
		ocrProvider:      p.ocrProvider,
		baseContext:      ctx,
	}
}
