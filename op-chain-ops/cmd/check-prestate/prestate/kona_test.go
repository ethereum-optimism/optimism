package prestate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestFindVersionsReturnsError checks that a failed tag lookup is returned to the caller.
func TestFindVersionsReturnsError(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, stderr, err := runGit("init", "--quiet"); err != nil {
		t.Fatalf("git init: %v (%s)", err, stderr)
	}

	for _, tc := range []struct {
		prestateType string
		tag          string
	}{
		{"cannon64-kona", "kona-client/v0.0.0-missing"},
		{"cannon64-kona-interop", "kona-client/v0.0.0-missing"},
		{"kona-sp1", "kona-sp1-program/v0.0.0-missing"},
	} {
		t.Run(tc.prestateType, func(t *testing.T) {
			p, ok := NewKonaPrestate(tc.prestateType)
			if !ok {
				t.Fatalf("NewKonaPrestate(%q) reported the type as unsupported", tc.prestateType)
			}
			_, _, _, _, err := p.FindVersions(testlog.Logger(t, log.LevelInfo), "0.0.0-missing")
			if err == nil || !strings.Contains(err.Error(), "failed to fetch superchain registry commit") || !strings.Contains(err.Error(), tc.tag) {
				t.Fatalf("got err %v, want a superchain registry commit lookup failure for %s", err, tc.tag)
			}
		})
	}
}

func TestNewKonaPrestateRejectsNonKonaType(t *testing.T) {
	if _, ok := NewKonaPrestate("cannon64"); ok {
		t.Fatal("NewKonaPrestate accepted the non-kona type cannon64")
	}
}

// TestGitlinkCommitFromSubdirectory guards the path resolution of gitlinkCommit.
// The README tells users to run the tool from op-chain-ops/cmd/check-prestate, so a
// cwd-relative lookup silently finds no gitlink and the tool reports the pin as missing.
func TestGitlinkCommitFromSubdirectory(t *testing.T) {
	const submodulePath = "superchain-registry"
	const submoduleSHA = "08d6a44910d75e28a5ee1c7c047d3bdd0bbdbdd2"

	repo := t.TempDir()
	t.Chdir(repo)
	for _, args := range [][]string{
		{"init", "--quiet"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "test"},
		{"update-index", "--add", "--cacheinfo", "160000," + submoduleSHA + "," + submodulePath},
		{"commit", "--quiet", "--no-gpg-sign", "-m", "add gitlink"},
	} {
		if _, stderr, err := runGit(args...); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, stderr)
		}
	}

	subdir := filepath.Join(repo, "op-chain-ops", "cmd", "check-prestate")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(subdir)

	sha, ok := gitlinkCommit("HEAD", submodulePath)
	if !ok {
		t.Fatal("gitlinkCommit found no gitlink when run from a subdirectory")
	}
	if sha != submoduleSHA {
		t.Fatalf("got %s, want %s", sha, submoduleSHA)
	}
}
