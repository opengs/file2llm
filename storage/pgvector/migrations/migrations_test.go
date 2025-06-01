package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

func TestPrepareMigrations_AllFilesPlaceholdersReplaced(t *testing.T) {
	schema := "test_schema"
	prefix := "app_"
	vectorDimensions := uint32(768)

	resultFS, err := PrepareMigrations(schema, prefix, vectorDimensions)
	if err != nil {
		t.Fatalf("PrepareMigrations failed: %v", err)
	}

	entries, err := fs.ReadDir(resultFS, ".")
	if err != nil {
		t.Fatalf("failed to read from resulting fs: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("no migration files found in resulting fs")
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		content, err := fs.ReadFile(resultFS, entry.Name())
		if err != nil {
			t.Errorf("failed to read file %s: %v", entry.Name(), err)
			continue
		}

		text := string(content)

		if strings.Contains(text, "SCHEMA_NAME") {
			t.Errorf("%s still contains SCHEMA_NAME placeholder", entry.Name())
		}
		if strings.Contains(text, "DATABASE_PREFIX_") {
			t.Errorf("%s still contains DATABASE_PREFIX_ placeholder", entry.Name())
		}
		if strings.Contains(text, "VECTOR_DIMENSIONS") {
			t.Errorf("%s still contains VECTOR_DIMENSIONS placeholder", entry.Name())
		}
	}
}
