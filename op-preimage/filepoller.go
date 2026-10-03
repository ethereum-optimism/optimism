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
		if err := f.contextErr(); err != nil {
			return read, err
		}
		if err := f.File.Reader().SetReadDeadline(f.deadline()); err != nil {
			return 0, err
		}
		n, err := f.File.Read(b[read:])
		read += n
		if errors.Is(err, os.ErrDeadlineExceeded) {
			if cerr := f.contextErr(); cerr != nil {
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
		if err := f.contextErr(); err != nil {
			return written, err
		}
		if err := f.File.Writer().SetWriteDeadline(f.deadline()); err != nil {
			return 0, err
		}
		n, err := f.File.Write(b[written:])
		written += n
		if errors.Is(err, os.ErrDeadlineExceeded) {
			if cerr := f.contextErr(); cerr != nil {
				return written, cerr
			}
		} else {
			if written >= len(b) {
				return written, err
			}
		}
	}
}

func (f *FilePoller) deadline() time.Time {
	deadline := time.Now().Add(f.pollTimeout)
	if ctxDeadline, ok := f.ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		return ctxDeadline
	}
	return deadline
}

func (f *FilePoller) contextErr() error {
	if err := f.ctx.Err(); err != nil {
		return err
	}
	// The file deadline may fire before the context timer publishes its error.
	if deadline, ok := f.ctx.Deadline(); ok && !time.Now().Before(deadline) {
		return context.DeadlineExceeded
	}
	return nil
}

func (p *FilePoller) Close() error {
	return p.File.Close()
}
