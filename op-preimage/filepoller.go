package preimage

import (
	"context"
	"errors"
	"os"
	"time"
)

// FilePoller is a ReadWriteCloser that polls the underlying file channel for reads and writes
// until its context is done. This is useful to detect when the other end of a
// blocking pre-image channel is no longer available.
type FilePoller struct {
	File        FileChannel
	ctx         context.Context
	pollTimeout time.Duration
}

// NewFilePoller returns a FilePoller that polls the underlying file channel for reads and writes until
// the provided ctx is done. The poll timeout is the maximum amount of time to wait for I/O before
// the operation is halted and the context is checked for cancellation.
func NewFilePoller(ctx context.Context, f FileChannel, pollTimeout time.Duration) *FilePoller {
	return &FilePoller{File: f, ctx: ctx, pollTimeout: pollTimeout}
}

func (f *FilePoller) Read(b []byte) (int, error) {
	var read int
	for {
		if err := f.File.Reader().SetReadDeadline(time.Now().Add(f.pollTimeout)); err != nil {
			return 0, err
		}
		n, err := f.File.Read(b[read:])
		read += n
		if errors.Is(err, os.ErrDeadlineExceeded) {
			if cerr := f.ctx.Err(); cerr != nil {
				return read, cerr
			}
		} else {
			if read >= len(b) {
				return read, err
			}
		}
	}
}

func (f *FilePoller) Write(b []byte) (int, error) {
	var written int
	for {
		if err := f.File.Writer().SetWriteDeadline(time.Now().Add(f.pollTimeout)); err != nil {
			return 0, err
		}
		n, err := f.File.Write(b[written:])
		written += n
		if errors.Is(err, os.ErrDeadlineExceeded) {
			if cerr := f.ctx.Err(); cerr != nil {
				return written, cerr
			}
		} else {
			if written >= len(b) {
				return written, err
			}
		}
	}
}

func (p *FilePoller) Close() error {
	return p.File.Close()
}
