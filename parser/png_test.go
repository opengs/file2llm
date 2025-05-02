package parser

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/opengs/file2llm/ocr"
	testdata "github.com/opengs/file2llm/test_data"
)

func TestPNG(t *testing.T) {
	ocrProvider := ocr.NewTestingOCRProvider(t)
	pngParser := NewPNGParser(ocrProvider)
	result := pngParser.Parse(context.Background(), bytes.NewReader(testdata.PNG))
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
