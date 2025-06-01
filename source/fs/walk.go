package fs

import (
	"errors"
	"io/fs"
	"path"
)

var errUsage = errors.New("walk: method Next must be called first")

type walker struct {
	fsys    fs.FS
	cur     visit
	stack   []visit
	descend bool
}

type visit struct {
	path    string
	info    fs.DirEntry
	err     error
	skipDir int
	skipPar int
}

func newWalker(fsys fs.FS, root string) *walker {
	info, err := fs.Stat(fsys, root)
	return &walker{
		fsys:  fsys,
		cur:   visit{err: errUsage},
		stack: []visit{{root, infoDirEntry{info}, err, 0, 0}},
	}
}

func (w *walker) Next() bool {
	if w.descend && w.cur.err == nil && w.cur.info.IsDir() {
		dir, err := fs.ReadDir(w.fsys, w.cur.path)
		n := len(w.stack)
		for i := len(dir) - 1; i >= 0; i-- {
			p := path.Join(w.cur.path, dir[i].Name())
			w.stack = append(w.stack, visit{p, dir[i], nil, len(w.stack), n})
		}
		if err != nil {
			// Second visit, to report ReadDir error.
			w.cur.err = err
			w.stack = append(w.stack, w.cur)
		}
	}

	if len(w.stack) == 0 {
		w.descend = false
		return false
	}
	i := len(w.stack) - 1
	w.cur = w.stack[i]
	w.stack = w.stack[:i]
	w.descend = true
	return true
}

func (w *walker) Path() string {
	return w.cur.path
}

func (w *walker) Entry() fs.DirEntry {
	return w.cur.info
}

func (w *walker) Err() error {
	return w.cur.err
}

func (w *walker) SkipDir() {
	w.descend = false
	w.stack = w.stack[:w.cur.skipDir]
}

func (w *walker) SkipParent() {
	w.descend = false
	w.stack = w.stack[:w.cur.skipPar]
}

type infoDirEntry struct{ f fs.FileInfo }

func (e infoDirEntry) Name() string               { return e.f.Name() }
func (e infoDirEntry) IsDir() bool                { return e.f.IsDir() }
func (e infoDirEntry) Type() fs.FileMode          { return e.f.Mode().Type() }
func (e infoDirEntry) Info() (fs.FileInfo, error) { return e.f, nil }
