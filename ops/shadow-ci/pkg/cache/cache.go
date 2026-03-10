package cache

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// Resolution is the result of resolving a category's cache.
type Resolution struct {
	Category string
	CacheKey string
	Hit      bool   // cache directory exists with matching key
	CacheDir string // path to cached artifacts (empty if miss)
}

// Resolver computes cache keys and manages the build cache directory.
type Resolver struct {
	repoRoot string // absolute path to repo root
	cacheDir string // absolute path to cache base dir (e.g. /tmp/shadow-ci-cache)
}

// NewResolver creates a cache resolver.
func NewResolver(repoRoot, cacheDir string) *Resolver {
	return &Resolver{repoRoot: repoRoot, cacheDir: cacheDir}
}

// ComputeKey computes a content-addressed cache key for a category.
// The key is sha256(sorted git tree hashes of input paths + mise.toml hash).
func (r *Resolver) ComputeKey(category string, cat model.JobCategoryConfig) (string, error) {
	inputs := cat.CacheInputs
	if len(inputs) == 0 {
		inputs = cat.TriggerPaths
	}
	if len(inputs) == 0 {
		return "", fmt.Errorf("category %s has no cache_inputs or trigger_paths", category)
	}

	h := sha256.New()

	// Sort inputs for determinism.
	sorted := make([]string, len(inputs))
	copy(sorted, inputs)
	sort.Strings(sorted)

	for _, input := range sorted {
		hash, err := r.gitTreeHash(input)
		if err != nil {
			return "", fmt.Errorf("hashing %s: %w", input, err)
		}
		fmt.Fprintf(h, "%s:%s\n", input, hash)
	}

	// Include mise.toml for toolchain versioning.
	miseHash, err := r.gitFileHash("mise.toml")
	if err != nil {
		miseHash = "none"
	}
	fmt.Fprintf(h, "mise.toml:%s\n", miseHash)

	return fmt.Sprintf("%x", h.Sum(nil))[:16], nil
}

// Resolve checks if a category has a cache key match.
// On a hit, the caller should Restore, then Verify workspace_paths exist.
func (r *Resolver) Resolve(category string, cat model.JobCategoryConfig) (*Resolution, error) {
	key, err := r.ComputeKey(category, cat)
	if err != nil {
		return &Resolution{Category: category}, err
	}

	res := &Resolution{Category: category, CacheKey: key}

	catCacheDir := filepath.Join(r.cacheDir, category)
	keyFile := filepath.Join(catCacheDir, "cache.key")

	existingKey, err := os.ReadFile(keyFile)
	if err != nil || strings.TrimSpace(string(existingKey)) != key {
		return res, nil
	}

	res.Hit = true
	res.CacheDir = catCacheDir
	return res, nil
}

// Restore copies cached artifacts from the cache directory to the repo.
func (r *Resolver) Restore(category string, cat model.JobCategoryConfig) error {
	catCacheDir := filepath.Join(r.cacheDir, category, "artifacts")
	for _, wsPath := range cat.WorkspacePaths {
		src := filepath.Join(catCacheDir, wsPath)
		dst := filepath.Join(r.repoRoot, wsPath)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		os.MkdirAll(filepath.Dir(dst), 0o755)
		os.RemoveAll(dst)
		cmd := exec.Command("cp", "-r", src, dst)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("restoring %s: %w", wsPath, err)
		}
	}
	return nil
}

// Save stores build artifacts in the cache directory.
func (r *Resolver) Save(category string, cat model.JobCategoryConfig, key string) error {
	catCacheDir := filepath.Join(r.cacheDir, category)
	artifactsDir := filepath.Join(catCacheDir, "artifacts")
	os.MkdirAll(artifactsDir, 0o755)

	for _, wsPath := range cat.WorkspacePaths {
		src := filepath.Join(r.repoRoot, wsPath)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		dst := filepath.Join(artifactsDir, wsPath)
		os.MkdirAll(filepath.Dir(dst), 0o755)
		os.RemoveAll(dst)
		cmd := exec.Command("cp", "-r", src, dst)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("saving %s: %w", wsPath, err)
		}
	}

	return os.WriteFile(filepath.Join(catCacheDir, "cache.key"), []byte(key), 0o644)
}

// Verify checks that all workspace_paths exist after a cache restore.
// This replaces the hand-written verify_command field — the framework
// knows what it cached and can check for itself.
func Verify(repoRoot string, cat model.JobCategoryConfig) error {
	for _, wsPath := range cat.WorkspacePaths {
		fullPath := filepath.Join(repoRoot, wsPath)
		if _, err := os.Stat(fullPath); err != nil {
			return fmt.Errorf("workspace path %s: %w", wsPath, err)
		}
	}
	return nil
}

// gitTreeHash returns the git tree hash for a path (file or directory).
func (r *Resolver) gitTreeHash(path string) (string, error) {
	fullPath := filepath.Join(r.repoRoot, path)
	info, err := os.Stat(fullPath)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", path, err)
	}

	if info.IsDir() {
		cmd := exec.Command("git", "rev-parse", "HEAD:"+strings.TrimSuffix(path, "/"))
		cmd.Dir = r.repoRoot
		out, err := cmd.Output()
		if err != nil {
			return r.hashDir(path)
		}
		return strings.TrimSpace(string(out)), nil
	}

	return r.gitFileHash(path)
}

// gitFileHash returns the git blob hash for a tracked file.
func (r *Resolver) gitFileHash(path string) (string, error) {
	cmd := exec.Command("git", "hash-object", path)
	cmd.Dir = r.repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("hash-object %s: %w", path, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// hashDir computes a hash over directory contents for untracked dirs.
func (r *Resolver) hashDir(path string) (string, error) {
	h := sha256.New()
	err := filepath.Walk(filepath.Join(r.repoRoot, path), func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(r.repoRoot, p)
		fmt.Fprintf(h, "path:%s\n", rel)
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := io.Copy(h, f); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
