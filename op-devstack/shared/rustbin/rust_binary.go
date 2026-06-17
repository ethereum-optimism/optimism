package rustbin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"

	opservice "github.com/ethereum-optimism/optimism/op-service"
)

// Spec describes a Rust binary to be built and located.
type Spec struct {
	SrcDir  string // directory name relative to monorepo root, e.g. "rollup-boost"
	Package string // cargo package name, e.g. "rollup-boost"
	Binary  string // binary name, e.g. "rollup-boost"
}

// EnsureExists locates or builds a Rust binary as needed.
//
// Env var overrides (suffix derived from binary name, e.g. "rollup-boost" -> "ROLLUP_BOOST"):
//   - RUST_BINARY_PATH_<BINARY>: absolute path to pre-built binary (skips build, must exist)
//   - RUST_SRC_DIR_<BINARY>: overrides SrcDir (absolute path to cargo project root)
//
// Build behavior:
//   - RUST_JIT_BUILD=1: runs cargo build in debug mode (letting cargo handle rebuild detection)
//   - Otherwise: only checks binary exists, errors if missing
func (s Spec) EnsureExists(ctx context.Context, logger log.Logger) (string, error) {
	envSuffix := toEnvVarSuffix(s.Binary)

	// Check for explicit binary path override
	if pathOverride := os.Getenv("RUST_BINARY_PATH_" + envSuffix); pathOverride != "" {
		if _, err := os.Stat(pathOverride); os.IsNotExist(err) {
			return "", fmt.Errorf("%s binary not found at overridden path %s", s.Binary, pathOverride)
		}
		logger.Info("Using overridden binary path", "binary", s.Binary, "path", pathOverride)
		return pathOverride, nil
	}

	// Determine source root
	srcRoot, err := resolveSrcRoot(s.SrcDir, envSuffix)
	if err != nil {
		return "", err
	}

	jitBuild := os.Getenv("RUST_JIT_BUILD") != ""

	if jitBuild {
		logger.Info("Building Rust binary (JIT)", "binary", s.Binary, "dir", srcRoot)
		if err := buildRustBinary(ctx, srcRoot, s.Package, s.Binary); err != nil {
			return "", err
		}
	}

	binaryPath, err := resolveBuiltRustBinaryPath(srcRoot, s.Binary)
	if err != nil {
		return "", fmt.Errorf("%s binary not found; run 'cd %s && just build-%s-debug' (or just build-%s for release) or set RUST_JIT_BUILD=1: %w",
			s.Binary, s.SrcDir, s.Binary, s.Binary, err)
	}
	return binaryPath, nil
}

// resolveSrcRoot determines the cargo project root, checking for env var override first.
func resolveSrcRoot(defaultSrcDir, envSuffix string) (string, error) {
	if srcOverride := os.Getenv("RUST_SRC_DIR_" + envSuffix); srcOverride != "" {
		return srcOverride, nil
	}

	rootDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	monorepoRoot, err := opservice.FindMonorepoRoot(rootDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(monorepoRoot, defaultSrcDir), nil
}

// toEnvVarSuffix converts a binary name to an env var suffix.
// e.g. "rollup-boost" -> "ROLLUP_BOOST"
func toEnvVarSuffix(binary string) string {
	return strings.ToUpper(strings.ReplaceAll(binary, "-", "_"))
}

func buildRustBinary(ctx context.Context, root, pkg, bin string) error {
	cmd := exec.CommandContext(ctx, "cargo", "build", "-p", pkg, "--bin", bin)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type cargoMetadata struct {
	TargetDirectory string `json:"target_directory"`
}

func resolveBuiltRustBinaryPath(srcRoot, binary string) (string, error) {
	targetDir, err := cargoTargetDirectory(srcRoot)
	if err != nil {
		return "", err
	}
	return selectNewestBinary(targetDir, binary)
}

// selectNewestBinary locates the built binary under targetDir, preferring the
// most recently modified when several profiles or target triples are present.
// Picking by mtime avoids silently shadowing a freshly built release binary with
// a stale debug leftover (alphabetical order would prefer "debug" over "release").
func selectNewestBinary(targetDir, binary string) (string, error) {
	candidates := []string{
		filepath.Join(targetDir, "release", binary),
		filepath.Join(targetDir, "debug", binary),
	}
	globMatches, err := filepath.Glob(filepath.Join(targetDir, "*", "release", binary))
	if err == nil {
		candidates = append(candidates, globMatches...)
	}

	seen := make(map[string]struct{}, len(candidates))
	var newest string
	var newestMod time.Time
	for _, candidate := range candidates {
		if _, dup := seen[candidate]; dup {
			continue
		}
		seen[candidate] = struct{}{}
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		if newest == "" || info.ModTime().After(newestMod) {
			newest = candidate
			newestMod = info.ModTime()
		}
	}

	if newest == "" {
		return "", fmt.Errorf("no built binary found under target dir %s", targetDir)
	}
	return newest, nil
}

func cargoTargetDirectory(srcRoot string) (string, error) {
	cmd := exec.Command("cargo", "metadata", "--no-deps", "--format-version", "1")
	cmd.Dir = srcRoot
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("cargo metadata: %w", err)
	}

	var meta cargoMetadata
	if err := json.Unmarshal(out, &meta); err != nil {
		return "", fmt.Errorf("parse cargo metadata: %w", err)
	}
	if meta.TargetDirectory == "" {
		return "", fmt.Errorf("cargo metadata returned empty target directory")
	}
	return meta.TargetDirectory, nil
}
