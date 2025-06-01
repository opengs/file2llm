//go:build !amd64

package lib

var useAVX2 bool = false

func norm_AVX2_F32(x []float32) float32 {
	panic("not implemented")
}

func divNumber_AVX2_F32(x []float32, a float32) {
	panic("not implemented")
}
