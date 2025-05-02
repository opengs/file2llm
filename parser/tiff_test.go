package parser

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/opengs/file2llm/ocr"
	testdata "github.com/opengs/file2llm/test_data"
)

func TestTIFF(t *testing.T) {
	ocrProvider := ocr.NewTestingOCRProvider(t)
	tiffParser := NewTiffParser(ocrProvider)
	result := tiffParser.Parse(context.Background(), bytes.NewReader(testdata.TIFF))
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
