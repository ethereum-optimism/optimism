package ioutil

import (
	"io"
	"os"
	"path/filepath"
)

type AtomicWriter struct {
	dest string
	temp string
	out  io.WriteCloser
}

// NewAtomicWriterCompressed creates a io.WriteCloser that performs an atomic write.
// The contents are initially written to a temporary file and only renamed into place when the writer is closed.
// NOTE: It's vital to check if an error is returned from Close() as it may indicate the file could not be renamed
// If path ends in .gz the contents written will be gzipped.
func NewAtomicWriterCompressed(path string, perm os.FileMode) (*AtomicWriter, error) {
	return newAtomicWriter(path, perm, true)
}

// NewAtomicWriter creates a io.WriteCloser that performs an atomic write.
// The contents are initially written to a temporary file and only renamed into place when the writer is closed.
// NOTE: It's vital to check if an error is returned from Close() as it may indicate the file could not be renamed
func NewAtomicWriter(path string, perm os.FileMode) (*AtomicWriter, error) {
	return newAtomicWriter(path, perm, false)
}

func newAtomicWriter(path string, perm os.FileMode, compressByFileType bool) (*AtomicWriter, error) {
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path))
	if err != nil {
		return nil, err
	}
	if err := f.Chmod(perm); err != nil {
		_ = f.Close()
		return nil, err
	}
	out := io.WriteCloser(f)
	if compressByFileType {
		out = CompressByFileType(path, f)
	}
	return &AtomicWriter{
		dest: path,
		temp: f.Name(),
		out:  out,
	}, nil
}

func (a *AtomicWriter) Write(p []byte) (n int, err error) {
	return a.out.Write(p)
}

// Abort releases any open resources and cleans up temporary files without renaming them into place.
// Does nothing if the writer has already been closed.
func (a *AtomicWriter) Abort() error {
	// Attempt to clean up the temp file even if Close fails.
	defer os.Remove(a.temp)
	return a.out.Close()
}

func (a *AtomicWriter) Close() error {
	// Attempt to clean up the temp file even if it can't be renamed into place.
	defer os.Remove(a.temp)
	if err := a.out.Close(); err != nil {
		return err
	}
	return os.Rename(a.temp, a.dest)
}
