package compressor_test

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	"github.com/stretchr/testify/require"
)

func TestBlindCompressorLimit(t *testing.T) {
	bc, err := compressor.NewBlindCompressor(compressor.Config{
		TargetOutputSize: 10,
	})
	require.NoError(t, err)

	// write far too much data to the compressor, but never flush
	for i := 0; i < 100; i++ {
		_, err := bc.Write([]byte("hello"))
		require.NoError(t, err)
		require.NoError(t, bc.FullErr())
	}

	// finally flush the compressor and see that it is full
	bc.Flush()
	require.Error(t, bc.FullErr())

	// write a little more data to the compressor and see that it is still full
	_, err = bc.Write([]byte("hello"))
	require.Error(t, err)
}
