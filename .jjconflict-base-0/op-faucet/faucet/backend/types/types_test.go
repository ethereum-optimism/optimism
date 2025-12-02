package types

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFaucetID(t *testing.T) {
	var x FaucetID
	require.NoError(t, x.UnmarshalText([]byte("hello")))
	require.Equal(t, "hello", x.String())
	data, err := x.MarshalText()
	require.NoError(t, err)
	require.Equal(t, "hello", string(data))

	var y FaucetID
	require.ErrorIs(t, y.UnmarshalText(bytes.Repeat([]byte("a"), maxIDLength+1)), ErrInvalidID)
	require.ErrorIs(t, y.UnmarshalText([]byte{}), ErrInvalidID)

	_, err = FaucetID("").MarshalText()
	require.ErrorIs(t, err, ErrInvalidID)

	_, err = FaucetID(strings.Repeat("a", maxIDLength+1)).MarshalText()
	require.ErrorIs(t, err, ErrInvalidID)
}
