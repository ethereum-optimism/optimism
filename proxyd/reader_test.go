package proxyd

import (
	"github.com/stretchr/testify/require"
	"io"
	"strings"
	"testing"
)

func TestLimitReader(t *testing.T) {
	data := "hellohellohellohello"
	r := LimitReader(strings.NewReader(data), 3)
	buf := make([]byte, 3)

	// Buffer reads OK
	n, err := r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 3, n)

	// Buffer is over limit
	n, err = r.Read(buf)
	require.Equal(t, ErrLimitReaderOverLimit, err)
	require.Equal(t, 0, n)

	// Buffer on initial read is over size
	buf = make([]byte, 16)
	r = LimitReader(strings.NewReader(data), 3)
	n, err = r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 3, n)

	// test with read all where the limit is less than the data
	r = LimitReader(strings.NewReader(data), 3)
	out, err := io.ReadAll(r)
	require.Equal(t, ErrLimitReaderOverLimit, err)
	require.Equal(t, "hel", string(out))

	// test with read all where the limit is more than the data
	r = LimitReader(strings.NewReader(data), 21)
	out, err = io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, data, string(out))
}
