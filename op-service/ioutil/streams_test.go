package ioutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNoOutputStream(t *testing.T) {
	writer, closer, aborter, err := NoOutputStream()()
	require.NoError(t, err)
	require.Nil(t, writer)
	require.Nil(t, closer)
	require.Nil(t, aborter)
}

func TestToStdOut(t *testing.T) {
	writer, closer, aborter, err := ToStdOut()()
	require.NoError(t, err)
	require.Same(t, os.Stdout, writer)

	// Should not close StdOut
	require.NoError(t, closer.Close())
	_, err = os.Stdout.WriteString("TestToStdOut After Close\n")
	require.NoError(t, err)

	aborter()
	_, err = os.Stdout.WriteString("TestToStdOut After Abort\n")
	require.NoError(t, err)
}

func TestToAtomicFile(t *testing.T) {
	t.Run("Abort", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.txt")
		writer, closer, aborter, err := ToAtomicFile(path, 0o644)()
		defer closer.Close()
		require.NoError(t, err)

		expected := []byte("test")
		_, err = writer.Write(expected)
		require.NoError(t, err)
		aborter()

		_, err = os.Stat(path)
		require.ErrorIs(t, err, os.ErrNotExist, "Should not have written file")
	})

	t.Run("Close", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.txt")
		writer, closer, _, err := ToAtomicFile(path, 0o644)()
		defer closer.Close()
		require.NoError(t, err)

		expected := []byte("test")
		_, err = writer.Write(expected)
		require.NoError(t, err)

		_, err = os.Stat(path)
		require.ErrorIs(t, err, os.ErrNotExist, "Target file should not exist prior to Close")

		require.NoError(t, closer.Close())
		actual, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})
}

func TestToStdOutOrFileOrNoop(t *testing.T) {
	t.Run("EmptyOutputPath", func(t *testing.T) {
		writer, _, _, err := ToStdOutOrFileOrNoop("", 0o644)()
		require.NoError(t, err)
		require.Nil(t, writer, "Should use no output stream")
	})

	t.Run("StdOut", func(t *testing.T) {
		writer, _, _, err := ToStdOutOrFileOrNoop("-", 0o644)()
		require.NoError(t, err)
		require.Same(t, os.Stdout, writer, "Should use std out")
	})

	t.Run("File", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.txt")
		writer, closer, _, err := ToStdOutOrFileOrNoop(path, 0o644)()
		defer closer.Close()
		require.NoError(t, err)

		expected := []byte("test")
		_, err = writer.Write(expected)
		require.NoError(t, err)
		require.NoError(t, closer.Close())
		actual, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})
}
