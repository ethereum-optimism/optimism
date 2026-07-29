package types

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSyncTesterID(t *testing.T) {
	var x SyncTesterID
	require.NoError(t, x.UnmarshalText([]byte("hello")))
	require.Equal(t, "hello", x.String())
	data, err := x.MarshalText()
	require.NoError(t, err)
	require.Equal(t, "hello", string(data))

	var y SyncTesterID
	require.ErrorIs(t, y.UnmarshalText(bytes.Repeat([]byte("a"), maxIDLength+1)), ErrInvalidID)
	require.ErrorIs(t, y.UnmarshalText([]byte{}), ErrInvalidID)

	_, err = SyncTesterID("").MarshalText()
	require.ErrorIs(t, err, ErrInvalidID)

	_, err = SyncTesterID(strings.Repeat("a", maxIDLength+1)).MarshalText()
	require.ErrorIs(t, err, ErrInvalidID)
}
