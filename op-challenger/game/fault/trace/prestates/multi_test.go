package prestates

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestDownloadPrestate(t *testing.T) {
	dir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.URL.Path))
	}))
	defer server.Close()
	provider := NewMultiPrestateProvider(parseURL(t, server.URL), dir)
	hash := common.Hash{0xaa}
	path, err := provider.PrestatePath(hash)
	require.NoError(t, err)
	in, err := ioutil.OpenDecompressed(path)
	require.NoError(t, err)
	defer in.Close()
	content, err := io.ReadAll(in)
	require.NoError(t, err)
	require.Equal(t, "/"+hash.Hex()+".json", string(content))
}

func TestCreateDirectory(t *testing.T) {
	dir := t.TempDir()
	dir = filepath.Join(dir, "test")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.URL.Path))
	}))
	defer server.Close()
	provider := NewMultiPrestateProvider(parseURL(t, server.URL), dir)
	hash := common.Hash{0xaa}
	path, err := provider.PrestatePath(hash)
	require.NoError(t, err)
	in, err := ioutil.OpenDecompressed(path)
	require.NoError(t, err)
	defer in.Close()
	content, err := io.ReadAll(in)
	require.NoError(t, err)
	require.Equal(t, "/"+hash.Hex()+".json", string(content))
}

func TestExistingPrestate(t *testing.T) {
	dir := t.TempDir()
	provider := NewMultiPrestateProvider(parseURL(t, "http://127.0.0.1:1"), dir)
	hash := common.Hash{0xaa}
	expectedFile := filepath.Join(dir, hash.Hex()+".json.gz")
	err := ioutil.WriteCompressedBytes(expectedFile, []byte("expected content"), os.O_WRONLY|os.O_CREATE, 0o644)
	require.NoError(t, err)

	path, err := provider.PrestatePath(hash)
	require.NoError(t, err)
	require.Equal(t, expectedFile, path)
	in, err := ioutil.OpenDecompressed(path)
	require.NoError(t, err)
	defer in.Close()
	content, err := io.ReadAll(in)
	require.NoError(t, err)
	require.Equal(t, "expected content", string(content))
}

func TestMissingPrestate(t *testing.T) {
	dir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()
	provider := NewMultiPrestateProvider(parseURL(t, server.URL), dir)
	hash := common.Hash{0xaa}
	path, err := provider.PrestatePath(hash)
	require.ErrorIs(t, err, ErrPrestateUnavailable)
	_, err = os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func parseURL(t *testing.T, str string) *url.URL {
	parsed, err := url.Parse(str)
	require.NoError(t, err)
	return parsed
}
