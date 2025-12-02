package artifacts

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/httputil"
	"github.com/klauspost/compress/zstd"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
)

var ErrUnsupportedArtifactsScheme = errors.New("unsupported artifacts URL scheme")

type Downloader interface {
	Download(ctx context.Context, url string, progress ioutil.Progressor, targetDir string) (string, error)
}

type Extractor interface {
	Extract(src string, dest string) (string, error)
}

func Download(ctx context.Context, loc *Locator, progressor ioutil.Progressor, targetDir string) (foundry.StatDirFs, error) {
	if progressor == nil {
		progressor = ioutil.NoopProgressor()
	}

	var err error
	u := loc.URL
	checker := new(noopIntegrityChecker)

	var artifactsFS fs.FS
	switch u.Scheme {
	case "http", "https":
		artifactsFS, err = downloadHTTP(ctx, u, progressor, checker, targetDir)
		if err != nil {
			return nil, fmt.Errorf("failed to download artifacts: %w", err)
		}
	case "file":
		// Check the path has forge-artifacts directory
		forgeArtifactsDir := path.Join(u.Path, "forge-artifacts")
		if _, err := os.Stat(forgeArtifactsDir); err != nil {
			// TODO(#18346): Accept this for now but in the future we should error
			artifactsFS = os.DirFS(u.Path)
		} else {
			artifactsFS = os.DirFS(forgeArtifactsDir)
		}
	case "embedded":
		artifactsFS, err = ExtractEmbedded(targetDir)
		if err != nil {
			return nil, fmt.Errorf("failed to extract embedded artifacts: %w", err)
		}
	default:
		return nil, ErrUnsupportedArtifactsScheme
	}
	return artifactsFS.(foundry.StatDirFs), nil
}

func downloadHTTP(ctx context.Context, u *url.URL, progressor ioutil.Progressor, checker integrityChecker, targetDir string) (fs.FS, error) {
	cacher := &CachingDownloader{
		d: new(HTTPDownloader),
	}

	tarballPath, err := cacher.Download(ctx, u.String(), progressor, targetDir)
	if err != nil {
		return nil, fmt.Errorf("failed to download artifacts: %w", err)
	}
	tmpDir, err := os.MkdirTemp(targetDir, "op-deployer-artifacts-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	if strings.HasSuffix(tarballPath, ".tzst") {
		_, err := ExtractFromFile(tmpDir, tarballPath)
		if err != nil {
			return nil, fmt.Errorf("failed to extract embedded artifacts: %w", err)
		}
	} else {
		extractor := &TarballExtractor{
			checker: checker,
		}
		if err := extractor.Extract(tarballPath, tmpDir); err != nil {
			return nil, fmt.Errorf("failed to extract tarball: %w", err)
		}
	}
	// TODO(#18346): Change this to provide the parent directory of the forge-artifacts directory
	return os.DirFS(path.Join(tmpDir, "forge-artifacts")), nil
}

type HTTPDownloader struct{}

func (d *HTTPDownloader) Download(ctx context.Context, url string, progress ioutil.Progressor, targetDir string) (string, error) {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to ensure cache directory '%s': %w", targetDir, err)
	}
	tmpFile, err := os.CreateTemp(targetDir, "op-deployer-artifacts-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	downloader := &httputil.Downloader{
		Progressor: progress,
	}
	if err := downloader.Download(ctx, url, tmpFile); err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	return tmpFile.Name(), nil
}

type CachingDownloader struct {
	d   Downloader
	mtx sync.Mutex
}

func (d *CachingDownloader) Download(ctx context.Context, url string, progress ioutil.Progressor, targetDir string) (string, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	var ext string
	if strings.HasSuffix(url, ".tzst") || strings.Contains(url, ".tzst") {
		ext = ".tzst"
	} else {
		ext = ".tgz"
	}

	cachePath := path.Join(targetDir, fmt.Sprintf("%x%s", sha256.Sum256([]byte(url)), ext))
	if _, err := os.Stat(cachePath); err == nil {
		return cachePath, nil
	}
	tmpPath, err := d.d.Download(ctx, url, progress, targetDir)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	if err := os.Rename(tmpPath, cachePath); err != nil {
		return "", fmt.Errorf("failed to move downloaded file to cache: %w", err)
	}
	return cachePath, nil
}

type TarballExtractor struct {
	checker integrityChecker
}

func (e *TarballExtractor) Extract(src string, dest string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read tarball: %w", err)
	}

	if err := e.checker.CheckIntegrity(data); err != nil {
		return fmt.Errorf("integrity check failed: %w", err)
	}

	var decompressor io.ReadCloser
	if e.isGzipCompressed(data) {
		gzr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		decompressor = gzr
	} else if e.isZstdCompressed(data) {
		zr, err := zstd.NewReader(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("failed to create zstd reader: %w", err)
		}
		decompressor = zr.IOReadCloser()
	} else {
		return fmt.Errorf("unsupported compression format: file does not appear to be gzip or zstd compressed")
	}
	defer decompressor.Close()

	tr := tar.NewReader(decompressor)
	if err := ioutil.Untar(dest, tr); err != nil {
		return fmt.Errorf("failed to untar: %w", err)
	}

	return nil
}

// isGzipCompressed checks if the data starts with gzip magic bytes (0x1f 0x8b)
func (e *TarballExtractor) isGzipCompressed(data []byte) bool {
	return len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b
}

// isZstdCompressed checks if the data starts with zstd magic bytes (0x28 0xb5 0x2f 0xfd)
func (e *TarballExtractor) isZstdCompressed(data []byte) bool {
	return len(data) >= 4 && data[0] == 0x28 && data[1] == 0xb5 && data[2] == 0x2f && data[3] == 0xfd
}
