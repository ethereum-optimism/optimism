package kvstore

import (
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestDirectoryKV(t *testing.T) {
	tmp := t.TempDir() // automatically removed by testing cleanup
	kv := newDirectoryKV(tmp)
	t.Cleanup(func() { // Can't use defer because kvTest runs tests in parallel.
		require.NoError(t, kv.Close())
	})
	kvTest(t, kv)
}

func TestDirectoryKV_CreateMissingDirectory(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "data")
	kv := newDirectoryKV(dir)
	defer kv.Close()
	val := []byte{1, 2, 3, 4}
	key := crypto.Keccak256Hash(val)
	require.NoError(t, kv.Put(key, val))
}
