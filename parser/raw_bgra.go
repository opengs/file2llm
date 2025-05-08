package parser

import (
	"bytes"
	"context"
	"errors"
	"image/png"
	"io"

	"github.com/opengs/file2llm/ocr"
	"github.com/opengs/file2llm/parser/bgra"
)

// Parses internal `image/file2llm-raw-bgra` streams
type RAWBGRAParser struct {
	ocrProvider  ocr.Provider
	convertToPNG bool
}

func NewRAWBGRAParser(ocrProvider ocr.Provider) *RAWBGRAParser {
	return &RAWBGRAParser{
		ocrProvider:  ocrProvider,
		convertToPNG: !ocrProvider.IsMimeTypeSupported("image/file2llm-raw-bgra"),
	}
}

func (p *RAWBGRAParser) SupportedMimeTypes() []string {
	return []string{"image/file2llm-raw-bgra"}
}

func (p *RAWBGRAParser) Parse(ctx context.Context, file io.Reader) Result {
	if p.convertToPNG {
		img, err := bgra.ReadRAWBGRAImageFromReader(file)
		if err != nil {
			return &RAWBGRAParserResult{Err: errors.Join(errors.New("failed to read raw BGRA image"), err)}
		}

		rgbaIMG := img.ConvertBGRAtoRGBAInplace()

		var outPNGImgBuf bytes.Buffer
		if err := png.Encode(&outPNGImgBuf, rgbaIMG); err != nil {
			return &RAWBGRAParserResult{Err: errors.Join(errors.New("failed to convert image to PNG"), err)}
		}

		text, err := p.ocrProvider.OCR(ctx, &outPNGImgBuf)
		if err != nil {
			return &RAWBGRAParserResult{Err: errors.Join(errors.New("errors while running OCR"), err)}
		}

		return &RAWBGRAParserResult{Text: text}
	}

	text, err := p.ocrProvider.OCR(ctx, file)
	if err != nil {
		return &RAWBGRAParserResult{Err: errors.Join(errors.New("errors while running OCR"), err)}
	}

	return &RAWBGRAParserResult{Text: text}
}

type RAWBGRAParserResult struct {
	Text string `json:"text"`
	Err  error  `json:"error"`
}

func (r *RAWBGRAParserResult) String() string {
	return r.Text
}

func (r *RAWBGRAParserResult) Error() error {
	return r.Err
}

func (r *RAWBGRAParserResult) Componets() []Result {
	return nil
}
