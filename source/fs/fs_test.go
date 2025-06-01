package fs

import (
	"context"
	"io"
	"io/fs"
	"strings"
	"testing"
	"time"

	"testing/fstest"
)

func TestFS_UUID(t *testing.T) {
	memFS := fstest.MapFS{}
	uuid := "abc-123"
	f := New(memFS, ".", uuid)

	if f.UUID() != uuid {
		t.Errorf("expected UUID %s, got %s", uuid, f.UUID())
	}
}

func TestFS_OpenAndIterateFiles(t *testing.T) {
	memFS := fstest.MapFS{
		"file1.txt": &fstest.MapFile{
			Data:    []byte("hello"),
			Mode:    0644,
			ModTime: time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC),
		},
		"file2.txt": &fstest.MapFile{
			Data:    []byte("world"),
			Mode:    0644,
			ModTime: time.Date(2024, 5, 2, 12, 0, 0, 0, time.UTC),
		},
		"dir": &fstest.MapFile{
			Mode: fs.ModeDir,
		},
	}

	f := New(memFS, ".", "test-uuid")
	iter, err := f.Open()
	if err != nil {
		t.Fatalf("failed to open FS: %v", err)
	}
	defer iter.Close()

	ctx := context.Background()
	var seen []string

	for {
		handler, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error from Next: %v", err)
		}

		seen = append(seen, handler.Path())

		// test Read
		data, err := io.ReadAll(handler)
		if err != nil {
			t.Errorf("failed to read data from %s: %v", handler.Path(), err)
		}

		if !(strings.Contains(string(data), "hello") || strings.Contains(string(data), "world")) {
			t.Errorf("unexpected file content: %s", data)
		}

		// test metadata
		if handler.Etag() == "" {
			t.Errorf("etag should not be empty for %s", handler.Path())
		}
	}

	if len(seen) != 2 {
		t.Errorf("expected 2 files, saw %d: %v", len(seen), seen)
	}
}

func TestFS_EmptyFS(t *testing.T) {
	memFS := fstest.MapFS{}
	f := New(memFS, ".", "test-uuid")

	iter, err := f.Open()
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer iter.Close()

	ctx := context.Background()
	_, err = iter.Next(ctx)
	if err != io.EOF {
		t.Errorf("expected io.EOF, got: %v", err)
	}
}
