package testengine_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/testengine"
)

// fakeBinary is resolved by these tests; its path variable is independent of the real engine's, so
// the tests run the same with or without a configured engine.
const (
	fakeBinary  = "testengine-fake-binary"
	fakePathEnv = "RUST_BINARY_PATH_TESTENGINE_FAKE_BINARY"
)

// unsetResolverEnv clears every variable the resolver reads for fakeBinary.
func unsetResolverEnv(t *testing.T) {
	t.Setenv(fakePathEnv, "")
	t.Setenv("REQUIRE_RUST_ENGINE", "")
	t.Setenv("RUST_JIT_BUILD", "")
}

// installFakeCargo puts a cargo stub first on PATH. `cargo metadata` reports targetDir as the
// workspace target directory and `cargo build --release` creates targetDir/release/<fakeBinary>.
// The returned path is a log of the stub's invocations.
func installFakeCargo(t *testing.T, targetDir string) (invocations string) {
	return installCargoStub(t, targetDir, targetDir)
}

// installCargoStub is installFakeCargo with the target directory `cargo metadata` reports
// (reportedDir) and the one `cargo build` writes to (buildDir) set apart.
func installCargoStub(t *testing.T, reportedDir, buildDir string) (invocations string) {
	dir := t.TempDir()
	invocations = filepath.Join(dir, "invocations")
	script := `#!/bin/sh
echo "$*" >> "` + invocations + `"
case "$1" in
metadata) printf '{"target_directory":"%s"}\n' "` + reportedDir + `" ;;
build)
  case " $* " in
  *" --release "*) mkdir -p "` + buildDir + `/release" && printf '#!/bin/sh\n' > "` + buildDir + `/release/` + fakeBinary + `" && chmod +x "` + buildDir + `/release/` + fakeBinary + `" ;;
  esac ;;
esac
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cargo"), []byte(script), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return invocations
}

// writeBuilt creates an executable fakeBinary at targetDir/rel, last modified at mtime.
func writeBuilt(t *testing.T, targetDir, rel string, mtime time.Time) string {
	path := filepath.Join(targetDir, rel, fakeBinary)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755))
	require.NoError(t, os.Chtimes(path, mtime, mtime))
	return path
}

// requireNoBuild asserts the cargo stub was never asked to build.
func requireNoBuild(t *testing.T, invocations string) {
	raw, err := os.ReadFile(invocations)
	if os.IsNotExist(err) {
		return
	}
	require.NoError(t, err)
	require.NotContains(t, string(raw), "build", "must not run cargo build")
}

func TestResolveBinaryPathOverride(t *testing.T) {
	unsetResolverEnv(t)
	bin := filepath.Join(t.TempDir(), fakeBinary)
	require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755))
	t.Setenv(fakePathEnv, bin)

	got, err := testengine.ResolveBinaryNamed(context.Background(), fakeBinary)
	require.NoError(t, err)
	require.Equal(t, bin, got)
}

// TestResolveBinaryPathOverrideRelative checks that a relative override resolves against the
// working directory: a bare name must not turn into a PATH lookup when the binary is spawned.
func TestResolveBinaryPathOverrideRelative(t *testing.T) {
	unsetResolverEnv(t)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, fakeBinary), []byte("#!/bin/sh\n"), 0o755))
	t.Chdir(dir)
	t.Setenv(fakePathEnv, fakeBinary)

	got, err := testengine.ResolveBinaryNamed(context.Background(), fakeBinary)
	require.NoError(t, err)
	require.True(t, filepath.IsAbs(got), "resolved %q is not absolute", got)
	require.FileExists(t, got)
	require.Equal(t, fakeBinary, filepath.Base(got))
}

func TestResolveBinaryPathOverrideNotExecutable(t *testing.T) {
	unsetResolverEnv(t)
	bin := filepath.Join(t.TempDir(), fakeBinary)
	require.NoError(t, os.WriteFile(bin, []byte("not a program"), 0o644))
	t.Setenv(fakePathEnv, bin)

	_, err := testengine.ResolveBinaryNamed(context.Background(), fakeBinary)
	require.ErrorContains(t, err, fakePathEnv)
	require.ErrorContains(t, err, "not executable")
}

func TestResolveBinaryPathOverrideMissing(t *testing.T) {
	unsetResolverEnv(t)
	t.Setenv(fakePathEnv, filepath.Join(t.TempDir(), fakeBinary))

	_, err := testengine.ResolveBinaryNamed(context.Background(), fakeBinary)
	require.ErrorContains(t, err, fakePathEnv)
	require.NotErrorIs(t, err, testengine.ErrNotConfigured)
}

// TestResolveBinaryRequiredButUnset checks that REQUIRE_RUST_ENGINE without a path is a hard error
// — never ErrNotConfigured, which tests may skip on — even when a build is requested or a built
// binary is at hand, so CI cannot fall back to building or to a stale local binary.
func TestResolveBinaryRequiredButUnset(t *testing.T) {
	unsetResolverEnv(t)
	t.Setenv("REQUIRE_RUST_ENGINE", "1")
	t.Setenv("RUST_JIT_BUILD", "1")
	targetDir := t.TempDir()
	writeBuilt(t, targetDir, "release", time.Now())
	invocations := installFakeCargo(t, targetDir)

	_, err := testengine.ResolveBinaryNamed(context.Background(), fakeBinary)
	require.ErrorContains(t, err, "REQUIRE_RUST_ENGINE")
	require.ErrorContains(t, err, fakePathEnv)
	require.NotErrorIs(t, err, testengine.ErrNotConfigured)
	require.NoFileExists(t, invocations, "must not run cargo")
}

// TestResolveBinaryUnconfigured checks that with nothing configured or built the resolver fails at
// once with ErrNotConfigured and instructions, instead of compiling the engine inside the test run.
func TestResolveBinaryUnconfigured(t *testing.T) {
	unsetResolverEnv(t)
	targetDir := t.TempDir()
	invocations := installFakeCargo(t, targetDir)

	_, err := testengine.ResolveBinaryNamed(context.Background(), fakeBinary)
	require.ErrorIs(t, err, testengine.ErrNotConfigured)
	require.ErrorContains(t, err, fakePathEnv)
	require.ErrorContains(t, err, "RUST_JIT_BUILD=1")
	require.ErrorContains(t, err, "just build-"+fakeBinary)
	require.ErrorContains(t, err, filepath.Join(targetDir, "release", fakeBinary))
	requireNoBuild(t, invocations)
}

// TestResolveBinaryWithoutCargo checks that a checkout without a Rust toolchain reports the engine
// as not configured, so callers that treat it as optional can skip.
func TestResolveBinaryWithoutCargo(t *testing.T) {
	unsetResolverEnv(t)
	t.Setenv("PATH", t.TempDir())

	_, err := testengine.ResolveBinaryNamed(context.Background(), fakeBinary)
	require.ErrorIs(t, err, testengine.ErrNotConfigured)
}

// TestResolveBinaryPrebuilt checks that an already-built binary in the target directory is used
// without building.
func TestResolveBinaryPrebuilt(t *testing.T) {
	unsetResolverEnv(t)
	targetDir := t.TempDir()
	want := writeBuilt(t, targetDir, "release", time.Now())
	invocations := installFakeCargo(t, targetDir)

	got, err := testengine.ResolveBinaryNamed(context.Background(), fakeBinary)
	require.NoError(t, err)
	require.Equal(t, want, got)
	requireNoBuild(t, invocations)
}

// TestResolveBinaryPrebuiltNewest checks that of several builds the most recently modified wins, so
// a stale build of one profile does not shadow a fresh one of the other.
func TestResolveBinaryPrebuiltNewest(t *testing.T) {
	unsetResolverEnv(t)
	targetDir := t.TempDir()
	now := time.Now()
	writeBuilt(t, targetDir, "release", now.Add(-time.Hour))
	debug := writeBuilt(t, targetDir, "debug", now)
	installFakeCargo(t, targetDir)

	got, err := testengine.ResolveBinaryNamed(context.Background(), fakeBinary)
	require.NoError(t, err)
	require.Equal(t, debug, got)
}

// TestResolveBinaryPrebuiltTargetTriple checks the layout CARGO_BUILD_TARGET produces, with the
// profile directory under a target-triple directory.
func TestResolveBinaryPrebuiltTargetTriple(t *testing.T) {
	unsetResolverEnv(t)
	targetDir := t.TempDir()
	want := writeBuilt(t, targetDir, filepath.Join("x86_64-unknown-linux-gnu", "release"), time.Now())
	installFakeCargo(t, targetDir)

	got, err := testengine.ResolveBinaryNamed(context.Background(), fakeBinary)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

// TestResolveBinaryJITBuild checks that an opted-in build returns the binary from the target
// directory cargo reports, which CARGO_TARGET_DIR or a cargo config can move away from rust/target.
func TestResolveBinaryJITBuild(t *testing.T) {
	unsetResolverEnv(t)
	t.Setenv("RUST_JIT_BUILD", "1")
	targetDir := t.TempDir()
	invocations := installFakeCargo(t, targetDir)

	got, err := testengine.ResolveBinaryNamed(context.Background(), fakeBinary)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(targetDir, "release", fakeBinary), got)
	raw, err := os.ReadFile(invocations)
	require.NoError(t, err)
	require.Contains(t, string(raw), "build --release -p "+fakeBinary)
}

// TestResolveBinaryJITBuildMissingOutput checks that a build which leaves no binary where the
// resolver looks is an error naming the places it looked.
func TestResolveBinaryJITBuildMissingOutput(t *testing.T) {
	unsetResolverEnv(t)
	t.Setenv("RUST_JIT_BUILD", "1")
	elsewhere := t.TempDir()
	installCargoStub(t, elsewhere, t.TempDir())

	_, err := testengine.ResolveBinaryNamed(context.Background(), fakeBinary)
	require.Error(t, err)
	require.NotErrorIs(t, err, testengine.ErrNotConfigured)
	require.ErrorContains(t, err, filepath.Join(elsewhere, "release", fakeBinary))
}
