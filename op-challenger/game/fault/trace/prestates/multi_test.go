package prestates

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
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
	hash := common.Hash{0xaa}
	provider := NewMultiPrestateProvider(parseURL(t, server.URL), dir, &stubStateConverter{hash: hash})
	path, err := provider.PrestatePath(hash)
	require.NoError(t, err)
	in, err := ioutil.OpenDecompressed(path)
	require.NoError(t, err)
	defer in.Close()
	content, err := io.ReadAll(in)
	require.NoError(t, err)
	require.Equal(t, "/"+hash.Hex()+".bin.gz", string(content))
}

func TestCreateDirectory(t *testing.T) {
	dir := t.TempDir()
	dir = filepath.Join(dir, "test")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.URL.Path))
	}))
	defer server.Close()
	hash := common.Hash{0xaa}
	provider := NewMultiPrestateProvider(parseURL(t, server.URL), dir, &stubStateConverter{hash: hash})
	path, err := provider.PrestatePath(hash)
	require.NoError(t, err)
	in, err := ioutil.OpenDecompressed(path)
	require.NoError(t, err)
	defer in.Close()
	content, err := io.ReadAll(in)
	require.NoError(t, err)
	require.Equal(t, "/"+hash.Hex()+".bin.gz", string(content))
}

func TestExistingPrestate(t *testing.T) {
	dir := t.TempDir()
	hash := common.Hash{0xaa}
	provider := NewMultiPrestateProvider(parseURL(t, "http://127.0.0.1:1"), dir, &stubStateConverter{hash: hash})
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
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		w.WriteHeader(404)
	}))
	defer server.Close()
	hash := common.Hash{0xaa}
	provider := NewMultiPrestateProvider(parseURL(t, server.URL), dir, &stubStateConverter{hash: hash})
	path, err := provider.PrestatePath(hash)
	require.ErrorIs(t, err, ErrPrestateUnavailable)
	_, err = os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
	expectedRequests := []string{
		"/" + hash.Hex() + ".bin.gz",
		"/" + hash.Hex() + ".json.gz",
		"/" + hash.Hex() + ".json",
	}
	require.Equal(t, expectedRequests, requests)
}

func TestStorePrestateWithCorrectExtension(t *testing.T) {
	extensions := []string{".bin.gz", ".json.gz", ".json"}
	for _, ext := range extensions {
		ext := ext
		t.Run(ext, func(t *testing.T) {
			dir := t.TempDir()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasSuffix(r.URL.Path, ext) {
					w.WriteHeader(404)
					return
				}
				_, _ = w.Write([]byte("content"))
			}))
			defer server.Close()
			hash := common.Hash{0xaa}
			provider := NewMultiPrestateProvider(parseURL(t, server.URL), dir, &stubStateConverter{hash: hash})
			path, err := provider.PrestatePath(hash)
			require.NoError(t, err)
			require.Truef(t, strings.HasSuffix(path, ext), "Expected path %v to have extension %v", path, ext)
			in, err := ioutil.OpenDecompressed(path)
			require.NoError(t, err)
			defer in.Close()
			content, err := io.ReadAll(in)
			require.NoError(t, err)
			require.Equal(t, "content", string(content))
		})
	}
}

func TestDetectInvalidPrestate(t *testing.T) {
	dir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()
	hash := common.Hash{0xaa}
	provider := NewMultiPrestateProvider(parseURL(t, server.URL), dir, &stubStateConverter{hash: hash, err: errors.New("boom")})
	_, err := provider.PrestatePath(hash)
	require.ErrorIs(t, err, ErrPrestateUnavailable)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, entries, "should not leave any files in temp dir")
}

func TestDetectPrestateWithWrongHash(t *testing.T) {
	dir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()
	hash := common.Hash{0xaa}
	actualHash := common.Hash{0xbb}
	provider := NewMultiPrestateProvider(parseURL(t, server.URL), dir, &stubStateConverter{hash: actualHash})
	_, err := provider.PrestatePath(hash)
	require.ErrorIs(t, err, ErrPrestateUnavailable)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, entries, "should not leave any files in temp dir")
}

func parseURL(t *testing.T, str string) *url.URL {
	parsed, err := url.Parse(str)
	require.NoError(t, err)
	return parsed
}

type stubStateConverter struct {
	err  error
	hash common.Hash
}

func (s *stubStateConverter) ConvertStateToProof(path string) (*utils.ProofData, uint64, bool, error) {
	return &utils.ProofData{ClaimValue: s.hash}, 0, false, s.err
}
