package bgra

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"io"

	"github.com/gabriel-vasile/mimetype"
)

var RAWBGRA_HEADER = []byte("FILE2LLM_RAW_RGBA______%%")

type BGRAImage struct {
	Width  uint64
	Height uint64
	Stride uint64
	Data   []byte
}

func (img *BGRAImage) ConvertBGRAtoRGBAInplace() *image.RGBA {
	convertBGRAtoRGBAInplaceFunc(int(img.Width), int(img.Height), int(img.Stride), img.Data, bgraToRgbaInPlaceFunc)
	rgbaIMG := &image.RGBA{
		Pix:    img.Data,
		Stride: int(img.Stride),
		Rect:   image.Rect(0, 0, int(img.Width), int(img.Height)),
	}
	return rgbaIMG
}

func ReadRAWBGRAImageFromReader(reader io.Reader) (*BGRAImage, error) {
	// Read and ommit header
	mimeHeader := make([]byte, len(RAWBGRA_HEADER))
	if _, err := io.ReadFull(reader, mimeHeader); err != nil {
		return nil, errors.Join(errors.New("failed to read header"), err)
	}
	if !bytes.Equal(mimeHeader, RAWBGRA_HEADER) {
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

	return &BGRAImage{
		Width:  width,
		Height: height,
		Stride: stride,
		Data:   data,
	}, nil
}

func ReadRAWBGRAImageFromBytes(fullData []byte) (*BGRAImage, error) {
	if len(fullData) < len(RAWBGRA_HEADER)+8+8+8 {
		return nil, errors.New("image data too small")
	}
	fullData = fullData[len(RAWBGRA_HEADER):]
	width := binary.BigEndian.Uint64(fullData)
	fullData = fullData[8:]
	height := binary.BigEndian.Uint64(fullData)
	fullData = fullData[8:]
	stride := binary.BigEndian.Uint64(fullData)
	fullData = fullData[8:]

	if height*width*4 > height*stride {
		return nil, errors.New("bad image data: height*width must be less or equal to height*stride")
	}

	if uint64(len(fullData)) != height*stride {
		return nil, errors.New("image data size doesnt match with height and stride")
	}

	return &BGRAImage{
		Width:  width,
		Height: height,
		Stride: stride,
		Data:   fullData,
	}, nil
}

func bgraMimeDetector(data []byte, limit uint32) bool {
	if limit < uint32(len(RAWBGRA_HEADER)) {
		return false
	}

	return bytes.HasPrefix(data, RAWBGRA_HEADER)
}

func init() {
	mimetype.Extend(bgraMimeDetector, "image/file2llm-raw-bgra", ".file2llm-raw-bgra")
}
