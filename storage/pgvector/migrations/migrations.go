package migrations

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/psanford/memfs"
)

//go:embed *.sql
var migrations embed.FS

func PrepareMigrations(schema string, prefix string, vectorDimensions uint32) (fs.FS, error) {
	rootFS := memfs.New()

	entries, err := migrations.ReadDir(".")
	if err != nil {
		return nil, errors.Join(errors.New("failed to read migrations directory"), err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		file, err := migrations.Open(entry.Name())
		if err != nil {
			return nil, err
		}
		fileData, err := io.ReadAll(file)
		if err != nil {
			file.Close()
			return nil, err
		}
		file.Close()

		newData := strings.ReplaceAll(string(fileData), "SCHEMA_NAME", schema)
		newData = strings.ReplaceAll(string(newData), "DATABASE_PREFIX_", prefix)
		newData = strings.ReplaceAll(string(newData), "VECTOR_DIMENSIONS", fmt.Sprintf("%d", vectorDimensions))

		if err := rootFS.WriteFile(entry.Name(), []byte(newData), 0755); err != nil {
			return nil, err
		}
	}

	return rootFS, nil
}
