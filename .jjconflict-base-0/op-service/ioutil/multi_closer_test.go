package ioutil

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultiCloser(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		closer := &MultiCloser{}
		require.NoError(t, closer.Close())
	})

	t.Run("Single", func(t *testing.T) {
		closee1 := &testCloser{}
		closer := &MultiCloser{closee1}
		require.NoError(t, closer.Close())
		require.True(t, closee1.closed)
	})

	t.Run("Multiple", func(t *testing.T) {
		closee1 := &testCloser{}
		closee2 := &testCloser{}
		closee3 := &testCloser{}
		closer := &MultiCloser{closee1, closee2, closee3}
		require.NoError(t, closer.Close())
		require.True(t, closee1.closed)
		require.True(t, closee2.closed)
		require.True(t, closee3.closed)
	})

	t.Run("ErrorOnFirst", func(t *testing.T) {
		closee1 := &testCloser{err: errors.New("first error")}
		closee2 := &testCloser{}
		closer := &MultiCloser{closee1, closee2}
		require.ErrorIs(t, closer.Close(), closee1.err)
		require.True(t, closee1.closed)
		require.True(t, closee2.closed, "Should still close closee2")
	})

	t.Run("ErrorOnMultiple", func(t *testing.T) {
		closee1 := &testCloser{err: errors.New("first error")}
		closee2 := &testCloser{err: errors.New("second error")}
		closee3 := &testCloser{}
		closer := &MultiCloser{closee1, closee2, closee3}
		// Returned error should combine all returned errors
		require.ErrorIs(t, closer.Close(), closee1.err)
		require.ErrorIs(t, closer.Close(), closee2.err)
		require.True(t, closee1.closed)
		require.True(t, closee2.closed, "Should still close closee2")
		require.True(t, closee3.closed, "Should still close closee3")
	})
}

type testCloser struct {
	closed bool
	err    error
}

func (t *testCloser) Close() error {
	t.closed = true
	return t.err
}
