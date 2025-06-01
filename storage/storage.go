package storage

import (
	"context"
	"errors"
	"time"
)

type ProcessorVersion struct {
	// If changes, files with older version must be reparsed and reembedded and cant be used in queries.
	Major int `json:"major" db:"major"`
	// If changes, files with older version must be reparsed and reembedded but still can be used in queries.
	Minor int `json:"minor" db:"minor"`
	// If changes, files should be reparsed and  reembedded but its totally fine to use them.
	Patch int `json:"patch" db:"patch"`

	// Model used to generate embeddings. If changes, file must be reembedded and cant be used.
	EmbeddingsModel string `json:"model" db:"model"`
}

type SourceUUID string

type DataSource struct {
	UUID SourceUUID `json:"uuid"`
}

type DataSourceStatistics struct {
	Files            uint64 `json:"files"`
	FilesProcessed   uint64 `json:"filesProcessed"`
	FilesParseErrors uint64 `json:"filesErrors"`
}

type FileUUID string
type File struct {
	Source DataSource `json:"source"`
	UUID   FileUUID   `json:"string"`
	ETag   string     `json:"etag"`
	Path   string     `json:"path"`

	Parsed           bool    `json:"parsed"`
	ParseError       *string `json:"parseError"`
	ParsePartsErrors string  `json:"parsePartsErrors"`

	// Timestamp when this file was first founded and created
	CreatedAt time.Time `json:"createdAt"`
	// File processor version
	ProcessorVersion ProcessorVersion `json:"processorVersion"`
	// Indicates when processing of the file is finished
	ProcessingFinished *time.Time `json:"processingFinished"`
}

type Embedding struct {
	File   File      `json:"file"`
	Chunk  string    `json:"chunk"`
	Vector []float32 `json:"vector"`
}

var ErrDataSourceDoesntExist = errors.New("data source doest not exist in storage")
var ErrFileDoesntExist = errors.New("file does not exist in storage data source")

type Storage interface {
	GetOrCreateSource(ctx context.Context, source SourceUUID) (*DataSource, error)
	// Deletes source, all its files and embeddings.
	DeleteSource(ctx context.Context, source SourceUUID) error

	// Searches for file in the specified source using provided path. If file doesnt exist - creates new one and returns it. New file will have `EmbeddingFinished` set to nil. Returns `true` if file was created during the operation
	GetOrCreateFile(ctx context.Context, source SourceUUID, path string, eTag string, processorVersion ProcessorVersion) (*File, bool, error)
	// Deletes file and all its embeddings. Returns file before deletion
	DeleteFile(ctx context.Context, source SourceUUID, file FileUUID) error
	// Updated file information and sets `ProcessingFinished` to current time
	FinishFileProcessing(ctx context.Context, source SourceUUID, file FileUUID, parsed bool, parseError string, parsePartsErrors []string) error
	// Stores embedding
	PutEmbedding(ctx context.Context, source SourceUUID, file FileUUID, chunk string, embeddingVector []float32) error

	// Performs vector similarity search and returns nearest vectors. If sources array is empty, searches in all available sources
	SearchSimilarEmbedddings(ctx context.Context, embeddingVector []float32, sources []SourceUUID, limit uint32) ([]Embedding, error)
}
