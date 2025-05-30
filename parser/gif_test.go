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
	ocrProvider := ocr.NewTestingOCRProvider(t)
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
