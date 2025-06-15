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

func (p *RAWBGRAParser) prepareData(file io.Reader) (io.Reader, error) {
	if p.convertToPNG {
		img, err := bgra.ReadRAWBGRAImageFromReader(file)
		if err != nil {
			return nil, errors.Join(errors.New("failed to read raw BGRA image"), err)
		}

		rgbaIMG := img.ConvertBGRAtoRGBAInplace()

		var outPNGImgBuf bytes.Buffer
		if err := png.Encode(&outPNGImgBuf, rgbaIMG); err != nil {
			return nil, errors.Join(errors.New("failed to convert image to PNG"), err)
		}

		return &outPNGImgBuf, nil
	}

	return file, nil
}

func (p *RAWBGRAParser) Parse(ctx context.Context, file io.Reader, path string) Result {
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

func (p *RAWBGRAParser) ParseStream(ctx context.Context, file io.Reader, path string) StreamResultIterator {
	return &ImageStreamResultIterator{
		path:             path,
		file:             file,
		imagePreparation: p.prepareData,
		ocrProvider:      p.ocrProvider,
		baseContext:      ctx,
	}
}
