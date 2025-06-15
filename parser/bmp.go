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
		return &ImageParserResult{Err: errors.Join(errors.New("failed to prepare image data"), err), FullPath: path}
	}

	text, err := p.ocrProvider.OCR(ctx, imageData)
	if err != nil {
		return &ImageParserResult{Err: errors.Join(errors.New("errors while running OCR"), err), FullPath: path}
	}

	return &ImageParserResult{Text: text, FullPath: path}
}

func (p *BMPParser) ParseStream(ctx context.Context, file io.Reader, path string) StreamResultIterator {
	return &ImageStreamResultIterator{
		path:             path,
		file:             file,
		imagePreparation: p.prepareData,
		ocrProvider:      p.ocrProvider,
		baseContext:      ctx,
	}
}
