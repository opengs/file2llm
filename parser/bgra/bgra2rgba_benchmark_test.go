package bgra

import "testing"

func BenchmarkConvertBGRAtoRGBA_AVX_InPlace(b *testing.B) {
	width, height := 1920, 1080
	stride := 4 * width
	data := make([]byte, stride*height)

	for i := 0; i < len(data); i += 4 {
		data[i] = 0xFF
		data[i+1] = 0x00
		data[i+2] = 0x00
		data[i+3] = 0xFF
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		convertBGRAtoRGBAInplaceFunc(width, height, stride, data, bgraToRgbaInPlaceAVX2)
	}
}

func BenchmarkConvertBGRAtoRGBA_SSE3_InPlace(b *testing.B) {
	width, height := 1920, 1080
	stride := 4 * width
	data := make([]byte, stride*height)

	for i := 0; i < len(data); i += 4 {
		data[i] = 0xFF
		data[i+1] = 0x00
		data[i+2] = 0x00
		data[i+3] = 0xFF
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		convertBGRAtoRGBAInplaceFunc(width, height, stride, data, bgraToRgbaInPlaceSSE3)
	}
}

func BenchmarkConvertBGRAtoRGBA_Fallback_InPlace(b *testing.B) {
	width, height := 1920, 1080
	stride := 4 * width
	data := make([]byte, stride*height)

	for i := 0; i < len(data); i += 4 {
		data[i] = 0xFF
		data[i+1] = 0x00
		data[i+2] = 0x00
		data[i+3] = 0xFF
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		convertBGRAtoRGBAInplaceFunc(width, height, stride, data, bgraToRgbaInPlaceFallback)
	}
}
