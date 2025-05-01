package parser

import (
	"bufio"
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

func (p *WebPParser) Parse(ctx context.Context, file io.Reader) Result {
	var imageData []byte
	var err error

	if p.ocrProvider.IsMimeTypeSupported("image/webp") {
		imageData, err = io.ReadAll(file)
		if err != nil {
			return &WebPParserResult{Err: errors.Join(errors.New("failed to read data to the bytes buffer"), err)}
		}
	} else {
		img, err := webp.Decode(file)
		if err != nil {
			return &WebPParserResult{Err: errors.Join(ErrBadFile, errors.New("failed to decode webp image for transcoding"), err)}
		}

		var outBuf bytes.Buffer
		if err := png.Encode(bufio.NewWriter(&outBuf), img); err != nil {
			return &WebPParserResult{Err: errors.Join(errors.New("failed to transcode image to PNG"), err)}
		}
		imageData = outBuf.Bytes()
	}

	text, err := p.ocrProvider.OCR(ctx, imageData)
	if err != nil {
		return &WebPParserResult{Err: errors.Join(errors.New("errors while running OCR"), err)}
	}

	return &WebPParserResult{Text: text}
}

type WebPParserResult struct {
	Text string `json:"text"`
	Err  error  `json:"error"`
}

func (r *WebPParserResult) String() string {
	return r.Text
}

func (r *WebPParserResult) Error() error {
	return r.Err
}

func (r *WebPParserResult) Componets() []Result {
	return nil
}
