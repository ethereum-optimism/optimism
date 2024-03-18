package compressor

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNonCompressor(t *testing.T) {
	require := require.New(t)
	c, err := NewNonCompressor(Config{
		TargetFrameSize: 1000,
		TargetNumFrames: 100,
	})
	require.NoError(err)

	const dlen = 100
	data := make([]byte, dlen)
	rng := rand.New(rand.NewSource(42))
	rng.Read(data)

	n, err := c.Write(data)
	require.NoError(err)
	require.Equal(n, dlen)
	l0 := c.Len()
	require.Less(l0, dlen)
	require.Equal(7, l0)
	c.Flush()
	l1 := c.Len()
	require.Greater(l1, l0)
	require.Greater(l1, dlen)

	n, err = c.Write(data)
	require.NoError(err)
	require.Equal(n, dlen)
	l2 := c.Len()
	require.Equal(l1+5, l2)
}
