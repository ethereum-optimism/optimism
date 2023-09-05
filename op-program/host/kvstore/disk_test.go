package kvstore

import (
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestDiskKV(t *testing.T) {
	tmp := t.TempDir() // automatically removed by testing cleanup
	kv := NewDiskKV(tmp)
	kvTest(t, kv)
}

func TestCreateMissingDirectory(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "data")
	kv := NewDiskKV(dir)
	val := []byte{1, 2, 3, 4}
	key := crypto.Keccak256Hash(val)
	require.NoError(t, kv.Put(key, val))
}
