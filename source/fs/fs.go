package fs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sync"

	"github.com/opengs/file2llm/source"
)

type FS struct {
	fs   fs.FS
	path string
	uuid string
}

func New(fs fs.FS, path string, uuid string) *FS {
	return &FS{
		fs:   fs,
		path: path,
		uuid: uuid,
	}
}

func (f *FS) UUID() string {
	return f.uuid
}

func (f *FS) Open() (source.Iterator, error) {
	return &fsIterator{
		fs:     f.fs,
		walker: newWalker(f.fs, f.path),
	}, nil
}

func (f *FS) NotifyFileProcessingStarted(ctx context.Context, event source.FileProcessingStartedEvent) error {
	return nil
}
func (f *FS) NotifyFileProcessingRunning(ctx context.Context, event source.FileProcessingRunningEvent) error {
	return nil
}
func (f *FS) NotifyFileProcessingDone(ctx context.Context, event source.FileProcessingDoneEvent) error {
	return nil
}

type fsIterator struct {
	fs     fs.FS
	walker *walker
	locker sync.Mutex
}

func (i *fsIterator) Next(ctx context.Context) (source.FileHandler, error) {
	i.locker.Lock()
	defer i.locker.Unlock()

	for i.walker.Next() {
		if i.walker.Err() != nil {
			return nil, i.walker.Err()
		}

		entry := i.walker.Entry()
		if entry.IsDir() {
			continue
		}

		fileInfo, err := i.walker.Entry().Info()
		if err != nil {
			return nil, errors.Join(errors.New("error while reading file info"), err)
		}

		etag := fmt.Sprintf("%s_%d", fileInfo.ModTime().String(), fileInfo.Size())
		handler := &fsFileHandler{
			fs:   i.fs,
			etag: etag,
			path: i.walker.Path(),
		}
		return handler, nil
	}

	if i.walker.Err() != nil {
		return nil, i.walker.Err()
	}

	return nil, io.EOF
}

func (i *fsIterator) Close() error {
	return nil
}

type fsFileHandler struct {
	fs   fs.FS
	fp   fs.File
	path string
	etag string
}

func (h *fsFileHandler) Etag() string {
	return h.etag
}

func (h *fsFileHandler) Path() string {
	return h.path
}

func (h *fsFileHandler) Close() error {
	if h.fp != nil {
		return h.fp.Close()
	}
	return nil
}
func (h *fsFileHandler) Read(p []byte) (n int, err error) {
	if h.fp == nil {
		fp, err := h.fs.Open(h.path)
		if err != nil {
			return 0, errors.Join(errors.New("failed to open file for reading"), err)
		}
		h.fp = fp
	}

	return h.fp.Read(p)
}

func (h *fsFileHandler) UserMetadata() any {
	return nil
}
