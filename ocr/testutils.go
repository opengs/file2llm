//go:build test

package ocr

import (
	"os"
	"sync"
	"testing"
)

var testingProvider Provider
var testingProviderInitError error
var testingProviderInit sync.Once

func NewTestingOCRProvider(t *testing.T) Provider {
	tempDir, err := os.MkdirTemp("", "file2llmtest_")
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}

	config := DefaultTesseractConfig()
	config.Variables = map[string]string{
		"load_system_dawg":  "0",
		"load_freq_dawg":    "0",
		"load_punc_dawg":    "0",
		"load_number_dawg":  "0",
		"load_unambig_dawg": "0",
		"load_bigram_dawg":  "0",
	}
	tesseract := NewTesseract(config)
	if err := tesseract.Init(); err != nil {
		os.RemoveAll(tempDir)
		t.Log(err.Error())
		t.FailNow()
	}

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
		tesseract.Destroy()
	})

	return tesseract
}
