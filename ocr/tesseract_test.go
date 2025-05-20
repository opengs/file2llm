package ocr

import (
	"bytes"
	"context"
	"strings"
	"testing"

	testdata "github.com/opengs/file2llm/test_data"
)

func TestTesseract(t *testing.T) {
	cfg := DefaultTesseractConfig()
	ocr := NewTesseract(cfg)
	if err := ocr.Init(); err != nil {
		t.Error(err.Error())
		return
	}
	defer ocr.Destroy()
	text, err := ocr.OCR(context.Background(), bytes.NewBuffer(testdata.PNG))
	if err != nil {
		t.Error(err.Error())
		return
	}

	text = strings.ToLower(text)
	if !strings.Contains(text, "hello") {
		t.Fail()
	}
}

func TestTesseractProgress(t *testing.T) {
	cfg := DefaultTesseractConfig()
	ocr := NewTesseract(cfg)
	if err := ocr.Init(); err != nil {
		t.Error(err.Error())
		return
	}
	defer ocr.Destroy()
	text, err := ocr.OCRWithProgress(context.Background(), bytes.NewBuffer(testdata.PNG)).Text()
	if err != nil {
		t.Error(err.Error())
		return
	}

	text = strings.ToLower(text)
	if !strings.Contains(text, "hello") {
		t.Fail()
	}
}
