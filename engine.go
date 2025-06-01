package file2llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/opengs/file2llm/chunker"
	"github.com/opengs/file2llm/embedder"
	"github.com/opengs/file2llm/parser"
	"github.com/opengs/file2llm/source"
	"github.com/opengs/file2llm/storage"
)

type Config struct {
	// Number of simultaniously processing sources
	Parallelism uint32
}

type Engine struct {
	version  storage.ProcessorVersion
	sources  []source.Source
	parser   parser.Parser
	chunker  chunker.Chunker
	embedder embedder.Embedder
	storage  storage.Storage
}

func (e *Engine) Process(ctx context.Context) error {
	for _, source := range e.sources {
		sourceIterator, err := source.Open()
		if err != nil {
			return errors.Join(errors.New("failed to open source"), err)
		}

		err = e.processSource(ctx, source, sourceIterator)
		sourceIterator.Close()

		if err != nil {
			return errors.Join(errors.New("failed to process source"), err)
		}
	}

	return nil
}

func (e *Engine) processSource(ctx context.Context, sourceInfo source.Source, sourceIterator source.Iterator) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		f, err := sourceIterator.Next(ctx)
		if err != nil {
			if err == io.EOF {
				break
			}

			return errors.Join(errors.New("error while iterating over source files"), err)
		}

		err = e.processFile(ctx, sourceInfo, f)
		if err != nil {
			err = errors.Join(errors.New("failed to process file"), err)
		}

		if closeErr := f.Close(); closeErr != nil {
			return errors.Join(errors.New("error during closing processed file"), closeErr, err)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) processFile(ctx context.Context, sourceInfo source.Source, f source.FileHandler) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	fileInfo, newFileCreated, err := e.storage.GetOrCreateFile(ctx, storage.SourceUUID(sourceInfo.UUID()), f.Path(), f.Etag(), e.version)
	if err != nil {
		return errors.Join(errors.New("error during file creation in the storage"), err)
	}

	mustEmbed := newFileCreated
	if !mustEmbed {
		mustEmbed = mustEmbed || (fileInfo.ProcessingFinished == nil && fileInfo.CreatedAt.Before(time.Now().Add(-time.Minute*30)))
		mustEmbed = mustEmbed || (fileInfo.ETag != f.Etag())
		mustEmbed = mustEmbed || (fileInfo.ProcessorVersion.EmbeddingsModel != e.version.EmbeddingsModel)
		mustEmbed = mustEmbed || (fileInfo.ProcessorVersion.Major != e.version.Major)
		mustEmbed = mustEmbed || (fileInfo.ProcessorVersion.Minor != e.version.Minor)
		if mustEmbed {
			if err := e.storage.DeleteFile(ctx, storage.SourceUUID(sourceInfo.UUID()), fileInfo.UUID); err != nil {
				if errors.Is(err, storage.ErrFileDoesntExist) {
					// Someone else stoled our work
					return nil
				}

				return errors.Join(errors.New("failed to delete old file before reembeding"), err)
			}

			fileInfo, newFileCreated, err = e.storage.GetOrCreateFile(ctx, storage.SourceUUID(sourceInfo.UUID()), f.Path(), f.Etag(), e.version)
			if err != nil {
				return errors.Join(errors.New("error during reembeded file creation"), err)
			}
			if !newFileCreated {
				// Someone else stoled our work
				return nil
			}
		}
	}

	if !mustEmbed {
		return nil
	}

	var processingUUID = fmt.Sprintf("%d", rand.Int63())
	if err := sourceInfo.NotifyFileProcessingStarted(ctx, source.FileProcessingStartedEvent{
		UUID:         processingUUID,
		Path:         f.Path(),
		UserMetadata: f.UserMetadata(),
	}); err != nil {
		return errors.Join(errors.New("failed to notify source about start of the file processing"), err)
	}

	openedFilePathes := make(map[string]*storage.File)
	openedFilePathes[f.Path()] = fileInfo
	defer func() {
		for _, openedFile := range openedFilePathes {
			e.storage.DeleteFile(ctx, storage.SourceUUID(sourceInfo.UUID()), openedFile.UUID) // Try to delete unfinished files
		}
	}()

	var sendProcessinRunningEventError error
	sendProcessinRunningEventErrorLock := sync.Mutex{}

	fileParseStream := e.parser.ParseStream(ctx, f, f.Path())
	fileParseRestream := make(chan parser.StreamResult, 1)
	go func() {
		defer close(fileParseRestream)
		for parseResult := range fileParseStream {
			if parseResult.Path() == f.Path() && parseResult.Stage() == parser.ProgressUpdate {
				if err := sourceInfo.NotifyFileProcessingRunning(ctx, source.FileProcessingRunningEvent{
					UUID:         processingUUID,
					Path:         f.Path(),
					UserMetadata: f.UserMetadata(),
					Progress:     parseResult.Progress(),
				}); err != nil {
					sendProcessinRunningEventErrorLock.Lock()
					sendProcessinRunningEventError = err
					sendProcessinRunningEventErrorLock.Unlock()
					return
				}
			}

			select {
			case fileParseRestream <- parseResult:
			case <-ctx.Done():
				return
			}
		}
	}()
	chunkStream := e.chunker.GenerateChunks(ctx, fileParseRestream)
	for chunk := range chunkStream {
		if chunk.Start != nil && chunk.Start.FilePath != f.Path() {
			// Handle inner files.
		}

		if chunk.Data != nil {
			relatedFileInfo := openedFilePathes[chunk.Data.FilePath]

			embeddings, err := e.embedder.GenerateEmbeddings(ctx, chunk.Data.Data)
			if err != nil {
				err = errors.Join(errors.New("error while generating embeddings"), err)
				if chunk.End.FilePath == f.Path() {
					if eventErr := sourceInfo.NotifyFileProcessingDone(ctx, source.FileProcessingDoneEvent{
						UUID:         processingUUID,
						Path:         f.Path(),
						UserMetadata: f.UserMetadata(),
						Reason:       source.FileProcessingAborted,
						Error:        chunk.End.Error,
					}); eventErr != nil {
						return errors.Join(errors.New("failed to notify source about end of the file processing"), eventErr, err)
					}
				}
				return err
			}
			if err := e.storage.PutEmbedding(ctx, storage.SourceUUID(sourceInfo.UUID()), relatedFileInfo.UUID, chunk.Data.Data, embeddings); err != nil {
				err = errors.Join(errors.New("failed to put embeddings in the storage"), err)
				if chunk.End.FilePath == f.Path() {
					if eventErr := sourceInfo.NotifyFileProcessingDone(ctx, source.FileProcessingDoneEvent{
						UUID:         processingUUID,
						Path:         f.Path(),
						UserMetadata: f.UserMetadata(),
						Reason:       source.FileProcessingAborted,
						Error:        chunk.End.Error,
					}); eventErr != nil {
						return errors.Join(errors.New("failed to notify source about end of the file processing"), eventErr, err)
					}
				}
				return err
			}
		}

		if chunk.End != nil {
			relatedFileInfo := openedFilePathes[chunk.Data.FilePath]

			var errorString string
			if chunk.End.Error != nil {
				errorString = chunk.End.Error.Error()
			}
			if err := e.storage.FinishFileProcessing(ctx, storage.SourceUUID(sourceInfo.UUID()), relatedFileInfo.UUID, chunk.End.Error == nil, errorString, nil); err != nil {
				err = errors.Join(errors.New("failed to finalize file processing in storage"), err)
				if chunk.End.FilePath == f.Path() {
					if eventErr := sourceInfo.NotifyFileProcessingDone(ctx, source.FileProcessingDoneEvent{
						UUID:         processingUUID,
						Path:         f.Path(),
						UserMetadata: f.UserMetadata(),
						Reason:       source.FileProcessingAborted,
						Error:        chunk.End.Error,
					}); eventErr != nil {
						return errors.Join(errors.New("failed to notify source about end of the file processing"), eventErr, err)
					}
				}
				return err
			}

			delete(openedFilePathes, chunk.Data.FilePath)

			if chunk.End.FilePath == f.Path() {
				reason := source.FileProcessingOk
				if chunk.End.Error != nil {
					reason = source.FileProcessingError
				}
				if err := sourceInfo.NotifyFileProcessingDone(ctx, source.FileProcessingDoneEvent{
					UUID:         processingUUID,
					Path:         f.Path(),
					UserMetadata: f.UserMetadata(),
					Reason:       reason,
					Error:        chunk.End.Error,
				}); err != nil {
					return errors.Join(errors.New("failed to notify source about end of the file processing"), err)
				}
			}
		}
	}

	sendProcessinRunningEventErrorLock.Lock()
	defer sendProcessinRunningEventErrorLock.Unlock()
	if sendProcessinRunningEventError != nil {
		if err := sourceInfo.NotifyFileProcessingDone(ctx, source.FileProcessingDoneEvent{
			UUID:         processingUUID,
			Path:         f.Path(),
			UserMetadata: f.UserMetadata(),
			Reason:       source.FileProcessingAborted,
			Error:        sendProcessinRunningEventError,
		}); err != nil {
			return errors.Join(errors.New("failed to notify source about end of the file processing"), err, sendProcessinRunningEventError)
		}
		return sendProcessinRunningEventError
	}

	return nil
}
