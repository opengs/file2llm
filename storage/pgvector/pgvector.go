package pgvector

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/opengs/file2llm/storage"
	"github.com/opengs/file2llm/storage/pgvector/migrations"
)

type PGVectorStorage struct {
	db *sql.DB

	partitionsEnabled         bool
	embeddingVectorDimensions uint32

	databaseName   string
	databaseSchema string
	databasePrefix string

	sourceTable          string
	fileTable            string
	embeddingTable       string
	processorVersionType string
}

func NewPGVectorStorage(db *sql.DB, options ...PGVectorOption) PGVectorStorage {
	storage := PGVectorStorage{
		db:                        db,
		partitionsEnabled:         false,
		embeddingVectorDimensions: 768,
		databaseName:              "postgress",
		databaseSchema:            "public",
		databasePrefix:            "file2llm_",
	}

	for _, option := range options {
		option(&storage)
	}

	storage.sourceTable = fmt.Sprintf("%s.%ssource", storage.databaseSchema, storage.databasePrefix)
	storage.fileTable = fmt.Sprintf("%s.%sfile", storage.databaseSchema, storage.databasePrefix)
	storage.embeddingTable = fmt.Sprintf("%s.%sembedding", storage.databaseSchema, storage.databasePrefix)
	storage.processorVersionType = fmt.Sprintf("%s.%sprocessor_version", storage.databaseSchema, storage.databasePrefix)

	return storage
}

// Make sure that all the tables are created inside PGVector and its ready to work. You can run this safelly several times.
func (s *PGVectorStorage) Install(ctx context.Context) error {
	migrationFiles, err := migrations.PrepareMigrations(s.databaseSchema, s.databasePrefix, s.embeddingVectorDimensions)
	if err != nil {
		return errors.Join(errors.New("failed to prepare migration files"), err)
	}

	driver, err := postgres.WithInstance(s.db, &postgres.Config{
		SchemaName:      s.databaseSchema,
		MigrationsTable: fmt.Sprintf("%smigrations", s.databasePrefix),
	})
	if err != nil {
		return errors.Join(errors.New("failed to create postgress migration driver"), err)
	}

	migrationsSource, err := iofs.New(migrationFiles, ".")
	if err != nil {
		return errors.Join(errors.New("failed to open postgress migrations source"), err)
	}

	migrator, err := migrate.NewWithInstance("migrations", migrationsSource, s.databaseName, driver)
	if err != nil {
		return errors.Join(errors.New("failed to create migrator"), err)
	}

	if err := migrator.Up(); err != nil {
		return errors.Join(errors.New("error while performing migration on the database"), err)
	}

	return nil
}

// Completelly removes itselve from the database
func (s *PGVectorStorage) UnInstall(ctx context.Context) error {
	migrationFiles, err := migrations.PrepareMigrations(s.databaseSchema, s.databasePrefix, s.embeddingVectorDimensions)
	if err != nil {
		return errors.Join(errors.New("failed to prepare migration files"), err)
	}

	driver, err := postgres.WithInstance(s.db, &postgres.Config{
		SchemaName:      s.databaseSchema,
		MigrationsTable: fmt.Sprintf("%smigrations", s.databasePrefix),
	})
	if err != nil {
		return errors.Join(errors.New("failed to create postgress migration driver"), err)
	}

	migrationsSource, err := iofs.New(migrationFiles, ".")
	if err != nil {
		return errors.Join(errors.New("failed to open postgress migrations source"), err)
	}

	migrator, err := migrate.NewWithInstance("migrations", migrationsSource, s.databaseName, driver)
	if err != nil {
		return errors.Join(errors.New("failed to create migrator"), err)
	}

	if err := migrator.Down(); err != nil {
		return errors.Join(errors.New("error while performing migration on the database"), err)
	}

	if _, err := s.db.Exec("DROP TABLE " + fmt.Sprintf("%s.%smigrations", s.databaseSchema, s.databasePrefix)); err != nil {
		return errors.Join(errors.New("failed to drop migrations table"), err)
	}

	return nil
}

func (s *PGVectorStorage) GetOrCreateSource(ctx context.Context, sourceUUID storage.SourceUUID) (*storage.DataSource, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, errors.Join(errors.New("failed to begin source creation transaction in database"), err)
	}
	defer tx.Rollback()

	query := fmt.Sprintf(`
		INSERT INTO %s (
			uuid
		)
		VALUES ($1)
		ON CONFLICT (uuid) DO UPDATE SET uuid = $1
		RETURNING source_id
	`, s.sourceTable)
	var sourceID int
	if err := tx.QueryRowContext(ctx, query, sourceUUID).Scan(&sourceID); err != nil {
		return nil, errors.Join(errors.New("failed to insert new source to the database"), err)
	}

	if s.partitionsEnabled {
		fileTablePartiotionQuery := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s_%d
			PARTITION OF %s
			FOR VALUES IN (%d)
		`, s.fileTable, sourceID, s.fileTable, sourceID)
		if _, err := tx.ExecContext(ctx, fileTablePartiotionQuery); err != nil {
			return nil, errors.Join(errors.New("failed to create partition for file table in the database"), err)
		}

		embeddingTablePartiotionQuery := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s_%d
			PARTITION OF %s
			FOR VALUES IN (%d)
		`, s.embeddingTable, sourceID, s.embeddingTable, sourceID)
		if _, err := tx.ExecContext(ctx, embeddingTablePartiotionQuery); err != nil {
			return nil, errors.Join(errors.New("failed to create partition for embedding table in the database"), err)
		}

		embeddingTablePartiotionIndexQuery := fmt.Sprintf(`
			CREATE INDEX IF NOT EXISTS idx_%sembedding_%d_vector
			ON %s_%d
			USING hnsw (embedding vector_cosine_ops)
		`, s.databasePrefix, sourceID, s.embeddingTable, sourceID)
		if _, err := tx.ExecContext(ctx, embeddingTablePartiotionIndexQuery); err != nil {
			return nil, errors.Join(errors.New("failed to create index on partition of embedding table in the database"), err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Join(errors.New("failed to commit source creation transaction in the database"), err)
	}

	return &storage.DataSource{
		UUID: sourceUUID,
	}, nil
}

func (s *PGVectorStorage) DeleteSource(ctx context.Context, sourceUUID storage.SourceUUID) error {
	query := fmt.Sprintf(`
		DELETE FROM %s
		WHERE uuid = $1
		RETURNING uuid
	`, s.sourceTable)
	var returnedUUID storage.SourceUUID
	if err := s.db.QueryRowContext(ctx, query, sourceUUID).Scan(&returnedUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrDataSourceDoesntExist
		}

		return errors.Join(errors.New("failed to delete data source from the database"), err)
	}

	return nil
}

func (s *PGVectorStorage) GetOrCreateFile(ctx context.Context, sourceUUID storage.SourceUUID, path string, eTag string, processorVersion storage.ProcessorVersion) (*storage.File, bool, error) {
	query := fmt.Sprintf(`
		WITH source_lookup AS (
			SELECT source_id
			FROM %s
			WHERE uuid = $1
		), ins AS (
			INSERT INTO %s (
				source_id,
				path,
				etag,
				processor_version
			)
			SELECT source_lookup.source_id, $2, $3, ($4, $5, $6, $7)::%s FROM source_lookup
			ON CONFLICT(source_id, path) DO NOTHING
			RETURNING file_id, etag, parsed, parse_error, parse_parts_errors, created_at, (processor_version).major, (processor_version).minor, (processor_version).patch, (processor_version).model, processing_finished, true as inserted
		)
		SELECT * FROM ins
		UNION ALL
		SELECT file_id, etag, parsed, parse_error, parse_parts_errors, created_at, (processor_version).major, (processor_version).minor, (processor_version).patch, (processor_version).model, processing_finished, false as inserted
		FROM %s f
		JOIN %s s ON f.source_id = s.source_id
		WHERE NOT EXISTS (SELECT 1 FROM ins) AND s.uuid = $1 AND path = $2;
	`, s.sourceTable, s.fileTable, s.processorVersionType, s.fileTable, s.sourceTable)
	var fileId uint64
	var currentETag string
	var parsed bool
	var parseError *string
	var parsePartsErrors string
	var createdAT time.Time
	var newProcessorVersion storage.ProcessorVersion
	var processingFinished *time.Time
	var inserted bool
	if err := s.db.QueryRowContext(ctx, query, sourceUUID, path, eTag, processorVersion.Major, processorVersion.Minor, processorVersion.Patch, processorVersion.EmbeddingsModel).Scan(&fileId, &currentETag, &parsed, &parseError, &parsePartsErrors, &createdAT, &newProcessorVersion.Major, &newProcessorVersion.Minor, &newProcessorVersion.Patch, &newProcessorVersion.EmbeddingsModel, &processingFinished, &inserted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, storage.ErrDataSourceDoesntExist
		}

		return nil, false, errors.Join(errors.New("failed to get or create file in the database"), err)
	}

	return &storage.File{
		Source: storage.DataSource{
			UUID: sourceUUID,
		},
		UUID:               storage.FileUUID(fmt.Sprintf("%d", fileId)),
		ETag:               currentETag,
		Path:               path,
		Parsed:             parsed,
		ParseError:         parseError,
		ParsePartsErrors:   parsePartsErrors,
		CreatedAt:          createdAT,
		ProcessorVersion:   newProcessorVersion,
		ProcessingFinished: processingFinished,
	}, inserted, nil
}

func (s *PGVectorStorage) DeleteFile(ctx context.Context, source storage.SourceUUID, file storage.FileUUID) error {
	fileID, err := strconv.Atoi(string(file))
	if err != nil {
		return storage.ErrFileDoesntExist
	}

	query := fmt.Sprintf(`
		DELETE FROM %s
		USING %s
		WHERE %s.uuid = $1
			AND %s.source_id = %s.source_id
			AND %s.file_id = $2
		RETURNING %s.file_id
	`, s.fileTable, s.sourceTable, s.sourceTable, s.fileTable, s.sourceTable, s.fileTable, s.fileTable)
	var returnedFileID uint64
	if err := s.db.QueryRowContext(ctx, query, source, fileID).Scan(&returnedFileID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrFileDoesntExist
		}

		return errors.Join(errors.New("failed to delete data file from the database"), err)
	}

	return nil
}

func (s *PGVectorStorage) FinishFileProcessing(ctx context.Context, source storage.SourceUUID, file storage.FileUUID, parsed bool, parseError string, parsePartsErrors []string) error {
	fileID, err := strconv.Atoi(string(file))
	if err != nil {
		return storage.ErrFileDoesntExist
	}

	if len(parsePartsErrors) == 0 {
		parsePartsErrors = make([]string, 0)
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET
			processing_finished = NOW(),
			parsed = $3,
			parse_error = $4,
			parse_parts_errors = $5
		FROM %s
		WHERE %s.source_id = %s.source_id
			AND %s.uuid = $1
			AND %s.file_id = $2
		RETURNING %s.source_id
	`, s.fileTable, s.sourceTable, s.fileTable, s.sourceTable, s.sourceTable, s.fileTable, s.sourceTable)
	var sourceId int
	if err := s.db.QueryRowContext(ctx, query, source, fileID, parsed, parseError, parsePartsErrors).Scan(&sourceId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrFileDoesntExist
		}

		return errors.Join(errors.New("failed to update file info in the database"), err)
	}

	return nil
}

func (s *PGVectorStorage) PutEmbedding(ctx context.Context, source storage.SourceUUID, file storage.FileUUID, chunk string, embeddingVector []float32) error {
	fileID, err := strconv.Atoi(string(file))
	if err != nil {
		return storage.ErrFileDoesntExist
	}

	query := fmt.Sprintf(`
		WITH source_cte AS (
			SELECT source_id FROM %s WHERE uuid = $1
		)
		INSERT INTO %s (source_id, file_id, chunk, embedding)
		SELECT source_id, $2, $3, $4 FROM source_cte
	`, s.sourceTable, s.embeddingTable)
	commandTag, err := s.db.Exec(query, source, fileID, chunk, embeddingToPgvectorFormat(embeddingVector))
	if err != nil {
		return errors.Join(errors.New("failed to insert embedding in the database"), err)
	}

	if affected, _ := commandTag.RowsAffected(); affected == 0 {
		return storage.ErrFileDoesntExist
	}

	return nil
}

func (s *PGVectorStorage) SearchSimilarEmbedddings(ctx context.Context, embeddingVector []float32, sources []storage.SourceUUID, limit uint32) ([]storage.Embedding, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if len(sources) == 0 {
		query := fmt.Sprintf(`
			SELECT
				s.uuid,

				f.file_id,
				f.etag,
				f.path,
				f.parsed,
				f.parse_error,
				f.parse_parts_errors,
				f.created_at,
				(f.processor_version).major, (f.processor_version).minor, (f.processor_version).patch, (f.processor_version).model,
				f.processing_finished,

				e.chunk,
				e.embedding
			FROM %s e
			JOIN %s s ON s.source_id = e.source_id
			JOIN %s f ON f.file_id = e.file_id
			ORDER BY e.embedding <=> $1
			LIMIT $2
		`, s.embeddingTable, s.sourceTable, s.fileTable)
		rows, err = s.db.QueryContext(ctx, query, embeddingToPgvectorFormat(embeddingVector), limit)
	} else {
		query := fmt.Sprintf(`
			WITH source_ids AS (
    			SELECT source_id FROM %s WHERE uuid = ANY($3)
			)
			SELECT
				s.uuid,

				f.file_id,
				f.etag,
				f.path,
				f.parsed,
				f.parse_error,
				f.parse_parts_errors,
				f.created_at,
				(f.processor_version).major, (f.processor_version).minor, (f.processor_version).patch, (f.processor_version).model,
				f.processing_finished,

				e.chunk,
				e.embedding
			FROM %s e
			JOIN %s s ON s.source_id = e.source_id
			JOIN %s f ON f.file_id = e.file_id
			JOIN source_ids si ON si.source_id = s.source_id 
			ORDER BY e.embedding <=> $1
			LIMIT $2
		`, s.sourceTable, s.embeddingTable, s.sourceTable, s.fileTable)
		rows, err = s.db.QueryContext(ctx, query, embeddingToPgvectorFormat(embeddingVector), limit, sources)
	}
	if err != nil {
		return nil, errors.Join(errors.New("failed to get embeddings from the database"), err)
	}
	defer rows.Close()

	var embeddings []storage.Embedding
	for rows.Next() {
		var fileID uint64
		var rawEmbeddingString string

		var emb storage.Embedding
		err = rows.Scan(
			&emb.File.Source.UUID,

			&fileID,
			&emb.File.ETag,
			&emb.File.Path,
			&emb.File.Parsed,
			&emb.File.ParseError,
			&emb.File.ParsePartsErrors,
			&emb.File.CreatedAt,
			&emb.File.ProcessorVersion.Major,
			&emb.File.ProcessorVersion.Minor,
			&emb.File.ProcessorVersion.Patch,
			&emb.File.ProcessorVersion.EmbeddingsModel,
			&emb.File.ProcessingFinished,

			&emb.Chunk,
			&rawEmbeddingString,
		)
		if err != nil {
			return nil, errors.Join(errors.New("failed to scan embeddings row from the database"), err)
		}
		emb.File.UUID = storage.FileUUID(fmt.Sprintf("%d", fileID))

		embeddingsVector, err := pgvectorFormatToEmbedding(rawEmbeddingString)
		if err != nil {
			return nil, errors.Join(errors.New("failed to decode embeddings vector received from pgvector"), err)
		}
		emb.Vector = embeddingsVector

		embeddings = append(embeddings, emb)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Join(errors.New("errors while response from the database"), err)
	}

	return embeddings, nil
}
