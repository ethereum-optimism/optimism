package ioutil

import (
	"io"
	"os"
)

var (
	stdOutStream OutputTarget = func() (io.Writer, io.Closer, Aborter, error) {
		return os.Stdout, &noopCloser{}, func() {}, nil
	}
)

type Aborter func()

type OutputTarget func() (io.Writer, io.Closer, Aborter, error)

func NoOutputStream() OutputTarget {
	return func() (io.Writer, io.Closer, Aborter, error) {
		return nil, nil, nil, nil
	}
}

func ToAtomicFile(path string, perm os.FileMode) OutputTarget {
	return func() (io.Writer, io.Closer, Aborter, error) {
		f, err := NewAtomicWriterCompressed(path, perm)
		if err != nil {
			return nil, nil, nil, err
		}
		return f, f, func() { _ = f.Abort() }, nil
	}
}

func ToStdOut() OutputTarget {
	return stdOutStream
}

func ToStdOutOrFileOrNoop(outputPath string, perm os.FileMode) OutputTarget {
	if outputPath == "" {
		return NoOutputStream()
	} else if outputPath == "-" {
		return ToStdOut()
	} else {
		return ToAtomicFile(outputPath, perm)
	}
}

type noopCloser struct{}

func (c *noopCloser) Close() error {
	return nil
}
