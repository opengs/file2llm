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
	result := webpParser.Parse(context.Background(), bytes.NewReader(testdata.WEBP))
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
