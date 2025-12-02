package preimage

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreimageKeyTypes(t *testing.T) {
	t.Run("LocalIndexKey", func(t *testing.T) {
		actual := LocalIndexKey(0xFFFFFFFF)

		// PreimageKey encoding
		expected := [32]byte{}
		expected[0] = byte(LocalKeyType)
		binary.BigEndian.PutUint64(expected[24:], 0xFFFFFFFF)
		require.Equal(t, expected, actual.PreimageKey())
	})

	t.Run("Keccak256Key", func(t *testing.T) {
		fauxHash := [32]byte{}
		fauxHash[31] = 0xFF
		actual := Keccak256Key(fauxHash)

		// PreimageKey encoding
		expected := [32]byte{}
		expected[0] = byte(Keccak256KeyType)
		expected[31] = 0xFF
		require.Equal(t, expected, actual.PreimageKey())

		// String encoding
		require.Equal(t, "0x00000000000000000000000000000000000000000000000000000000000000ff", actual.String())
	})

	t.Run("Sha256Key", func(t *testing.T) {
		fauxHash := [32]byte{}
		fauxHash[31] = 0xFF
		actual := Sha256Key(fauxHash)

		// PreimageKey encoding
		expected := [32]byte{}
		expected[0] = byte(Sha256KeyType)
		expected[31] = 0xFF
		require.Equal(t, expected, actual.PreimageKey())

		// String encoding
		require.Equal(t, "0x00000000000000000000000000000000000000000000000000000000000000ff", actual.String())
	})

	t.Run("BlobKey", func(t *testing.T) {
		fauxHash := [32]byte{}
		fauxHash[31] = 0xFF
		actual := BlobKey(fauxHash)

		// PreimageKey encoding
		expected := [32]byte{}
		expected[0] = byte(BlobKeyType)
		expected[31] = 0xFF
		require.Equal(t, expected, actual.PreimageKey())

		// String encoding
		require.Equal(t, "0x00000000000000000000000000000000000000000000000000000000000000ff", actual.String())
	})

	t.Run("KZGPointEvaluationKey", func(t *testing.T) {
		fauxHash := [32]byte{}
		fauxHash[31] = 0xFF
		actual := PrecompileKey(fauxHash)

		// PreimageKey encoding
		expected := [32]byte{}
		expected[0] = byte(PrecompileKeyType)
		expected[31] = 0xFF
		require.Equal(t, expected, actual.PreimageKey())

		// String encoding
		require.Equal(t, "0x00000000000000000000000000000000000000000000000000000000000000ff", actual.String())
	})
}
