package compression

import "io"

type Compressor interface {
	// Writer is used to write uncompressed data which will be compressed. Should return
	// ErrCompressorFull if the compressor is full and no more data should be written.
	io.Writer
	// Closer Close function should be called before reading any data.
	io.Closer
	// Reader is used to Read compressed data; should only be called after Close.
	io.Reader
	// Reset will reset all written data
	Reset()
	// Len returns an estimate of the current length of the compressed data; calling Flush will
	// increase the accuracy at the expense of a poorer compression ratio.
	Len() int
	// Flush flushes any uncompressed data to the compression buffer. This will result in a
	// non-optimal compression ratio.
	Flush() error
	// FullErr returns ErrCompressorFull if the compressor is known to be full. Note that
	// calls to Write will fail if an error is returned from this method, but calls to Write
	// can still return ErrCompressorFull even if this does not.
	FullErr() error
}
