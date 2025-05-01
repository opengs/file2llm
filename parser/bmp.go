package parser

import (
	"bufio"
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

func (p *BMPParser) Parse(ctx context.Context, file io.Reader) Result {
	var imageData []byte
	var err error

	if p.ocrProvider.IsMimeTypeSupported("image/bmp") {
		imageData, err = io.ReadAll(file)
		if err != nil {
			return &BMPParserResult{Err: errors.Join(errors.New("failed to read data to the bytes buffer"), err)}
		}
	} else {
		img, err := bmp.Decode(file)
		if err != nil {
			return &BMPParserResult{Err: errors.Join(ErrBadFile, errors.New("failed to decode bmp image for transcoding"), err)}
		}

		var outBuf bytes.Buffer
		if err := png.Encode(bufio.NewWriter(&outBuf), img); err != nil {
			return &BMPParserResult{Err: errors.Join(errors.New("failed to transcode image to PNG"), err)}
		}
		imageData = outBuf.Bytes()
	}

	text, err := p.ocrProvider.OCR(ctx, imageData)
	if err != nil {
		return &BMPParserResult{Err: errors.Join(errors.New("errors while running OCR"), err)}
	}

	return &BMPParserResult{Text: text}
}

type BMPParserResult struct {
	Text string `json:"text"`
	Err  error  `json:"error"`
}

func (r *BMPParserResult) String() string {
	return r.Text
}

func (r *BMPParserResult) Error() error {
	return r.Err
}

func (r *BMPParserResult) Componets() []Result {
	return nil
}
