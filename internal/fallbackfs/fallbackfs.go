package fallbackfs

import (
	"errors"
	"io/fs"
)

type wrapper struct {
	fs       fs.FS
	fallback string
}

type FS interface {
	Open(name string) (fs.File, error)
}

func (w wrapper) Open(name string) (fs.File, error) {
	f, err := w.fs.Open(name)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		return w.fs.Open(w.fallback)
	}
	return f, err
}

func New(fs fs.FS, fallbackToFile string) FS {
	return wrapper{
		fs:       fs,
		fallback: fallbackToFile,
	}
}
