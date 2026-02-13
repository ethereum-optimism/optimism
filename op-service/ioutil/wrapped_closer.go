package ioutil

import (
	"errors"
	"io"
)

// WrappedReadCloser is a struct that closes both the gzip.Reader and the underlying io.Closer.
type WrappedReadCloser struct {
	io.ReadCloser
	closer io.Closer
}

// WrappedWriteCloser is a struct that closes both the gzip.Writer and the underlying io.Closer.
type WrappedWriteCloser struct {
	io.WriteCloser
	closer io.Closer
}

// Close closes both the gzip.Reader and the underlying reader.
func (g *WrappedReadCloser) Close() error {
	return errors.Join(g.ReadCloser.Close(), g.closer.Close())
}

// Close closes both the gzip.Writer and the underlying writer.
func (g *WrappedWriteCloser) Close() error {
	return errors.Join(g.WriteCloser.Close(), g.closer.Close())
}

// NewWrappedReadCloser is a constructor function that initializes a WrappedReadCloser structure.
func NewWrappedReadCloser(r io.ReadCloser, c io.Closer) *WrappedReadCloser {
	return &WrappedReadCloser{
		ReadCloser: r,
		closer:     c,
	}
}

// NewWrappedWriteCloser is a constructor function that initializes a WrappedWriteCloser structure.
func NewWrappedWriteCloser(r io.WriteCloser, c io.Closer) *WrappedWriteCloser {
	return &WrappedWriteCloser{
		WriteCloser: r,
		closer:      c,
	}
}
