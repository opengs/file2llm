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
		return &ImageParserResult{Err: errors.Join(errors.New("failed to prepare image data"), err), FullPath: path}
	}

	text, err := p.ocrProvider.OCR(ctx, imageData)
	if err != nil {
		return &ImageParserResult{Err: errors.Join(errors.New("errors while running OCR"), err), FullPath: path}
	}

	return &ImageParserResult{Text: text, FullPath: path}
}

func (p *TiffParser) ParseStream(ctx context.Context, file io.Reader, path string) StreamResultIterator {
	return &ImageStreamResultIterator{
		path:             path,
		file:             file,
		imagePreparation: p.prepareData,
		ocrProvider:      p.ocrProvider,
		baseContext:      ctx,
	}
}
