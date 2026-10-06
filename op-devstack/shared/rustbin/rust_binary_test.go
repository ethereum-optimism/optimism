package rustbin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSelectNewestBinaryPrefersFreshReleaseOverStaleDebug(t *testing.T) {
	targetDir := t.TempDir()
	releaseBin := filepath.Join(targetDir, "release", "op-reth")
	debugBin := filepath.Join(targetDir, "debug", "op-reth")
	writeStubBinary(t, debugBin, time.Now().Add(-time.Hour))
	writeStubBinary(t, releaseBin, time.Now())

	got, err := selectNewestBinary(targetDir, "op-reth")
	if err != nil {
		t.Fatalf("selectNewestBinary: %v", err)
	}
	if got != releaseBin {
		t.Fatalf("expected freshest binary %q, got %q", releaseBin, got)
	}
}

func TestSelectNewestBinaryPrefersFreshDebugOverStaleRelease(t *testing.T) {
	targetDir := t.TempDir()
	releaseBin := filepath.Join(targetDir, "release", "op-reth")
	debugBin := filepath.Join(targetDir, "debug", "op-reth")
	writeStubBinary(t, releaseBin, time.Now().Add(-time.Hour))
	writeStubBinary(t, debugBin, time.Now())

	got, err := selectNewestBinary(targetDir, "op-reth")
	if err != nil {
		t.Fatalf("selectNewestBinary: %v", err)
	}
	if got != debugBin {
		t.Fatalf("expected freshest binary %q, got %q", debugBin, got)
	}
}

func TestSelectNewestBinaryMissing(t *testing.T) {
	if _, err := selectNewestBinary(t.TempDir(), "op-reth"); err == nil {
		t.Fatal("expected error when no binary is present")
	}
}

func writeStubBinary(t *testing.T, path string, mod time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("stub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mod, mod); err != nil {
		t.Fatal(err)
	}
}

func TestBuildRustBinaryFeatures(t *testing.T) {
	for _, tc := range []struct {
		name     string
		features []string
		want     string
	}{
		{name: "default", want: "build\n-p\npackage\n--bin\nbinary\n"},
		{name: "test-config", features: []string{"kona-sp1-ethereum-client-utils/test-config-fallback"}, want: "build\n-p\npackage\n--bin\nbinary\n--features\nkona-sp1-ethereum-client-utils/test-config-fallback\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			argsPath := filepath.Join(root, "args")
			script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$TEST_CARGO_ARGS\"\n"
			if err := os.WriteFile(filepath.Join(root, "cargo"), []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("TEST_CARGO_ARGS", argsPath)
			if err := buildRustBinary(context.Background(), root, "package", "binary", tc.features); err != nil {
				t.Fatal(err)
			}
			args, err := os.ReadFile(argsPath)
			if err != nil {
				t.Fatal(err)
			}
			if string(args) != tc.want {
				t.Fatalf("unexpected Cargo arguments: got %q, want %q", strings.Split(string(args), "\n"), strings.Split(tc.want, "\n"))
			}
		})
	}
}
