package parser

import (
	"bytes"
	"strings"
	"testing"

	"github.com/opengs/file2llm/ocr"
	testdata "github.com/opengs/file2llm/test_data"
)

func TestBMPConverting(t *testing.T) {
	cfg := ocr.DefaultTesseractConfig()
	cfg.SupportedImageFormats = []string{"image/png"}
	ocrProvider := ocr.NewTestingOCRProvider(t, cfg)
	bmpParser := NewBMPParser(ocrProvider)
	result := bmpParser.Parse(t.Context(), bytes.NewReader(testdata.BMP), "")
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

func TestBMPStreamConverting(t *testing.T) {
	cfg := ocr.DefaultTesseractConfig()
	cfg.SupportedImageFormats = []string{"image/png"}
	ocrProvider := ocr.NewTestingOCRProvider(t, cfg)
	bmpParser := NewBMPParser(ocrProvider)

	hasNewStage := false
	hasCompletedStage := false
	var lastResult StreamResult

	parseProgress := bmpParser.ParseStream(t.Context(), bytes.NewReader(testdata.BMP), "")
	defer parseProgress.Close()
	for parseProgress.Next(t.Context()) {
		progress := parseProgress.Current()
		hasNewStage = hasNewStage || (progress.Stage() == ProgressNew)
		hasCompletedStage = hasCompletedStage || (progress.Stage() == ProgressCompleted)
		lastResult = progress
	}
	if !hasNewStage || !hasCompletedStage {
		t.FailNow()
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
