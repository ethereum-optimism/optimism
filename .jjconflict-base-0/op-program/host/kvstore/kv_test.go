package kvstore

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func kvTest(t *testing.T, kv KV) {
	t.Run("roundtrip", func(t *testing.T) {
		t.Parallel()
		_, err := kv.Get(common.Hash{0xaa})
		require.Equal(t, err, ErrNotFound, "file (in new tmp dir) does not exist yet")

		require.NoError(t, kv.Put(common.Hash{0xaa}, []byte("hello world")))
		dat, err := kv.Get(common.Hash{0xaa})
		require.NoError(t, err, "pre-image must exist now")
		require.Equal(t, "hello world", string(dat), "pre-image must match")
	})

	t.Run("empty pre-image", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, kv.Put(common.Hash{0xbb}, []byte{}))
		dat, err := kv.Get(common.Hash{0xbb})
		require.NoError(t, err, "pre-image must exist now")
		require.Zero(t, len(dat), "pre-image must be empty")
	})

	t.Run("zero pre-image key", func(t *testing.T) {
		t.Parallel()
		// in case we give a pre-image a special empty key. If it was a hash then we wouldn't know the pre-image.
		require.NoError(t, kv.Put(common.Hash{}, []byte("hello")))
		dat, err := kv.Get(common.Hash{})
		require.NoError(t, err, "pre-image must exist now")
		require.Equal(t, "hello", string(dat), "pre-image must match")
	})

	t.Run("non-string value", func(t *testing.T) {
		t.Parallel()
		// in case we give a pre-image a special empty key. If it was a hash then we wouldn't know the pre-image.
		require.NoError(t, kv.Put(common.Hash{0xcc}, []byte{4, 2}))
		dat, err := kv.Get(common.Hash{0xcc})
		require.NoError(t, err, "pre-image must exist now")
		require.Equal(t, []byte{4, 2}, dat, "pre-image must match")
	})

	t.Run("allowing multiple writes for same pre-image", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, kv.Put(common.Hash{0xdd}, []byte{4, 2}))
		require.NoError(t, kv.Put(common.Hash{0xdd}, []byte{4, 2}))
	})
}
