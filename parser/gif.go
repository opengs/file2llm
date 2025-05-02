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

func (p *GIFParser) Parse(ctx context.Context, file io.Reader) Result {
	var imageData []byte
	var err error

	if p.ocrProvider.IsMimeTypeSupported("image/gif") {
		imageData, err = io.ReadAll(file)
		if err != nil {
			return &GIFParserResult{Err: errors.Join(errors.New("failed to read data to the bytes buffer"), err)}
		}
	} else {
		gifData, err := gif.DecodeAll(file)
		if err != nil {
			return &GIFParserResult{Err: errors.Join(ErrBadFile, errors.New("failed to decode gif image for transcoding"), err)}
		}
		if len(gifData.Image) == 0 {
			return &GIFParserResult{Err: errors.Join(ErrBadFile, errors.New("gif image has zero frames"))}
		}

		var outBuf bytes.Buffer
		if err := png.Encode(&outBuf, gifData.Image[0]); err != nil {
			return &GIFParserResult{Err: errors.Join(errors.New("failed to transcode image to PNG"), err)}
		}
		imageData = outBuf.Bytes()
	}

	text, err := p.ocrProvider.OCR(ctx, imageData)
	if err != nil {
		return &GIFParserResult{Err: errors.Join(errors.New("errors while running OCR"), err)}
	}

	return &GIFParserResult{Text: text}
}

type GIFParserResult struct {
	Text string `json:"text"`
	Err  error  `json:"error"`
}

func (r *GIFParserResult) String() string {
	return r.Text
}

func (r *GIFParserResult) Error() error {
	return r.Err
}

func (r *GIFParserResult) Componets() []Result {
	return nil
}
