package freshness

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
	"github.com/ethereum-optimism/optimism/ops/checks/policy"
)

func testPolicy() *policy.Policy {
	return &policy.Policy{
		Freshness: policy.FreshnessConfig{
			StaleMultiplier: 0.3,
			MaxAgeDays:      30,
		},
	}
}

// TestNop — no-op checker always returns 1.0.
func TestNop(t *testing.T) {
	c := Nop()
	edge := &graph.Edge{Properties: map[string]any{"source_sha": "deadbeef"}}
	if got := c.Assess(edge); got != 1.0 {
		t.Errorf("Nop().Assess = %f, want 1.0", got)
	}
}

// TestAssess_NoStamps — legacy edge with no stamps returns 1.0.
func TestAssess_NoStamps(t *testing.T) {
	root := t.TempDir()
	c := New(root, testPolicy())
	edge := &graph.Edge{}
	if got := c.Assess(edge); got != 1.0 {
		t.Errorf("unstamped edge: Assess = %f, want 1.0", got)
	}
}

// TestAssess_ShaMatches — stamped SHA equals current content SHA.
func TestAssess_ShaMatches(t *testing.T) {
	root := t.TempDir()
	setupContractsFile(t, root, "src/L1/X.sol", []byte("contract X {}\n"))

	// Compute the real current SHA and stamp it.
	sha, err := HashFile(filepath.Join(root, "packages/contracts-bedrock/src/L1/X.sol"))
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	edge := &graph.Edge{
		From: "sol:test/L1/X.t.sol",
		To:   "sol:src/L1/X.sol",
		Properties: map[string]any{
			"source_sha": sha,
		},
	}
	c := New(root, testPolicy())
	if got := c.Assess(edge); got != 1.0 {
		t.Errorf("matching SHA: Assess = %f, want 1.0", got)
	}
}

// TestAssess_ShaMismatch — stamped SHA differs from current content.
func TestAssess_ShaMismatch(t *testing.T) {
	root := t.TempDir()
	setupContractsFile(t, root, "src/L1/X.sol", []byte("contract X { uint256 newField; }\n"))

	edge := &graph.Edge{
		To: "sol:src/L1/X.sol",
		Properties: map[string]any{
			"source_sha": "1111111111111111111111111111111111111111",
		},
	}
	c := New(root, testPolicy())
	if got := c.Assess(edge); got != 0.3 {
		t.Errorf("mismatched SHA: Assess = %f, want 0.3", got)
	}
}

// TestAssess_SourceMissing — file stamped but no longer on disk.
func TestAssess_SourceMissing(t *testing.T) {
	root := t.TempDir()
	edge := &graph.Edge{
		To: "sol:src/L1/Deleted.sol",
		Properties: map[string]any{
			"source_sha": "abc",
		},
	}
	c := New(root, testPolicy())
	if got := c.Assess(edge); got != 0.3 {
		t.Errorf("missing file: Assess = %f, want 0.3 (stale)", got)
	}
}

// TestAssess_TimeDecay — edge with no SHAs but timestamp older than
// max_age_days returns stale_multiplier.
func TestAssess_TimeDecay(t *testing.T) {
	root := t.TempDir()

	old := time.Now().UTC().Add(-60 * 24 * time.Hour).Format(time.RFC3339)
	edge := &graph.Edge{
		Properties: map[string]any{
			"generated_at": old,
		},
	}
	c := New(root, testPolicy())
	if got := c.Assess(edge); got != 0.3 {
		t.Errorf("old edge no shas: Assess = %f, want 0.3", got)
	}

	recent := time.Now().UTC().Add(-1 * 24 * time.Hour).Format(time.RFC3339)
	edge.Properties["generated_at"] = recent
	if got := c.Assess(edge); got != 1.0 {
		t.Errorf("recent edge no shas: Assess = %f, want 1.0", got)
	}
}

// TestAssess_GoFileNodeResolves — a go:<module>/<rel>.go node ID
// resolves to a filesystem path via the checker's go.mod-derived
// module prefix, and SHA matching works the same as for Solidity.
func TestAssess_GoFileNodeResolves(t *testing.T) {
	root := t.TempDir()

	// Write a go.mod establishing the module path.
	goMod := "module github.com/acme/widgets\n\ngo 1.24.0\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	// Write the source file under the module prefix.
	dir := filepath.Join(root, "widget", "core")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	filePath := filepath.Join(dir, "thing.go")
	if err := os.WriteFile(filePath, []byte("package core\n\nvar X = 1\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Stamp a matching SHA on an edge pointing at this file's node.
	sha, err := HashFile(filePath)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	edge := &graph.Edge{
		From: "go:github.com/acme/widgets/test/thingtest",
		To:   "go:github.com/acme/widgets/widget/core/thing.go",
		Properties: map[string]any{
			"source_sha": sha,
		},
	}

	c := New(root, testPolicy())
	if got := c.Assess(edge); got != 1.0 {
		t.Errorf("matching Go file SHA: Assess = %f, want 1.0", got)
	}

	// Mismatch: rewrite the file's content, SHA no longer matches stamp.
	if err := os.WriteFile(filePath, []byte("package core\n\nvar X = 2\n"), 0o644); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	c2 := New(root, testPolicy()) // fresh checker — avoid cache hit from prior hash
	if got := c2.Assess(edge); got != 0.3 {
		t.Errorf("mismatched Go file SHA: Assess = %f, want 0.3", got)
	}
}

// TestAssess_GoFileNodeWithoutGoMod — if go.mod is missing, Go file
// nodes return default (no resolution), so freshness falls back to
// time-decay. Edges without SHA stamps are treated as fresh (1.0).
func TestAssess_GoFileNodeWithoutGoMod(t *testing.T) {
	root := t.TempDir()
	edge := &graph.Edge{
		To: "go:github.com/acme/widgets/widget/core/thing.go",
		Properties: map[string]any{
			"source_sha": "abc",
		},
	}
	c := New(root, testPolicy())
	// No go.mod means we can't resolve the Go file node to a path, so
	// hashForNode returns "" → treated as "file missing/unreadable" →
	// stale multiplier.
	if got := c.Assess(edge); got != 0.3 {
		t.Errorf("unresolvable Go file with stamped SHA: Assess = %f, want stale 0.3", got)
	}
}

// TestHashFile_MatchesGit — our blob SHA matches git's algorithm.
func TestHashFile_MatchesGit(t *testing.T) {
	// Known git blob SHA for content "hello\n" is ce013625030ba8dba906f756967f9e9ca394464a
	tmp := filepath.Join(t.TempDir(), "hello.txt")
	if err := os.WriteFile(tmp, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	sha, err := HashFile(tmp)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	want := "ce013625030ba8dba906f756967f9e9ca394464a"
	if sha != want {
		t.Errorf("HashFile = %q, want %q (git blob SHA of 'hello\\n')", sha, want)
	}
}

func setupContractsFile(t *testing.T, root, relPath string, content []byte) {
	t.Helper()
	full := filepath.Join(root, "packages/contracts-bedrock", relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, content, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
