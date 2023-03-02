package fallbackfs

import (
	"errors"
	"io/fs"
)

type fallbackfs struct {
	fs       fs.FS
	fallback string
}

func (w fallbackfs) Open(name string) (fs.File, error) {
	f, err := w.fs.Open(name)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		return w.fs.Open(w.fallback)
	}
	return f, err
}

func New(fs fs.FS, fallbackToFile string) fs.FS {
	return fallbackfs{
		fs:       fs,
		fallback: fallbackToFile,
	}
}
