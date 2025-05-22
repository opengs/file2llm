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
	ocrProvider := ocr.NewTestingOCRProvider(t)
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
	ocrProvider := ocr.NewTestingOCRProvider(t)
	webpParser := NewWebPParser(ocrProvider)

	hasNewStage := false
	hasCompletedStage := false
	var lastResult StreamResult

	parseProgress := webpParser.ParseStream(context.Background(), bytes.NewReader(testdata.WEBP), "")
	for progress := range parseProgress {
		hasNewStage = hasNewStage || (progress.Stage() == ProgressNew)
		hasCompletedStage = hasCompletedStage || (progress.Stage() == ProgressCompleted)
		lastResult = progress
	}
	if !hasNewStage || !hasCompletedStage {
		t.Fail()
	}

	resultString := lastResult.String()
	resultString = strings.ToLower(resultString)
	if !strings.Contains(resultString, "hello") {
		t.Fail()
	}
}
