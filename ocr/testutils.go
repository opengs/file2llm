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

type TestingProviderConfig func()

func NewTestingOCRProvider(t *testing.T, config ...TesseractConfig) Provider {
	tempDir, err := os.MkdirTemp("", "file2llmtest_")
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}

	cfg := DefaultTesseractConfig()
	if len(config) > 0 {
		cfg = config[0]
	}
	tesseract := NewTesseract(cfg)
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
