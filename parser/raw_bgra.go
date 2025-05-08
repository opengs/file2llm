package parser

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"image"
	"image/png"
	"io"

	"github.com/gabriel-vasile/mimetype"
	"github.com/opengs/file2llm/ocr"
	"github.com/opengs/file2llm/parser/bgra2rgba"
)

var RAWRGBA_HEADER = []byte("FILE2LLM_RAW_RGBA______%%")

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
		img, err := ReadRAWBGRAImageFromReader(file)
		if err != nil {
			return &RAWBGRAParserResult{Err: errors.Join(errors.New("failed to read raw BGRA image"), err)}
		}

		bgra2rgba.ConvertBGRAtoRGBAInplace(int(img.Width), int(img.Height), int(img.Stride), img.Data)
		rgbaIMG := &image.RGBA{
			Pix:    img.Data,
			Stride: int(img.Stride),
			Rect:   image.Rect(0, 0, int(img.Width), int(img.Height)),
		}

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

type RAWBGRAImage struct {
	Width  uint64
	Height uint64
	Stride uint64
	Data   []byte
}

func ReadRAWBGRAImageFromReader(reader io.Reader) (*RAWBGRAImage, error) {
	// Read and ommit header
	mimeHeader := make([]byte, len(RAWRGBA_HEADER))
	if _, err := io.ReadFull(reader, mimeHeader); err != nil {
		return nil, errors.Join(errors.New("failed to read header"), err)
	}
	if !bytes.Equal(mimeHeader, RAWRGBA_HEADER) {
		return nil, errors.New("wrong data type: header not match")
	}

	var width uint64
	if err := binary.Read(reader, binary.BigEndian, &width); err != nil {
		return nil, errors.Join(errors.New("failed to read image width"), err)
	}
	var height uint64
	if err := binary.Read(reader, binary.BigEndian, &height); err != nil {
		return nil, errors.Join(errors.New("failed to read image height"), err)
	}
	var stride uint64
	if err := binary.Read(reader, binary.BigEndian, &stride); err != nil {
		return nil, errors.Join(errors.New("failed to read image stride"), err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.Join(errors.New("error while reading image data"), err)
	}

	if height*width*4 > height*stride {
		return nil, errors.New("bad image data: height*width must be less or equal to height*stride")
	}

	if uint64(len(data)) != height*stride {
		return nil, errors.New("image data size doesnt match with height and stride")
	}

	return &RAWBGRAImage{
		Width:  width,
		Height: height,
		Stride: stride,
		Data:   data,
	}, nil
}

func bgraMimeDetector(data []byte, limit uint32) bool {
	if limit < uint32(len(RAWRGBA_HEADER)) {
		return false
	}

	return bytes.HasPrefix(data, RAWRGBA_HEADER)
}

func init() {
	mimetype.Extend(bgraMimeDetector, "image/file2llm-raw-bgra", ".file2llm-raw-bgra")
}
