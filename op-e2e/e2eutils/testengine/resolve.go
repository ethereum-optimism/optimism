package testengine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	opservice "github.com/ethereum-optimism/optimism/op-service"
)

const (
	// requireEngineEnv, when set, makes resolveBinary fail unless the binary's path variable is set,
	// so a CI job whose prebuilt binary was not provisioned fails instead of building one or picking
	// up a local build.
	requireEngineEnv = "REQUIRE_RUST_ENGINE"
	// jitBuildEnv, when set, makes resolveBinary build a binary whose path variable is unset.
	jitBuildEnv = "RUST_JIT_BUILD"
)

// ErrNotConfigured is wrapped by the error ResolveBinary returns when no binary is configured,
// built, or requested to be built. The error's message says how to provide one; callers for which
// the engine is optional can skip on it.
var ErrNotConfigured = errors.New("engine binary not configured")

// binaryPathEnv is the variable that points resolveBinary at a prebuilt binary called name:
// RUST_BINARY_PATH_ followed by name in upper snake case.
func binaryPathEnv(name string) string {
	return "RUST_BINARY_PATH_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
}

// resolveBinary returns the path of the Rust binary called name, built from the cargo package of
// the same name in the monorepo's rust/ workspace. In order of precedence:
//   - RUST_BINARY_PATH_<NAME> (name in upper snake case) set: that path, made absolute, which must
//     be an executable file;
//   - otherwise, REQUIRE_RUST_ENGINE set: an error, so a job that must use a provisioned binary
//     never builds one or picks up a local build;
//   - otherwise, RUST_JIT_BUILD set: the binary `cargo build --release` produces. Unlike the debug
//     JIT build of op-devstack's rustbin this builds the release profile, because the tests that
//     drive the engine execute many blocks, which a debug build runs far slower;
//   - otherwise, an already-built binary: the most recently modified release or debug build;
//   - otherwise: an error wrapping ErrNotConfigured that says how to provide the binary.
//
// Builds are looked up in the target directory `cargo metadata` reports (which honours
// CARGO_TARGET_DIR), both directly under the profile directory and under a target-triple directory
// (the layout CARGO_BUILD_TARGET produces). The returned path always names an existing executable.
func resolveBinary(ctx context.Context, name string) (string, error) {
	pathEnv := binaryPathEnv(name)
	if path := os.Getenv(pathEnv); path != "" {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("%s=%s: %w", pathEnv, path, err)
		}
		if err := checkExecutable(abs); err != nil {
			return "", fmt.Errorf("%s=%s: %w", pathEnv, path, err)
		}
		return abs, nil
	}
	if os.Getenv(requireEngineEnv) != "" {
		return "", fmt.Errorf("%s is set but %s is not: refusing to build or look up %s", requireEngineEnv, pathEnv, name)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root, err := opservice.FindMonorepoRoot(cwd)
	if err != nil {
		return "", err
	}
	rustDir := filepath.Join(root, "rust")

	if os.Getenv(jitBuildEnv) != "" {
		build := exec.CommandContext(ctx, "cargo", "build", "--release", "-p", name, "--bin", name)
		build.Dir = rustDir
		build.Stdout = os.Stderr
		build.Stderr = os.Stderr
		if err := build.Run(); err != nil {
			return "", fmt.Errorf("cargo build --release -p %s (in %s): %w", name, rustDir, err)
		}
		targetDir, err := cargoTargetDir(ctx, rustDir)
		if err != nil {
			return "", err
		}
		path, err := newestBuild(targetDir, name, "release")
		if err != nil {
			return "", fmt.Errorf("cargo build --release -p %s succeeded but %w", name, err)
		}
		return path, nil
	}

	targetDir, err := cargoTargetDir(ctx, rustDir)
	if err != nil {
		return "", notConfigured(name, pathEnv, filepath.Join(rustDir, "target"), err)
	}
	path, err := newestBuild(targetDir, name, "release", "debug")
	if err != nil {
		return "", notConfigured(name, pathEnv, targetDir, err)
	}
	return path, nil
}

// notConfigured reports that no binary called name is configured or built, with the ways to
// provide one; targetDir is where a build lands and cause why none was found there.
func notConfigured(name, pathEnv, targetDir string, cause error) error {
	return fmt.Errorf("%w: %s: set %s to a built binary, build it from rust/ with `just build-%s` "+
		"(it lands at %s), or set %s=1 to build it on demand (%v)",
		ErrNotConfigured, name, pathEnv, name, filepath.Join(targetDir, "release", name), jitBuildEnv, cause)
}

// newestBuild returns the most recently modified executable called name built under targetDir for
// one of profiles, looking in targetDir/<profile> and targetDir/<target triple>/<profile>. On a tie
// the earlier profile wins.
func newestBuild(targetDir, name string, profiles ...string) (string, error) {
	var looked []string
	for _, profile := range profiles {
		looked = append(looked, filepath.Join(targetDir, profile, name), filepath.Join(targetDir, "*", profile, name))
	}
	var newest string
	var newestMod time.Time
	for _, pattern := range looked {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return "", fmt.Errorf("look up %s: %w", pattern, err)
		}
		for _, candidate := range matches {
			if checkExecutable(candidate) != nil {
				continue
			}
			info, err := os.Stat(candidate)
			if err != nil {
				continue
			}
			if newest == "" || info.ModTime().After(newestMod) {
				newest, newestMod = candidate, info.ModTime()
			}
		}
	}
	if newest == "" {
		return "", fmt.Errorf("no executable %s found at %s", name, strings.Join(looked, " or "))
	}
	return newest, nil
}

// checkExecutable returns an error unless path is a regular file with an execute bit set.
func checkExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return errors.New("not executable")
	}
	return nil
}

func cargoTargetDir(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "cargo", "metadata", "--no-deps", "--format-version", "1")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("cargo metadata (in %s): %w", dir, err)
	}
	var meta struct {
		TargetDirectory string `json:"target_directory"`
	}
	if err := json.Unmarshal(out, &meta); err != nil {
		return "", fmt.Errorf("parse cargo metadata: %w", err)
	}
	if meta.TargetDirectory == "" {
		return "", fmt.Errorf("cargo metadata reported no target directory")
	}
	return meta.TargetDirectory, nil
}
