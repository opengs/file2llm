package file2llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
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

		if _, err := e.storage.GetOrCreateSource(ctx, storage.SourceUUID(sourceInfo.UUID())); err != nil {
			return errors.Join(errors.New("failed to ensure that source exists in the storage"), err)
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

	var processingUUID = fmt.Sprintf("%s-%s-%d-%d", sourceInfo.UUID(), f.Path(), time.Now().UnixNano(), rand.Int63())
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

	fileParseStream := e.parser.ParseStream(ctx, f, f.Path())
	defer fileParseStream.Close()

	chunkStream := e.chunker.GenerateChunks(ctx, fileParseStream)
	for chunkStream.Next(ctx) {
		chunk := chunkStream.Current()

		if chunk.Start != nil && chunk.Start.FilePath != f.Path() {
			// Handle inner files.
			// openedFilePathes[chunk.Start.FilePath] = chunk.Start
		}

		if chunk.Data != nil {
			relatedFileInfo, ok := openedFilePathes[chunk.Data.FilePath]
			if !ok {
				// most probably chunk comes from embedded file (like image in PDF) not inner file (like attachment in email).
				continue
			}

			embeddings, err := e.embedder.GenerateEmbeddings(ctx, chunk.Data.Data)
			if err != nil {
				err = errors.Join(errors.New("error while generating embeddings"), err)
				if chunk.Data.FilePath == f.Path() {
					if eventErr := sourceInfo.NotifyFileProcessingDone(ctx, source.FileProcessingDoneEvent{
						UUID:         processingUUID,
						Path:         f.Path(),
						UserMetadata: f.UserMetadata(),
						Reason:       source.FileProcessingAborted,
						Error:        err,
					}); eventErr != nil {
						return errors.Join(errors.New("failed to notify source about end of the file processing"), eventErr, err)
					}
				}
				return err
			}
			if err := e.storage.PutEmbedding(ctx, storage.SourceUUID(sourceInfo.UUID()), relatedFileInfo.UUID, chunk.Data.Data, embeddings); err != nil {
				err = errors.Join(errors.New("failed to put embeddings in the storage"), err)
				if chunk.Data.FilePath == f.Path() {
					if eventErr := sourceInfo.NotifyFileProcessingDone(ctx, source.FileProcessingDoneEvent{
						UUID:         processingUUID,
						Path:         f.Path(),
						UserMetadata: f.UserMetadata(),
						Reason:       source.FileProcessingAborted,
						Error:        err,
					}); eventErr != nil {
						return errors.Join(errors.New("failed to notify source about end of the file processing"), eventErr, err)
					}
				}
				return err
			}
		}

		if chunk.End != nil {
			relatedFileInfo, ok := openedFilePathes[chunk.End.FilePath]
			if !ok {
				// most probably chunk comes from embedded file (like image in PDF) not inner file (like attachment in email).
				continue
			}

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

	return nil
}
