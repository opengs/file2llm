package ocr

import (
	"context"
	"io"
)

type ProviderName string

// Handles progress of the OCR
type OCRProgress interface {
	// Receives updates with % completion from 0 to 100.
	// If noone reads from chanel, OCR is not blocked.
	// Chanel may not contain latest information if it is not readed fast.
	CompletionUpdates() chan uint8
	// Contains actual completion progress in % from 0 to 100.
	Completion() uint8
	// Wait until operation is fully completed and get final text.
	Text() (string, error)
}

// Provides OCR functionality
type Provider interface {
	// Get text from image. Thread safe
	OCR(ctx context.Context, image io.Reader) (string, error)
	// Starts OCR in the background and returns hander to check progress updates.
	OCRWithProgress(ctx context.Context, image io.Reader) OCRProgress
	// Check if this provider supports specific mime type
	IsMimeTypeSupported(mimeType string) bool
}
