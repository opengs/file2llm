package parser

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/opengs/file2llm/ocr"
	testdata "github.com/opengs/file2llm/test_data"
)

func TestJPEG(t *testing.T) {
	cfg := ocr.DefaultTesseractConfig()
	cfg.SupportedImageFormats = []string{"image/jpeg"}
	ocrProvider := ocr.NewTestingOCRProvider(t, cfg)
	jpegParser := NewJPEGParser(ocrProvider)
	result := jpegParser.Parse(context.Background(), bytes.NewReader(testdata.JPEG), "")
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

func TestJPEGConverting(t *testing.T) {
	cfg := ocr.DefaultTesseractConfig()
	cfg.SupportedImageFormats = []string{"image/png"}
	ocrProvider := ocr.NewTestingOCRProvider(t, cfg)
	jpegParser := NewJPEGParser(ocrProvider)
	result := jpegParser.Parse(context.Background(), bytes.NewReader(testdata.JPEG), "")
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

func TestJPEGStream(t *testing.T) {
	cfg := ocr.DefaultTesseractConfig()
	cfg.SupportedImageFormats = []string{"image/jpeg"}
	ocrProvider := ocr.NewTestingOCRProvider(t, cfg)
	jpegParser := NewJPEGParser(ocrProvider)

	hasNewStage := false
	hasCompletedStage := false
	var lastResult StreamResult

	parseProgress := jpegParser.ParseStream(context.Background(), bytes.NewReader(testdata.JPEG), "")
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

func TestJPEGStreamConverting(t *testing.T) {
	cfg := ocr.DefaultTesseractConfig()
	cfg.SupportedImageFormats = []string{"image/png"}
	ocrProvider := ocr.NewTestingOCRProvider(t, cfg)
	jpegParser := NewJPEGParser(ocrProvider)

	hasNewStage := false
	hasCompletedStage := false
	var lastResult StreamResult

	parseProgress := jpegParser.ParseStream(context.Background(), bytes.NewReader(testdata.JPEG), "")
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
