package parser

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/opengs/file2llm/ocr"
	testdata "github.com/opengs/file2llm/test_data"
)

func TestGIF(t *testing.T) {
	cfg := ocr.DefaultTesseractConfig()
	cfg.SupportedImageFormats = []string{"image/gif"}
	ocrProvider := ocr.NewTestingOCRProvider(t, cfg)
	gifParser := NewGIFParser(ocrProvider)
	result := gifParser.Parse(context.Background(), bytes.NewReader(testdata.GIF), "")
	if result.Error() != nil {
		t.Error(result.Error())
		return
	}

	resultString := result.String()
	resultString = strings.ToLower(resultString)
	if !strings.Contains(resultString, "hello") {
		t.Fail()
	}
}

func TestGIFConverting(t *testing.T) {
	cfg := ocr.DefaultTesseractConfig()
	cfg.SupportedImageFormats = []string{"image/png"}
	ocrProvider := ocr.NewTestingOCRProvider(t, cfg)
	gifParser := NewGIFParser(ocrProvider)
	result := gifParser.Parse(context.Background(), bytes.NewReader(testdata.GIF), "")
	if result.Error() != nil {
		t.Error(result.Error())
		return
	}

	resultString := result.String()
	resultString = strings.ToLower(resultString)
	if !strings.Contains(resultString, "hello") {
		t.Fail()
	}
}

func TestGIFStream(t *testing.T) {
	cfg := ocr.DefaultTesseractConfig()
	cfg.SupportedImageFormats = []string{"image/gif"}
	ocrProvider := ocr.NewTestingOCRProvider(t, cfg)
	gifParser := NewGIFParser(ocrProvider)

	hasNewStage := false
	hasCompletedStage := false
	var lastResult StreamResult

	parseProgress := gifParser.ParseStream(context.Background(), bytes.NewReader(testdata.GIF), "")
	defer parseProgress.Close()
	for parseProgress.Next(t.Context()) {
		progress := parseProgress.Current()
		hasNewStage = hasNewStage || (progress.Stage() == ProgressNew)
		hasCompletedStage = hasCompletedStage || (progress.Stage() == ProgressCompleted)
		lastResult = progress
	}
	if !hasNewStage || !hasCompletedStage {
		t.Fail()
	}
	if lastResult.Error() != nil {
		t.Fatal(lastResult.Error())
	}

	resultString := lastResult.String()
	resultString = strings.ToLower(resultString)
	if !strings.Contains(resultString, "hello") {
		t.Fail()
	}
}

func TestGIFStreamConverting(t *testing.T) {
	cfg := ocr.DefaultTesseractConfig()
	cfg.SupportedImageFormats = []string{"image/png"}
	ocrProvider := ocr.NewTestingOCRProvider(t, cfg)
	gifParser := NewGIFParser(ocrProvider)

	hasNewStage := false
	hasCompletedStage := false
	var lastResult StreamResult

	parseProgress := gifParser.ParseStream(context.Background(), bytes.NewReader(testdata.GIF), "")
	defer parseProgress.Close()
	for parseProgress.Next(t.Context()) {
		progress := parseProgress.Current()
		hasNewStage = hasNewStage || (progress.Stage() == ProgressNew)
		hasCompletedStage = hasCompletedStage || (progress.Stage() == ProgressCompleted)
		lastResult = progress
	}
	if !hasNewStage || !hasCompletedStage {
		t.Fail()
	}
	if lastResult.Error() != nil {
		t.Fatal(lastResult.Error())
	}

	resultString := lastResult.String()
	resultString = strings.ToLower(resultString)
	if !strings.Contains(resultString, "hello") {
		t.Fail()
	}
}
