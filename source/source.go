package source

import (
	"context"
	"io"
)

type FileProcessingStartedEvent struct {
	// Unique identifier of the processing. Same for all events during same processing
	UUID string
	// Path to the processed file
	Path string
	// File user metadata
	UserMetadata any
}

type FileProcessingRunningEvent struct {
	// Unique identifier of the processing. Same for all events during same processing
	UUID string
	// Path to the processed file
	Path string
	// File user metadata
	UserMetadata any
	// Progress in percentages from 0 to 100
	Progress uint8
}

type FileProcessingDoneReason string

const FileProcessingOk FileProcessingDoneReason = "OK"
const FileProcessingError FileProcessingDoneReason = "ERROR"
const FileProcessingAborted FileProcessingDoneReason = "ABORTED"

type FileProcessingDoneEvent struct {
	// Unique identifier of the processing. Same for all events during same processing
	UUID string
	// Path to the processed file
	Path string
	// File user metadata
	UserMetadata any
	// Why processing finished
	Reason FileProcessingDoneReason
	// Only valid if reason is ERROR
	Error error
}

// Place where data located
type Source interface {
	UUID() string
	// Open data source for iteration
	Open() (Iterator, error)

	// Notify source that processing of the file is started.
	NotifyFileProcessingStarted(ctx context.Context, event FileProcessingStartedEvent) error
	// Notify source that processing of the file is still running.
	// Notifications may be not synchronized with start and done events but handled one by one and in order. Use event UUID to detect sync problems.
	// Event handling blocks processing and stops it if returns error. Separate done event will be emited if this hanler return error.
	// Running events must be emmited with at most 30 seconds interval.
	NotifyFileProcessingRunning(ctx context.Context, event FileProcessingRunningEvent) error
	// Notify source that processing of the file is finished
	NotifyFileProcessingDone(ctx context.Context, event FileProcessingDoneEvent) error
}

// Opened data source
type Iterator interface {
	io.Closer

	// Get and open next file. Thread safe. If there are no files left, returns [io.EOF] error
	Next(ctx context.Context) (FileHandler, error)
}

type FileHandler interface {
	io.ReadCloser

	// Unique identifier of the file content
	Etag() string
	// Path to the file in the data source
	Path() string

	// Arbitrary metadata created by user that is used inside user application. Will be passed to source suring events.
	UserMetadata() any
}
