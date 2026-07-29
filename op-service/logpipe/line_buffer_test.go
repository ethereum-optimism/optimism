package logpipe

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLogProcessorSplitsAcrossWrites(t *testing.T) {
	var lines [][]byte
	proc := NewLineBuffer(func(line []byte) {
		// Copy to avoid aliasing the buffer
		lines = append(lines, append([]byte(nil), line...))
	})

	_, err := proc.Write([]byte("hello "))
	require.NoError(t, err)
	_, err = proc.Write([]byte("world\nsecond"))
	require.NoError(t, err)
	_, err = proc.Write([]byte(" line\nthird line\n"))
	require.NoError(t, err)

	require.NoError(t, proc.Close())

	require.Equal(t, [][]byte{
		[]byte("hello world"),
		[]byte("second line"),
		[]byte("third line"),
	}, lines)
}

func TestLogProcessorFlushesTrailingPartialLine(t *testing.T) {
	var lines [][]byte
	proc := NewLineBuffer(func(line []byte) {
		lines = append(lines, append([]byte(nil), line...))
	})

	_, err := proc.Write([]byte("partial line without newline"))
	require.NoError(t, err)
	require.NoError(t, proc.Close())

	require.Equal(t, [][]byte{
		[]byte("partial line without newline"),
	}, lines)
}
