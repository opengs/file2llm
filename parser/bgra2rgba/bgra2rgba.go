package bgra2rgba

import (
	"runtime"

	"golang.org/x/sys/cpu"
)

func bgraToRgbaInPlaceAVX2(data []byte)
func bgraToRgbaInPlaceSSE3(data []byte)

func bgraToRgbaInPlaceFallback(data []byte) {
	n := len(data) / 4
	for i := 0; i < n; i++ {
		base := i * 4
		b := data[base+0]
		//g := data[base+1]
		r := data[base+2]
		//a := data[base+3]

		data[base+0] = r
		//data[base+1] = g
		data[base+2] = b
		//data[base+3] = a
	}
}

var bgraToRgbaInPlaceFunc func(dst []byte)

func init() {
	if runtime.GOARCH != "amd64" {
		// Non-x86 CPUs (ARM, ARM64, etc.) — always fallback
		bgraToRgbaInPlaceFunc = bgraToRgbaInPlaceFallback
		return
	}

	switch {
	case cpu.X86.HasAVX2:
		bgraToRgbaInPlaceFunc = bgraToRgbaInPlaceAVX2
	case cpu.X86.HasSSE2 && cpu.X86.HasSSE3:
		bgraToRgbaInPlaceFunc = bgraToRgbaInPlaceSSE3
	default:
		bgraToRgbaInPlaceFunc = bgraToRgbaInPlaceFallback
	}
}

func convertBGRAtoRGBAInplaceFunc(width, height, stride int, data []byte, f func(dst []byte)) {
	if stride == width*4 {
		f(data)
		return
	}

	for y := range height {
		start := y * stride
		end := start + width*4
		f(data[start:end])
	}
}

func ConvertBGRAtoRGBAInplace(width, height, stride int, data []byte) {
	convertBGRAtoRGBAInplaceFunc(width, height, stride, data, bgraToRgbaInPlaceFunc)
}
