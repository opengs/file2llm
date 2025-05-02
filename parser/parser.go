package parser

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/gabriel-vasile/mimetype"
	"github.com/opengs/file2llm/ocr"
)

var ErrBadFile = errors.New("bad file or corrupted")
var ErrParserDisabled = errors.New("parser disabled")

type ErrMimeTypeNotSupported struct {
	MimeType *mimetype.MIME
}

func (e *ErrMimeTypeNotSupported) Error() string {
	return fmt.Sprintf("mime type of the file is not supported: %s", e.MimeType)
}

type Parser interface {
	// Returns list of supported mime types by this parser
	SupportedMimeTypes() []string
	// Parse file. Thread safe
	Parse(ctx context.Context, file io.Reader) Result
}

// Parsing result
type Result interface {
	// Convert entire result to LLM readable string
	String() string
	// Not empty if there where error
	Error() error
	// Parsed subcomponents. For example images in the PDF or files inside archives
	Componets() []Result
}

// Build parser with all possible file types included
func New(ocrProvider ocr.Provider) Parser {
	composite := NewCompositeParser()
	if ocrProvider != nil {
		composite.AddParsers(
			NewPNGParser(ocrProvider),
			NewJPEGParser(ocrProvider),
			NewBMPParser(ocrProvider),
			NewGIFParser(ocrProvider),
			NewTiffParser(ocrProvider),
			NewWebPParser(ocrProvider),
		)
	}

	composite.AddParsers(NewPDFParser(composite))
	return composite
}
