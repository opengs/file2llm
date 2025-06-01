package lib

import (
	"runtime"

	"golang.org/x/sys/cpu"
)

var useAVX2 bool = cpu.X86.HasAVX2 && cpu.X86.HasFMA && runtime.GOOS != "darwin"

//go:noescape
func norm_AVX2_F32(x []float32) float32

//go:noescape
func divNumber_AVX2_F32(x []float32, a float32)
