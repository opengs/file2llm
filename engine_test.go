package file2llm

import (
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	slidechunk "github.com/opengs/file2llm/chunker/slide_chunk"
	"github.com/opengs/file2llm/embedder/ollama"
	"github.com/opengs/file2llm/ocr"
	"github.com/opengs/file2llm/parser"
	"github.com/opengs/file2llm/source"
	"github.com/opengs/file2llm/source/fs"
	"github.com/opengs/file2llm/storage"
	"github.com/opengs/file2llm/storage/pgvector"
	testdata "github.com/opengs/file2llm/test_data"
)

func TestEngineE2E(t *testing.T) {
	ollamaBaseURL := os.Getenv("TEST_EMBEDDER_OLLAMA_BASEURL")
	if ollamaBaseURL == "" {
		t.Skip("TEST_EMBEDDER_OLLAMA_BASEURL is not configured")
	}
	dbURL := os.Getenv("TEST_STORAGE_PGVECTOR_DBURL")
	if dbURL == "" {
		t.Skip("TEST_STORAGE_PGVECTOR_DBURL is not configured")
	}

	eSource := fs.New(testdata.FS, "fs", fmt.Sprintf("testsource-%d", rand.Int63()))

	eOCR := ocr.NewTesseract(ocr.DefaultTesseractConfig())
	if err := eOCR.Init(); err != nil {
		t.Fatal(err.Error())
	}
	defer eOCR.Destroy()
	eParser := parser.New(eOCR)
	eEmbedder := ollama.New("all-minilm", ollama.WithBaseURL(ollamaBaseURL), ollama.WithDimensions(384))
	if err := eEmbedder.PullModel(t.Context()); err != nil {
		t.Fatal(err.Error())
	}
	eChunker := slidechunk.New(512, 128)

	var eStorage storage.Storage
	{
		cfg, err := pgx.ParseConfig(dbURL)
		if err != nil {
			t.Fatal(err.Error())
		}

		db := stdlib.OpenDB(*cfg)
		t.Cleanup(func() {
			db.Close()
		})

		schemaName := fmt.Sprintf("testschema_%d", rand.Int63())
		if _, err := db.ExecContext(t.Context(), "CREATE SCHEMA "+schemaName); err != nil {
			t.Fatal(err.Error())
		}
		t.Cleanup(func() {
			db.ExecContext(t.Context(), "DROP SCHEMA "+schemaName)
		})

		storage := pgvector.NewPGVectorStorage(db, pgvector.WithDatabaseSchema(schemaName), pgvector.WithEmbeddingVectorDimensions(384))
		t.Cleanup(func() {
			storage.UnInstall(t.Context())
		})

		if err := storage.Install(t.Context()); err != nil {
			t.Fatal(err.Error())
		}

		eStorage = &storage
	}

	engine := Engine{
		version:  storage.ProcessorVersion{},
		sources:  []source.Source{eSource},
		parser:   eParser,
		chunker:  eChunker,
		embedder: eEmbedder,
		storage:  eStorage,
	}
	if err := engine.Process(t.Context()); err != nil {
		t.Fatal(err.Error())
	}

	searchVector, err := eEmbedder.GenerateEmbeddings(t.Context(), "Spatial hashing is an efficient approach for performing proximity queries on objects in collision detection,\ncrowd simulations, and navigation in 3D space. It can also be used to enhance other proximity-related tasks,\nparticularly in virtual realities. This paper describes a fast approach for creating a 1D hash table that handles\nproximity maps with fixed-size vectors and pivots. Because it allows for linear memory iteration and quick\nproximity detection, this method is suitable for reaching interactive frame rates with a high number of sim-\nulating objects.")
	if err != nil {
		t.Fatal(err.Error())
	}

	similarEmbeddings, err := eStorage.SearchSimilarEmbedddings(t.Context(), searchVector, nil, 1)
	if err != nil {
		t.Fatal(err.Error())
	}

	if similarEmbeddings[0].Chunk != "" {
		t.Fatal(err.Error())
	}

	t.Fatal(err.Error())
}
