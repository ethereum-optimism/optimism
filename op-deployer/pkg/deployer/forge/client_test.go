package forge

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type ioStruct struct {
	ID    uint8
	Data  []byte
	Slice []uint32
	Array [3]uint64
}

func TestMinimalSources(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cl := NewClient(PathBinary())
	cl.Wd = projDir(t)

	// Build artifacts
	require.NoError(t, cl.Build(ctx))

	// Then copy them somewhere else
	tmpDir := t.TempDir()
	require.NoError(t, copyDir("testdata/testproject/out", path.Join(tmpDir, "out")))
	require.NoError(t, copyDir("testdata/testproject/cache", path.Join(tmpDir, "cache")))
	require.NoError(t, copyDir("testdata/testproject/script", path.Join(tmpDir, "script")))
	require.NoError(t, copyDir("testdata/testproject/foundry.toml", path.Join(tmpDir, "foundry.toml")))

	// Then see if we can successfully run a script. It must not recompile, since we included the
	// cache.
	cl.Wd = tmpDir
	caller := NewScriptCaller(
		cl,
		"script/Test.s.sol:TestScript",
		"runWithBytes(bytes)",
		&BytesScriptEncoder[ioStruct]{TypeName: "ioStruct"},
		&BytesScriptDecoder[ioStruct]{TypeName: "ioStruct"},
	)
	in := ioStruct{
		ID:    1,
		Data:  []byte{0x01, 0x02, 0x03, 0x04},
		Slice: []uint32{0x01, 0x02, 0x03, 0x04},
		Array: [3]uint64{0x01, 0x02, 0x03},
	}
	out, err := caller(ctx, in)
	require.NoError(t, err)
	require.EqualValues(t, ioStruct{
		ID:    2,
		Data:  in.Data,
		Slice: in.Slice,
		Array: in.Array,
	}, out)
	require.NoDirExists(t, path.Join(tmpDir, "broadcast"), "the run must leave nothing in the project")
}

// TestClient_Smoke smoke tests the Client by running the Version command on it.
func TestClient_Smoke(t *testing.T) {
	bin := PathBinary()
	cl := NewClient(bin)

	version, err := cl.Version(context.Background())
	require.NoError(t, err)
	require.Regexp(t, regexp.MustCompile(`\d+\.\d+\.\d+`), version.Semver)
	require.Regexp(t, regexp.MustCompile(`^[a-f0-9]+$`), version.SHA)
}

func TestClient_OutputRedirection(t *testing.T) {
	bin := PathBinary()
	cl := NewClient(bin)
	cl.Stdout = new(bytes.Buffer)

	_, err := cl.Version(context.Background())
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(cl.Stdout.(*bytes.Buffer).String(), "forge Version"))
}

func TestScriptCaller(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bin := PathBinary()
	cl := NewClient(bin)
	cl.Wd = projDir(t)

	require.NoError(t, cl.Clean(ctx))
	caller := NewScriptCaller(
		cl,
		"script/Test.s.sol:TestScript",
		"runWithBytes(bytes)",
		&BytesScriptEncoder[ioStruct]{TypeName: "ioStruct"},
		&BytesScriptDecoder[ioStruct]{TypeName: "ioStruct"},
	)

	in := ioStruct{
		ID:    1,
		Data:  []byte{0x01, 0x02},
		Slice: []uint32{0x01, 0x02, 0x03, 0x04},
		Array: [3]uint64{0x01, 0x02, 0x03},
	}
	_, err := caller(ctx, in)
	require.ErrorContains(t, err, "compile cache", "a cleaned project has no compile cache")

	require.NoError(t, cl.Build(ctx))
	out, err := caller(ctx, in)
	require.NoError(t, err)
	require.EqualValues(t, ioStruct{
		ID:    2,
		Data:  in.Data,
		Slice: in.Slice,
		Array: in.Array,
	}, out)
}

func TestScriptCaller_RejectsCompile(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// A project whose compile cache does not match its sources makes forge compile.
	tmpDir := t.TempDir()
	cl := NewClient(PathBinary())
	cl.Wd = projDir(t)
	require.NoError(t, cl.Build(ctx))
	require.NoError(t, copyDir("testdata/testproject/script", path.Join(tmpDir, "script")))
	require.NoError(t, copyDir("testdata/testproject/foundry.toml", path.Join(tmpDir, "foundry.toml")))
	require.NoError(t, copyDir("testdata/testproject/cache", path.Join(tmpDir, "cache")))
	cl.Wd = tmpDir

	caller := NewScriptCaller(
		cl,
		"script/Test.s.sol:TestScript",
		"runWithBytes(bytes)",
		&BytesScriptEncoder[ioStruct]{TypeName: "ioStruct"},
		&BytesScriptDecoder[ioStruct]{TypeName: "ioStruct"},
	)
	_, err := caller(ctx, ioStruct{ID: 1})
	require.ErrorContains(t, err, "forge compiled the project")
}

// stubProject returns a project directory with a compile cache, for script runs with a stub forge.
func stubProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cache"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, compileCacheFile), []byte("{}"), 0o644))
	return dir
}

// stubBinary writes an executable shell script and returns it as a Binary.
func stubBinary(t *testing.T, script string) Binary {
	t.Helper()
	p := filepath.Join(t.TempDir(), "stub-forge")
	require.NoError(t, os.WriteFile(p, []byte("#!/bin/sh\n"+script+"\n"), 0o755))
	return StaticBinary(p)
}

// TestClient_TimeoutUnblocksHungCommand reproduces the CI hang shape: a forge
// that never finishes (here: sleeps) while a child process keeps the output
// pipes open even after the parent is killed. Without Client.Timeout the call
// blocks until the process exits on its own; without cmd.WaitDelay it blocks
// on the pipe drain even after the kill.
func TestClient_TimeoutUnblocksHungCommand(t *testing.T) {
	oldDelay := waitDelay
	waitDelay = time.Second
	t.Cleanup(func() { waitDelay = oldDelay })

	cl := NewClient(stubBinary(t, "sleep 60 &\nexec sleep 60"))
	cl.Stdout = io.Discard
	cl.Stderr = io.Discard
	cl.Timeout = 500 * time.Millisecond

	start := time.Now()
	err := cl.Build(context.Background())
	require.Error(t, err)
	require.Less(t, time.Since(start), 10*time.Second)
}

func TestClient_SlowBroadcast(t *testing.T) {
	tests := []struct {
		name     string
		opts     []string
		wantSlow bool
	}{
		{name: "broadcast", opts: []string{"--broadcast"}, wantSlow: true},
		{name: "read-only", opts: nil, wantSlow: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := NewClient(stubBinary(t, `echo "$@"`))
			cl.Wd = stubProject(t)
			cl.Stdout = io.Discard

			out, err := cl.RunScript(context.Background(), "Script.s.sol", "run()", nil, tt.opts...)
			require.NoError(t, err)
			require.Equal(t, tt.wantSlow, strings.Contains(out, "--slow"))
		})
	}
}

func TestClient_RunScriptErrorHasReason(t *testing.T) {
	cl := NewClient(stubBinary(t, `echo "Error: script failed: boom" >&2; exit 1`))
	cl.Wd = stubProject(t)
	cl.Stderr = io.Discard

	_, err := cl.RunScript(context.Background(), "Script.s.sol", "run()", nil)
	require.ErrorContains(t, err, "Error: script failed: boom")
}

func TestClient_WithEnv(t *testing.T) {
	base := NewClient(stubBinary(t, `echo "A=$OP_DEPLOYER_TEST_A B=$OP_DEPLOYER_TEST_B"`))
	base.Wd = stubProject(t)
	base.Stdout = io.Discard
	base = base.WithEnv("OP_DEPLOYER_TEST_A=1")
	withB := base.WithEnv("OP_DEPLOYER_TEST_B=2")

	out, err := withB.RunScript(context.Background(), "Script.s.sol", "run()", nil)
	require.NoError(t, err)
	require.Equal(t, "A=1 B=2\n", out)

	out, err = base.RunScript(context.Background(), "Script.s.sol", "run()", nil)
	require.NoError(t, err)
	require.Equal(t, "A=1 B=\n", out, "WithEnv must not change the client it copies")
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		return copyFile(path, targetPath)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func projDir(t *testing.T) string {
	_, testFilename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	dir := filepath.Join(filepath.Dir(testFilename), "testdata", "testproject")
	absProjDir, err := filepath.Abs(dir)
	require.NoError(t, err)
	return absProjDir
}
