package compressor

import (
	"bytes"
	"compress/zlib"
	"io"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCloseOverheadZlib(t *testing.T) {
	var buf bytes.Buffer
	z := zlib.NewWriter(&buf)
	rng := rand.New(rand.NewSource(420))
	_, err := io.CopyN(z, rng, 0xff)
	require.NoError(t, err)

	require.NoError(t, z.Flush())
	fsize := buf.Len()
	require.NoError(t, z.Close())
	csize := buf.Len()
	require.Equal(t, CloseOverheadZlib, csize-fsize)
}
