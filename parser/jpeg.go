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

func (p *JPEGParser) Parse(ctx context.Context, file io.Reader) Result {
	var imageData io.Reader
	var err error

	if p.ocrProvider.IsMimeTypeSupported("image/jpeg") {
		imageData = file
	} else {
		img, _, err := image.Decode(file)
		if err != nil {
			return &JPEGParserResult{Err: errors.Join(ErrBadFile, errors.New("failed to decode jpeg image for transcoding"), err)}
		}

		var outBuf bytes.Buffer
		if err := png.Encode(&outBuf, img); err != nil {
			return &JPEGParserResult{Err: errors.Join(errors.New("failed to transcode image to PNG"), err)}
		}
		imageData = &outBuf
	}

	text, err := p.ocrProvider.OCR(ctx, imageData)
	if err != nil {
		return &JPEGParserResult{Err: errors.Join(errors.New("errors while running OCR"), err)}
	}

	return &JPEGParserResult{Text: text}
}

type JPEGParserResult struct {
	Text string `json:"text"`
	Err  error  `json:"error"`
}

func (r *JPEGParserResult) String() string {
	return r.Text
}

func (r *JPEGParserResult) Error() error {
	return r.Err
}

func (r *JPEGParserResult) Componets() []Result {
	return nil
}
