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

	// Then see if we can successfully run a script
	cl.Wd = tmpDir
	caller := NewScriptCaller(
		cl,
		"script/Test.s.sol:TestScript",
		"runWithBytes(bytes)",
		&BytesScriptEncoder[ioStruct]{TypeName: "ioStruct"},
		&BytesScriptDecoder[ioStruct]{TypeName: "ioStruct"},
	)
	// It should not recompile since we included the cache.
	in := ioStruct{
		ID:    1,
		Data:  []byte{0x01, 0x02, 0x03, 0x04},
		Slice: []uint32{0x01, 0x02, 0x03, 0x04},
		Array: [3]uint64{0x01, 0x02, 0x03},
	}
	out, changed, err := caller(ctx, in)
	require.NoError(t, err)
	require.False(t, changed)
	require.EqualValues(t, ioStruct{
		ID:    2,
		Data:  in.Data,
		Slice: in.Slice,
		Array: in.Array,
	}, out)
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
	out, recompiled, err := caller(context.Background(), in)
	require.NoError(t, err)
	require.True(t, recompiled)
	require.EqualValues(t, ioStruct{
		ID:    2,
		Data:  in.Data,
		Slice: in.Slice,
		Array: in.Array,
	}, out)
	out, recompiled, err = caller(context.Background(), in)
	require.NoError(t, err)
	require.False(t, recompiled)
	require.EqualValues(t, ioStruct{
		ID:    2,
		Data:  in.Data,
		Slice: in.Slice,
		Array: in.Array,
	}, out)
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
		slow     bool
		opts     []string
		wantSlow bool
	}{
		{name: "broadcast with slow", slow: true, opts: []string{"--broadcast"}, wantSlow: true},
		{name: "read-only with slow", slow: true, opts: nil, wantSlow: false},
		{name: "broadcast without slow", slow: false, opts: []string{"--broadcast"}, wantSlow: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := NewClient(stubBinary(t, `echo "$@"`))
			cl.Stdout = io.Discard
			cl.SlowBroadcast = tt.slow

			out, err := cl.RunScript(context.Background(), "Script.s.sol", "run()", nil, tt.opts...)
			require.NoError(t, err)
			require.Equal(t, tt.wantSlow, strings.Contains(out, "--slow"))
		})
	}
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
