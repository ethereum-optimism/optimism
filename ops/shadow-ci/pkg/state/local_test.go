package state

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := NewLocalStore(dir)

	err := store.Save("flake-db", []byte(`{"records":{}}`))
	require.NoError(t, err)

	data, err := store.Load("flake-db")
	require.NoError(t, err)
	assert.Equal(t, `{"records":{}}`, string(data))
}

func TestLocalStore_LoadNotFound(t *testing.T) {
	dir := t.TempDir()
	store := NewLocalStore(dir)

	_, err := store.Load("nonexistent")
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestLocalStore_Overwrite(t *testing.T) {
	dir := t.TempDir()
	store := NewLocalStore(dir)

	store.Save("key", []byte("v1"))
	store.Save("key", []byte("v2"))

	data, err := store.Load("key")
	require.NoError(t, err)
	assert.Equal(t, "v2", string(data))
}

func TestLocalStore_MultipleKeys(t *testing.T) {
	dir := t.TempDir()
	store := NewLocalStore(dir)

	store.Save("a", []byte("alpha"))
	store.Save("b", []byte("beta"))

	a, _ := store.Load("a")
	b, _ := store.Load("b")
	assert.Equal(t, "alpha", string(a))
	assert.Equal(t, "beta", string(b))
}
