package ioutil

import (
	"io"
	"os"
	"path/filepath"
)

type atomicWriter struct {
	dest string
	temp string
	out  io.WriteCloser
}

// NewAtomicWriterCompressed creates a io.WriteCloser that performs an atomic write.
// The contents are initially written to a temporary file and only renamed into place when the writer is closed.
// NOTE: It's vital to check if an error is returned from Close() as it may indicate the file could not be renamed
// If path ends in .gz the contents written will be gzipped.
func NewAtomicWriterCompressed(path string, perm os.FileMode) (io.WriteCloser, error) {
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path))
	if err != nil {
		return nil, err
	}
	if err := f.Chmod(perm); err != nil {
		_ = f.Close()
		return nil, err
	}
	return &atomicWriter{
		dest: path,
		temp: f.Name(),
		out:  CompressByFileType(path, f),
	}, nil
}

func (a *atomicWriter) Write(p []byte) (n int, err error) {
	return a.out.Write(p)
}

func (a *atomicWriter) Close() error {
	// Attempt to clean up the temp file even if it can't be renamed into place.
	defer os.Remove(a.temp)
	if err := a.out.Close(); err != nil {
		return err
	}
	return os.Rename(a.temp, a.dest)
}
