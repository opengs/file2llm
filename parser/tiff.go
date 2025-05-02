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

func (p *TiffParser) Parse(ctx context.Context, file io.Reader) Result {
	var imageData []byte
	var err error

	if p.ocrProvider.IsMimeTypeSupported("image/tiff") {
		imageData, err = io.ReadAll(file)
		if err != nil {
			return &TiffParserResult{Err: errors.Join(errors.New("failed to read data to the bytes buffer"), err)}
		}
	} else {
		img, err := tiff.Decode(file)
		if err != nil {
			return &TiffParserResult{Err: errors.Join(ErrBadFile, errors.New("failed to decode tiff image for transcoding"), err)}
		}

		var outBuf bytes.Buffer
		if err := png.Encode(&outBuf, img); err != nil {
			return &TiffParserResult{Err: errors.Join(errors.New("failed to transcode image to PNG"), err)}
		}
		imageData = outBuf.Bytes()
	}

	text, err := p.ocrProvider.OCR(ctx, imageData)
	if err != nil {
		return &TiffParserResult{Err: errors.Join(errors.New("errors while running OCR"), err)}
	}

	return &TiffParserResult{Text: text}
}

type TiffParserResult struct {
	Text string `json:"text"`
	Err  error  `json:"error"`
}

func (r *TiffParserResult) String() string {
	return r.Text
}

func (r *TiffParserResult) Error() error {
	return r.Err
}

func (r *TiffParserResult) Componets() []Result {
	return nil
}
