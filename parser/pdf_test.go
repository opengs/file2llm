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
