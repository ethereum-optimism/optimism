package sysgo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	opservice "github.com/ethereum-optimism/optimism/op-service"
)

// RustBinarySpec describes a Rust binary to be built and located.
type RustBinarySpec struct {
	SrcDir  string // directory name relative to monorepo root, e.g. "rollup-boost"
	Package string // cargo package name, e.g. "rollup-boost"
	Binary  string // binary name, e.g. "rollup-boost"
}

// EnsureRustBinary locates or builds a Rust binary as needed.
//
// Env var overrides (suffix derived from binary name, e.g. "rollup-boost" -> "ROLLUP_BOOST"):
//   - RUST_BINARY_PATH_<BINARY>: absolute path to pre-built binary (skips build, must exist)
//   - RUST_SRC_DIR_<BINARY>: overrides SrcDir (absolute path to cargo project root)
//
// Build behavior:
//   - RUST_JIT_BUILD=1: runs cargo build --release (letting cargo handle rebuild detection)
//   - Otherwise: only checks binary exists, errors if missing
func EnsureRustBinary(p devtest.P, spec RustBinarySpec) (string, error) {
	envSuffix := toEnvVarSuffix(spec.Binary)

	// Check for explicit binary path override
	if pathOverride := os.Getenv("RUST_BINARY_PATH_" + envSuffix); pathOverride != "" {
		if _, err := os.Stat(pathOverride); os.IsNotExist(err) {
			return "", fmt.Errorf("%s binary not found at overridden path %s", spec.Binary, pathOverride)
		}
		p.Logger().Info("Using overridden binary path", "binary", spec.Binary, "path", pathOverride)
		return pathOverride, nil
	}

	// Determine source root
	srcRoot, err := resolveSrcRoot(spec.SrcDir, envSuffix)
	if err != nil {
		return "", err
	}

	binaryPath := filepath.Join(srcRoot, "target", "release", spec.Binary)
	jitBuild := os.Getenv("RUST_JIT_BUILD") != ""

	if jitBuild {
		p.Logger().Info("Building Rust binary (JIT)", "binary", spec.Binary, "dir", srcRoot)
		if err := buildRustBinary(p.Ctx(), srcRoot, spec.Package, spec.Binary); err != nil {
			return "", err
		}
	} else {
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			return "", fmt.Errorf("%s binary not found at %s; "+
				"run 'just build-rust-debug' before the test or set RUST_JIT_BUILD=1", spec.Binary, binaryPath)
		}
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
	cmd := exec.CommandContext(ctx, "cargo", "build", "--release", "-p", pkg, "--bin", bin)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
