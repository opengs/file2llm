package parser

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/opengs/file2llm/ocr"
	testdata "github.com/opengs/file2llm/test_data"
)

func TestPDF(t *testing.T) {
	ocrProvider := ocr.NewTestingOCRProvider(t)
	pdfParser := NewPDFParser(New(ocrProvider))
	result := pdfParser.Parse(context.Background(), bytes.NewReader(testdata.PDF), "")
	if result.Error() != nil {
		t.Error(result.Error())
		return
	}

	resultString := result.String()
	if !strings.Contains(resultString, "TITLE") || !strings.Contains(resultString, "normal text") || !strings.Contains(resultString, "HELLO") {
		t.Fail()
	}
}

func TestPDFStream(t *testing.T) {
	ocrProvider := ocr.NewTestingOCRProvider(t)
	pdfParser := NewPDFParser(New(ocrProvider))

	hasNewStage := false
	hasCompletedStage := false
	var lastResult StreamResult

	var resultString string
	parseProgress := pdfParser.ParseStream(context.Background(), bytes.NewReader(testdata.PDF), "")
	defer parseProgress.Close()
	for parseProgress.Next(t.Context()) {
		progress := parseProgress.Current()
		resultString += progress.String()
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

	resultString = strings.ToLower(resultString)
	if !strings.Contains(resultString, "title") || !strings.Contains(resultString, "normal text") || !strings.Contains(resultString, "hello") {
		t.Fail()
	}
}
