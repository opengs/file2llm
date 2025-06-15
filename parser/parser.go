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
	// Parse file. Thread safe. Use path to track subfiles or use file name as hint for mime type detection.
	Parse(ctx context.Context, file io.Reader, path string) Result
	// Parse file. Thread safe. Use path to track subfiles or use file name as hint for mime type detection. Return chanel that streams results.
	ParseStream(ctx context.Context, file io.Reader, path string) StreamResultIterator
}

// Parsing result
type Result interface {
	// Get full path to the file
	Path() string
	// Convert entire result to LLM readable string
	String() string
	// Not empty if there where error
	Error() error
	// Parsed subfiles. For example files inside archives
	Subfiles() []Result
}

type StreamResultIterator interface {
	// Block until next stream result available or context is done. If no result available, returns false.
	Next(ctx context.Context) bool
	// Return current stream result
	Current() StreamResult
	// Free all the associated resources
	Close()
}

type ParseProgressStage string

const ProgressNew ParseProgressStage = "NEW"

// Indicates that
const ProgressUpdate ParseProgressStage = "UPDATE"

// Raises on the end of file parsing
const ProgressCompleted ParseProgressStage = "COMPLETED"

type StreamResult interface {
	// Get full path to the file
	Path() string
	// Current file processing progress
	Stage() ParseProgressStage
	// Progress in percents from 0 to 100
	Progress() uint8
	// Underlying result
	SubResult() StreamResult
	// Convert entire result to LLM readable string
	String() string
	// Not empty if there where error
	Error() error
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
			NewRAWBGRAParser(ocrProvider),
		)
	}

	composite.AddParsers(NewPDFParser(composite))
	composite.AddParsers(NewTARParser(composite))
	composite.AddParsers(NewEMLParser(composite))
	return composite
}
