package pgvector

import (
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/opengs/file2llm/storage/testlib"
)

func randSchemaName(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz"

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func getTestingStorage(t *testing.T, options ...PGVectorOption) *PGVectorStorage {
	dbURL := os.Getenv("TEST_STORAGE_PGVECTOR_DBURL")
	if dbURL == "" {
		t.Skip("TEST_STORAGE_PGVECTOR_DBURL is not configured")
	}

	cfg, err := pgx.ParseConfig(dbURL)
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}

	db := stdlib.OpenDB(*cfg)
	t.Cleanup(func() {
		db.Close()
	})

	schemaName := randSchemaName(32)
	if _, err := db.ExecContext(t.Context(), "CREATE SCHEMA "+schemaName); err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
	t.Cleanup(func() {
		db.ExecContext(t.Context(), "DROP SCHEMA "+schemaName)
	})

	options = append([]PGVectorOption{WithDatabaseSchema(schemaName), WithEmbeddingVectorDimensions(768)}, options...)
	storage := NewPGVectorStorage(db, options...)
	t.Cleanup(func() {
		storage.UnInstall(t.Context())
	})

	if err := storage.Install(t.Context()); err != nil {
		t.Error(err.Error())
		t.FailNow()
	}

	return &storage
}

func TestUnInstall(t *testing.T) {
	storage := getTestingStorage(t)
	if err := storage.UnInstall(t.Context()); err != nil {
		t.Error(err.Error())
		t.FailNow()
	}

	query := fmt.Sprintf(`
		WITH objects AS (
			SELECT 'table' AS type, tablename AS name
			FROM pg_tables
			WHERE schemaname = '%s'

			UNION ALL
			SELECT 'view', viewname
			FROM pg_views
			WHERE schemaname = '%s'

			UNION ALL
			SELECT 'materialized_view', matviewname
			FROM pg_matviews
			WHERE schemaname = '%s'

			UNION ALL
			SELECT 'sequence', sequencename
			FROM pg_sequences
			WHERE schemaname = '%s'

			UNION ALL
			SELECT 'type', typname
			FROM pg_type
			WHERE typnamespace = (SELECT oid FROM pg_namespace WHERE nspname = '%s')
			AND typtype IN ('c', 'e') -- composite or enum types
			AND typcategory NOT IN ('A', 'P') -- exclude array and pseudo types

			UNION ALL
			SELECT 'function', proname
			FROM pg_proc
			WHERE pronamespace = (SELECT oid FROM pg_namespace WHERE nspname = '%s')
		)
		SELECT *
		FROM objects;
	`, storage.databaseSchema, storage.databaseSchema, storage.databaseSchema, storage.databaseSchema, storage.databaseSchema, storage.databaseSchema)

	rows, err := storage.db.QueryContext(t.Context(), query)
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
	defer rows.Close()

	for rows.Next() {
		t.Fail()
		var objectType, name string
		if err := rows.Scan(&objectType, &name); err != nil {
			t.Error(err.Error())
			t.FailNow()
		}

		t.Logf("Type: %s. Object: %s", objectType, name)
	}

	if rows.Err() != nil {
		t.Error(rows.Err().Error())
		t.FailNow()
	}
}

func TestStorageNotPartitioned(t *testing.T) {
	testlib.TestStorage(t, getTestingStorage(t, WithPartitionsEnabled(false)), 768)
}

func TestStoragePartitioned(t *testing.T) {
	testlib.TestStorage(t, getTestingStorage(t, WithPartitionsEnabled(true)), 768)
}
