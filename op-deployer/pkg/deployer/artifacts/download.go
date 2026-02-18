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
	"path/filepath"
	"strings"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/httputil"
	"github.com/klauspost/compress/zstd"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
)

var ErrUnsupportedArtifactsScheme = errors.New("unsupported artifacts URL scheme")

// ExtractToFreshBundle extracts artifacts from the given locator into a fresh per-client
// directory, reusing existing extraction codepaths. Returns the bundle directory path
// (parent of forge-artifacts) and registers it for cleanup.
func ExtractToFreshBundle(ctx context.Context, loc *Locator, progressor ioutil.Progressor, targetDir string) (bundleDir string, err error) {
	if progressor == nil {
		progressor = ioutil.NoopProgressor()
	}

	u := loc.URL
	checker := new(noopIntegrityChecker)

	switch u.Scheme {
	case "embedded":
		parentDir, err := os.MkdirTemp(targetDir, "op-deployer-bundle-*")
		if err != nil {
			return "", fmt.Errorf("failed to create temp dir: %w", err)
		}
		RegisterForCleanup(parentDir)
		bundleDir, err := ExtractEmbeddedForForge(parentDir)
		if err != nil {
			return "", fmt.Errorf("failed to extract embedded artifacts: %w", err)
		}
		return bundleDir, nil
	case "http", "https":
		return extractHTTPToFreshBundle(ctx, u, progressor, checker, targetDir)
	case "file":
		return copyFileSchemeToFreshBundle(u.Path, targetDir)
	default:
		return "", ErrUnsupportedArtifactsScheme
	}
}

func extractHTTPToFreshBundle(ctx context.Context, u *url.URL, progressor ioutil.Progressor, checker integrityChecker, targetDir string) (string, error) {
	cacher := &CachingDownloader{d: new(HTTPDownloader)}
	tarballPath, err := cacher.Download(ctx, u.String(), progressor, targetDir)
	if err != nil {
		return "", fmt.Errorf("failed to download artifacts: %w", err)
	}

	tmpDir, err := os.MkdirTemp(targetDir, "op-deployer-bundle-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	RegisterForCleanup(tmpDir)

	if strings.HasSuffix(tarballPath, ".tzst") {
		_, err := ExtractFromFile(tmpDir, tarballPath)
		if err != nil {
			return "", fmt.Errorf("failed to extract tarball: %w", err)
		}
		// ExtractFromFile creates tmpDir/bundle-* with "out" inside; bundle dir is parent of artifacts
		entries, err := os.ReadDir(tmpDir)
		if err != nil {
			return "", fmt.Errorf("failed to read extracted dir: %w", err)
		}
		for _, e := range entries {
			if e.IsDir() && strings.HasPrefix(e.Name(), "bundle-") {
				return filepath.Join(tmpDir, e.Name()), nil
			}
		}
		return "", fmt.Errorf("bundle directory not found after .tzst extraction")
	}

	extractor := &TarballExtractor{checker: checker}
	if err := extractor.Extract(tarballPath, tmpDir); err != nil {
		return "", fmt.Errorf("failed to extract tarball: %w", err)
	}
	// .tgz tarballs have forge-artifacts at top level; bundle dir is tmpDir
	return tmpDir, nil
}

func copyFileSchemeToFreshBundle(srcPath, targetDir string) (string, error) {
	if targetDir == "" {
		targetDir = os.TempDir()
	}
	dstDir, err := os.MkdirTemp(targetDir, "op-deployer-bundle-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	RegisterForCleanup(dstDir)
	// Try hard links first (cp -al style); fall back to copy if hard links fail (e.g. cross-filesystem)
	if err := linkDirContents(srcPath, dstDir); err != nil {
		_ = os.RemoveAll(dstDir)
		_ = os.MkdirAll(dstDir, 0o755)
		if err := copyDirContents(srcPath, dstDir); err != nil {
			return "", fmt.Errorf("failed to copy file scheme artifacts: %w", err)
		}
	}
	return dstDir, nil
}

// linkDirContents recursively hard-links directory contents. Returns an error if any link fails
// (e.g. cross-filesystem, or on Windows). Caller should fall back to copyDirContents.
func linkDirContents(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := os.MkdirAll(dstPath, 0o755); err != nil {
				return err
			}
			if err := linkDirContents(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := os.Link(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyDirContents(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := os.MkdirAll(dstPath, 0o755); err != nil {
				return err
			}
			if err := copyDirContents(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

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
	tmpDir, err := os.MkdirTemp(targetDir, "bundle-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Register for automatic cleanup on process exit
	RegisterForCleanup(tmpDir)
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
