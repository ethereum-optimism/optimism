package proxyd

import (
	"errors"
	"io"
)

var ErrLimitReaderOverLimit = errors.New("over read limit")

func LimitReader(r io.Reader, n int64) io.Reader { return &LimitedReader{r, n} }

// A LimitedReader reads from R but limits the amount of
// data returned to just N bytes. Each call to Read
// updates N to reflect the new amount remaining.
// Unlike the standard library version, Read returns
// ErrLimitReaderOverLimit when N <= 0.
type LimitedReader struct {
	R io.Reader // underlying reader
	N int64     // max bytes remaining
}

func (l *LimitedReader) Read(p []byte) (int, error) {
	if l.N <= 0 {
		return 0, ErrLimitReaderOverLimit
	}
	if int64(len(p)) > l.N {
		p = p[0:l.N]
	}
	n, err := l.R.Read(p)
	l.N -= int64(n)
	return n, err
}
