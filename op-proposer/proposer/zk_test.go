package proposer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestZKExtraData(t *testing.T) {
	t.Run("ParentIndex", func(t *testing.T) {
		extraData := zkExtraData(1, []byte{0xbe, 0xef})
		require.Equal(t, []byte{0x00, 0x00, 0x00, 0x01, 0xbe, 0xef}, extraData)
	})

	t.Run("RootGameSentinel", func(t *testing.T) {
		extraData := zkExtraData(0xffffffff, []byte{0x01})
		require.Equal(t, []byte{0xff, 0xff, 0xff, 0xff, 0x01}, extraData)
	})

	t.Run("EmptyProof", func(t *testing.T) {
		extraData := zkExtraData(0x01020304, nil)
		require.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, extraData)
	})
}
