package artifacts

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/klauspost/compress/zstd"
)

const cacheMetadataFile = ".op-deployer-artifacts.json"

type ResolvedArtifacts struct {
	FS           foundry.StatDirFs
	ProjectDir   string
	ArtifactsDir string
	CacheKey     string
	CacheOwned   bool
}

type cacheMetadata struct {
	Version      int    `json:"version"`
	CacheKey     string `json:"cacheKey"`
	Source       string `json:"source"`
	ArtifactsDir string `json:"artifactsDir"`
}

func Resolve(ctx context.Context, loc *Locator, progressor ioutil.Progressor, cacheDir string) (*ResolvedArtifacts, error) {
	if progressor == nil {
		progressor = ioutil.NoopProgressor()
	}

	switch loc.URL.Scheme {
	case "http", "https":
		return resolveHTTP(ctx, loc.URL, progressor, cacheDir)
	case "file":
		return resolveFile(loc.URL)
	case "embedded":
		return resolveEmbedded(cacheDir)
	default:
		return nil, ErrUnsupportedArtifactsScheme
	}
}

func resolveHTTP(ctx context.Context, u *url.URL, progressor ioutil.Progressor, cacheDir string) (*ResolvedArtifacts, error) {
	archivesDir := filepath.Join(cacheDir, "artifacts", "archives")
	cacher := &CachingDownloader{d: new(HTTPDownloader)}
	archivePath, err := cacher.Download(ctx, u.String(), progressor, archivesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to download artifacts: %w", err)
	}

	cacheKey, err := hashFile(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to hash artifact archive: %w", err)
	}

	return resolveCachedProject(cacheDir, cacheKey, "http", func(dest string) error {
		if strings.HasSuffix(archivePath, ".tzst") {
			return extractZstdTarFile(archivePath, dest)
		}
		extractor := &TarballExtractor{checker: new(noopIntegrityChecker)}
		return extractor.Extract(archivePath, dest)
	})
}

func resolveEmbedded(cacheDir string) (*ResolvedArtifacts, error) {
	data, err := embeddedArtifactBytes()
	if err != nil {
		return nil, err
	}

	cacheKey := fmt.Sprintf("%x", sha256.Sum256(data))
	return resolveCachedProject(cacheDir, cacheKey, "embedded", func(dest string) error {
		return extractZstdTar(bytesReader(data), dest)
	})
}

func resolveFile(u *url.URL) (*ResolvedArtifacts, error) {
	projectDir, artifactsDir, err := detectProjectDirs(u.Path, true)
	if err != nil {
		return nil, err
	}
	artifactsFS := os.DirFS(artifactsDir)
	statFS, ok := artifactsFS.(foundry.StatDirFs)
	if !ok {
		return nil, fmt.Errorf("artifact directory %q does not implement StatDirFs", artifactsDir)
	}
	return &ResolvedArtifacts{
		FS:           statFS,
		ProjectDir:   projectDir,
		ArtifactsDir: artifactsDir,
		CacheKey:     fmt.Sprintf("%x", sha256.Sum256([]byte(filepath.Clean(projectDir)))),
		CacheOwned:   false,
	}, nil
}

func resolveCachedProject(cacheDir, cacheKey, source string, extract func(dest string) error) (*ResolvedArtifacts, error) {
	projectsDir := filepath.Join(cacheDir, "artifacts", "projects")
	tmpRoot := filepath.Join(cacheDir, "artifacts", "tmp")
	projectDir := filepath.Join(projectsDir, cacheKey)

	if resolved, err := validateCachedProject(projectDir, cacheKey); err == nil {
		return resolved, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if err := os.MkdirAll(tmpRoot, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create artifact temp dir: %w", err)
	}
	if err := os.MkdirAll(projectsDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create artifact projects dir: %w", err)
	}

	tmpDir, err := os.MkdirTemp(tmpRoot, cacheKey+"-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create artifact extraction dir: %w", err)
	}
	RegisterForCleanup(tmpDir)

	if err := extract(tmpDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("failed to extract artifacts: %w", err)
	}

	projectRoot, artifactsDir, err := detectProjectDirs(tmpDir, false)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("failed to validate extracted artifacts: %w", err)
	}
	if projectRoot != tmpDir {
		normalizedTmp, err := os.MkdirTemp(tmpRoot, cacheKey+"-normalized-*")
		if err != nil {
			_ = os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("failed to create normalized artifact extraction dir: %w", err)
		}
		_ = os.RemoveAll(normalizedTmp)
		if err := os.Rename(projectRoot, normalizedTmp); err != nil {
			_ = os.RemoveAll(tmpDir)
			_ = os.RemoveAll(normalizedTmp)
			return nil, fmt.Errorf("failed to normalize extracted artifacts: %w", err)
		}
		_ = os.RemoveAll(tmpDir)
		tmpDir = normalizedTmp
		artifactsDir = filepath.Join(tmpDir, filepath.Base(artifactsDir))
		RegisterForCleanup(tmpDir)
	}

	metadata := cacheMetadata{
		Version:      1,
		CacheKey:     cacheKey,
		Source:       source,
		ArtifactsDir: filepath.Base(artifactsDir),
	}
	if err := writeCacheMetadata(tmpDir, metadata); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, err
	}

	if err := os.Rename(tmpDir, projectDir); err != nil {
		if _, statErr := os.Stat(projectDir); statErr == nil {
			_ = os.RemoveAll(tmpDir)
			return validateCachedProject(projectDir, cacheKey)
		}
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("failed to publish artifact cache project: %w", err)
	}

	return validateCachedProject(projectDir, cacheKey)
}

func validateCachedProject(projectDir, cacheKey string) (*ResolvedArtifacts, error) {
	metadataPath, err := safeCachePath(projectDir, cacheMetadataFile)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, err
	}
	var metadata cacheMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to read artifact cache metadata: %w", err)
	}
	if metadata.Version != 1 {
		return nil, fmt.Errorf("unsupported artifact cache metadata version %d", metadata.Version)
	}
	if metadata.CacheKey != cacheKey {
		return nil, fmt.Errorf("artifact cache metadata key mismatch: expected %s, got %s", cacheKey, metadata.CacheKey)
	}
	artifactsDir, err := safeCachePath(projectDir, metadata.ArtifactsDir)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(artifactsDir); err != nil {
		return nil, fmt.Errorf("artifact cache entry is missing artifact dir: %w", err)
	}
	artifactsFS := os.DirFS(artifactsDir)
	statFS, ok := artifactsFS.(foundry.StatDirFs)
	if !ok {
		return nil, fmt.Errorf("artifact directory %q does not implement StatDirFs", artifactsDir)
	}
	if _, err := statFS.ReadDir("."); err != nil {
		return nil, fmt.Errorf("artifact cache dir is not readable: %w", err)
	}
	return &ResolvedArtifacts{
		FS:           statFS,
		ProjectDir:   projectDir,
		ArtifactsDir: artifactsDir,
		CacheKey:     cacheKey,
		CacheOwned:   true,
	}, nil
}

func detectProjectDirs(input string, allowRootArtifacts bool) (string, string, error) {
	clean := filepath.Clean(input)
	info, err := os.Stat(clean)
	if err != nil {
		return "", "", fmt.Errorf("failed to stat artifact path %q: %w", input, err)
	}
	if !info.IsDir() {
		return "", "", fmt.Errorf("artifact path %q is not a directory", input)
	}

	base := filepath.Base(clean)
	if base == "forge-artifacts" || base == "out" {
		return filepath.Dir(clean), clean, nil
	}

	for _, dir := range []string{"forge-artifacts", "out"} {
		candidate := filepath.Join(clean, dir)
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return clean, candidate, nil
		}
	}

	if allowRootArtifacts {
		return clean, clean, nil
	}

	return "", "", fmt.Errorf("artifact path %q does not contain forge-artifacts or out", input)
}

func writeCacheMetadata(projectDir string, metadata cacheMetadata) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode artifact cache metadata: %w", err)
	}
	data = append(data, '\n')
	metadataPath, err := safeCachePath(projectDir, cacheMetadataFile)
	if err != nil {
		return err
	}
	if err := os.WriteFile(metadataPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write artifact cache metadata: %w", err)
	}
	return nil
}

func safeCachePath(baseDir string, name string) (string, error) {
	if name == "" || filepath.IsAbs(name) || filepath.Base(name) != name {
		return "", fmt.Errorf("invalid artifact cache path component %q", name)
	}
	if name != cacheMetadataFile && !slices.Contains([]string{"forge-artifacts", "out"}, name) {
		return "", fmt.Errorf("unsupported artifact cache path component %q", name)
	}
	return filepath.Join(baseDir, name), nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func embeddedArtifactBytes() ([]byte, error) {
	f, err := embedDir.Open(filepath.Join("forge-artifacts", embeddedArtifactsZstdShort))
	if err != nil {
		return nil, fmt.Errorf("could not open embedded artifacts %q: %w", embeddedArtifactsZstdShort, err)
	}
	defer f.Close()
	return io.ReadAll(f)
}

func extractZstdTarFile(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("could not open zstd tar file: %w", err)
	}
	defer f.Close()
	return extractZstdTar(f, dest)
}

func extractZstdTar(src io.Reader, dest string) error {
	zr, err := zstd.NewReader(src)
	if err != nil {
		return fmt.Errorf("could not create zstd reader: %w", err)
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	if err := ioutil.Untar(dest, tr); err != nil {
		return fmt.Errorf("failed to untar zstd artifacts: %w", err)
	}
	return nil
}

func bytesReader(data []byte) io.Reader {
	return bytes.NewReader(data)
}
