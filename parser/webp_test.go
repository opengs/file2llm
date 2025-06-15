package parser

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/opengs/file2llm/ocr"
	testdata "github.com/opengs/file2llm/test_data"
)

func TestWEBP(t *testing.T) {
	cfg := ocr.DefaultTesseractConfig()
	cfg.SupportedImageFormats = []string{"image/webp"}
	ocrProvider := ocr.NewTestingOCRProvider(t, cfg)
	webpParser := NewWebPParser(ocrProvider)
	result := webpParser.Parse(context.Background(), bytes.NewReader(testdata.WEBP), "")
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

func TestWEBPConverting(t *testing.T) {
	cfg := ocr.DefaultTesseractConfig()
	cfg.SupportedImageFormats = []string{"image/png"}
	ocrProvider := ocr.NewTestingOCRProvider(t, cfg)
	webpParser := NewWebPParser(ocrProvider)
	result := webpParser.Parse(context.Background(), bytes.NewReader(testdata.WEBP), "")
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

func TestWEBPStream(t *testing.T) {
	cfg := ocr.DefaultTesseractConfig()
	cfg.SupportedImageFormats = []string{"image/webp"}
	ocrProvider := ocr.NewTestingOCRProvider(t, cfg)
	webpParser := NewWebPParser(ocrProvider)

	hasNewStage := false
	hasCompletedStage := false
	var lastResult StreamResult

	parseProgress := webpParser.ParseStream(context.Background(), bytes.NewReader(testdata.WEBP), "")
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

func TestWEBPStreamConverting(t *testing.T) {
	cfg := ocr.DefaultTesseractConfig()
	cfg.SupportedImageFormats = []string{"image/png"}
	ocrProvider := ocr.NewTestingOCRProvider(t, cfg)
	webpParser := NewWebPParser(ocrProvider)

	hasNewStage := false
	hasCompletedStage := false
	var lastResult StreamResult

	parseProgress := webpParser.ParseStream(context.Background(), bytes.NewReader(testdata.WEBP), "")
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
