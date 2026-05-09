package forge

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

func BuildCacheKey(artifactKey, projectDir string) (string, error) {
	foundryTomlPath, err := safeProjectPath(projectDir, "foundry.toml")
	if err != nil {
		return "", err
	}
	foundryToml, err := os.ReadFile(foundryTomlPath)
	if err != nil {
		return "", fmt.Errorf("failed to read foundry.toml for forge cache key: %w", err)
	}
	h := sha256.New()
	_, _ = h.Write([]byte(artifactKey))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(StandardVersion))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(foundryToml)
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func ProjectOptions(projectDir, cacheDir, buildKey string) []string {
	buildDir := filepath.Join(cacheDir, "forge", "builds", buildKey)
	return []string{
		"--root", projectDir,
		"--cache-path", filepath.Join(buildDir, "cache"),
		"--out", filepath.Join(buildDir, "out"),
	}
}

func safeProjectPath(projectDir string, name string) (string, error) {
	if name == "" || filepath.IsAbs(name) || filepath.Base(name) != name {
		return "", fmt.Errorf("invalid project path component %q", name)
	}
	return filepath.Join(projectDir, name), nil
}
