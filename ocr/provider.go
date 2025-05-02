package ocr

import "context"

type ProviderName string

// Provides OCR functionality
type Provider interface {
	// Get text from image. Thread safe
	OCR(ctx context.Context, image []byte) (string, error)
	// Check if this provider supports specific mime type
	IsMimeTypeSupported(mimeType string) bool
}
