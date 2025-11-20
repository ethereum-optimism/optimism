package artifacts

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/minio/sha256-simd"

	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestDownloadArtifacts_MockArtifacts(t *testing.T) {
	// Create a test tar.gz file for the mock server
	testTarGzPath := createTestTarGz(t)
	f, err := os.Open(testTarGzPath)
	require.NoError(t, err)
	defer f.Close()

	var callCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := io.Copy(w, f)
		require.NoError(t, err)
		// Seek to beginning of file for next request
		_, err = f.Seek(0, 0)
		require.NoError(t, err)
		atomic.AddInt32(&callCount, 1)
	}))
	defer ts.Close()

	ctx := context.Background()
	artifactsURL, err := url.Parse(ts.URL)
	require.NoError(t, err)
	loc := &Locator{
		URL: artifactsURL,
	}

	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	t.Run("success", func(t *testing.T) {
		fs, err := Download(ctx, loc, nil, testCacheDir)
		require.NoError(t, err)
		require.NotNil(t, fs)

		info, err := fs.Stat("WETH98.sol/WETH98.json")
		require.NoError(t, err)
		require.Greater(t, info.Size(), int64(0))
	})

	t.Run("bad integrity", func(t *testing.T) {
		_, err := downloadHTTP(ctx, loc.URL, nil, &hashIntegrityChecker{
			hash: common.Hash{'B', 'A', 'D'},
		}, testCacheDir)
		require.Error(t, err)
		require.ErrorContains(t, err, "integrity check failed")
	})

	correctIntegrity := &hashIntegrityChecker{
		hash: common.HexToHash("0x8171bd7ea902495701fecf396cdc9906273c8230205645a1293d5e27aea7ac9f"),
	}

	t.Run("ok integrity", func(t *testing.T) {
		_, err := downloadHTTP(ctx, loc.URL, nil, correctIntegrity, testCacheDir)
		require.NoError(t, err)
	})

	t.Run("caching works", func(t *testing.T) {
		u, err := url.Parse(loc.URL.String())
		require.NoError(t, err)
		u.Path = fmt.Sprintf("/different-path-%d", time.Now().UnixNano())

		startCalls := atomic.LoadInt32(&callCount)
		_, err = downloadHTTP(ctx, u, nil, correctIntegrity, testCacheDir)
		require.NoError(t, err)
		startCalls++
		require.Equal(t, startCalls, atomic.LoadInt32(&callCount))

		_, err = downloadHTTP(ctx, u, nil, correctIntegrity, testCacheDir)
		require.NoError(t, err)
		require.Equal(t, startCalls, atomic.LoadInt32(&callCount))
	})

	t.Run("caching validates integrity", func(t *testing.T) {
		u, err := url.Parse(loc.URL.String())
		require.NoError(t, err)
		u.Path = fmt.Sprintf("/different-path-%d", time.Now().UnixNano())
		_, err = downloadHTTP(ctx, u, nil, correctIntegrity, testCacheDir)
		require.NoError(t, err)

		cacheFile := fmt.Sprintf("%s/%x.tgz", testCacheDir, sha256.Sum256([]byte(u.String())))
		t.Cleanup(func() {
			require.NoError(t, os.Remove(cacheFile))
		})

		cacheF, err := os.OpenFile(cacheFile, os.O_RDWR, 0o644)
		require.NoError(t, err)
		_, err = cacheF.Write([]byte("bad data"))
		require.NoError(t, err)
		require.NoError(t, cacheF.Close())

		_, err = downloadHTTP(ctx, u, nil, correctIntegrity, testCacheDir)
		require.ErrorContains(t, err, "integrity check failed")
	})
}

func TestTarballExtractor_Extract(t *testing.T) {
	t.Run("gzip extraction", func(t *testing.T) {
		extractor := &TarballExtractor{
			checker: &noopIntegrityChecker{},
		}

		tempDir := t.TempDir()
		gzPath := createTestTarGz(t)
		destDir := tempDir

		err := extractor.Extract(gzPath, destDir)
		require.NoError(t, err)

		// Verify the extracted content
		forgeArtifactsDir := destDir + "/forge-artifacts"
		require.DirExists(t, forgeArtifactsDir)

		wethFile := forgeArtifactsDir + "/WETH98.sol/WETH98.json"
		require.FileExists(t, wethFile)

		info, err := os.Stat(wethFile)
		require.NoError(t, err)
		require.Greater(t, info.Size(), int64(0))
	})

	t.Run("zstd extraction", func(t *testing.T) {
		extractor := &TarballExtractor{
			checker: &noopIntegrityChecker{},
		}

		tempDir := t.TempDir()
		zstPath := createTestTarZst(t)
		destDir := tempDir

		err := extractor.Extract(zstPath, destDir)
		require.NoError(t, err)

		forgeArtifactsDir := destDir + "/forge-artifacts"
		require.DirExists(t, forgeArtifactsDir)

		wethFile := forgeArtifactsDir + "/WETH98.sol/WETH98.json"
		require.FileExists(t, wethFile)

		info, err := os.Stat(wethFile)
		require.NoError(t, err)
		require.Greater(t, info.Size(), int64(0))
	})

	t.Run("unsupported compression", func(t *testing.T) {
		extractor := &TarballExtractor{
			checker: &noopIntegrityChecker{},
		}

		tempDir := t.TempDir()

		// Create a test file with invalid compression
		invalidFile := tempDir + "/invalid.tar"
		err := os.WriteFile(invalidFile, []byte("not compressed data"), 0644)
		require.NoError(t, err)

		err = extractor.Extract(invalidFile, tempDir+"/dest")
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported compression format")
	})
}

func TestTarballExtractor_CompressionDetection(t *testing.T) {
	extractor := &TarballExtractor{}

	t.Run("gzip magic bytes", func(t *testing.T) {
		data := []byte{0x1f, 0x8b} // gzip magic bytes
		require.True(t, extractor.isGzipCompressed(data))
		require.False(t, extractor.isZstdCompressed(data))
	})

	t.Run("zstd magic bytes", func(t *testing.T) {
		data := []byte{0x28, 0xb5, 0x2f, 0xfd} // zstd magic bytes
		require.False(t, extractor.isGzipCompressed(data))
		require.True(t, extractor.isZstdCompressed(data))
	})

	t.Run("unknown magic bytes", func(t *testing.T) {
		data := []byte{0x00, 0x00, 0x00, 0x00} // unknown
		require.False(t, extractor.isGzipCompressed(data))
		require.False(t, extractor.isZstdCompressed(data))
	})

	t.Run("insufficient data", func(t *testing.T) {
		data := []byte{0x1f} // too short
		require.False(t, extractor.isGzipCompressed(data))
		require.False(t, extractor.isZstdCompressed(data))
	})
}

// createTestTar creates a tar buffer with test content
func createTestTar(t *testing.T) *bytes.Buffer {
	var tarBuffer bytes.Buffer
	tarWriter := tar.NewWriter(&tarBuffer)

	err := tarWriter.WriteHeader(&tar.Header{
		Name:     "forge-artifacts/",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	})
	require.NoError(t, err)

	err = tarWriter.WriteHeader(&tar.Header{
		Name:     "forge-artifacts/WETH98.sol/",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	})
	require.NoError(t, err)

	testContent := `{"abi":[],"bytecode":{"object":"608060405234801561001057600080fd5b50600436106100365760003560e01c80635c60da1b1461003b575b600080fd5b6100436100b5565b6040805173ffffffffffffffffffffffffffffffffffffffff9092168252519081900360200190f35b73000000000000000000000000000000000000000090565b9091019056fe"},"deployedBytecode":{"object":"608060405234801561001057600080fd5b50600436106100365760003560e01c80635c60da1b1461003b575b600080fd5b6100436100b5565b6040805173ffffffffffffffffffffffffffffffffffffffff9092168252519081900360200190f35b73000000000000000000000000000000000000000090565b9091019056fe"}}`
	err = tarWriter.WriteHeader(&tar.Header{
		Name: "forge-artifacts/WETH98.sol/WETH98.json",
		Mode: 0644,
		Size: int64(len(testContent)),
	})
	require.NoError(t, err)
	_, err = tarWriter.Write([]byte(testContent))
	require.NoError(t, err)

	err = tarWriter.Close()
	require.NoError(t, err)

	return &tarBuffer
}

// createTestTarGz creates a temporary tar.gz file with test content
func createTestTarGz(t *testing.T) string {
	tempDir := t.TempDir()
	tarGzPath := filepath.Join(tempDir, "test.tar.gz")

	tarBuffer := createTestTar(t)

	gzFile, err := os.Create(tarGzPath)
	require.NoError(t, err)
	defer gzFile.Close()

	gzWriter := gzip.NewWriter(gzFile)
	_, err = gzWriter.Write(tarBuffer.Bytes())
	require.NoError(t, err)
	err = gzWriter.Close()
	require.NoError(t, err)

	return tarGzPath
}

// createTestTarZst creates a temporary tar.zst file with test content
func createTestTarZst(t *testing.T) string {
	tempDir := t.TempDir()
	tarZstPath := filepath.Join(tempDir, "test.tar.zst")

	tarBuffer := createTestTar(t)

	zstFile, err := os.Create(tarZstPath)
	require.NoError(t, err)
	defer zstFile.Close()

	zstWriter, err := zstd.NewWriter(zstFile, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(19)))
	require.NoError(t, err)
	_, err = zstWriter.Write(tarBuffer.Bytes())
	require.NoError(t, err)
	err = zstWriter.Close()
	require.NoError(t, err)

	return tarZstPath
}
