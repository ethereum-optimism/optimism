package freshness

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	c := New(root, testPolicy(), nil)
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
	c := New(root, testPolicy(), nil)
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
	c := New(root, testPolicy(), nil)
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
	c := New(root, testPolicy(), nil)
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
	c := New(root, testPolicy(), nil)
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

	c := New(root, testPolicy(), nil)
	if got := c.Assess(edge); got != 1.0 {
		t.Errorf("matching Go file SHA: Assess = %f, want 1.0", got)
	}

	// Mismatch: rewrite the file's content, SHA no longer matches stamp.
	if err := os.WriteFile(filePath, []byte("package core\n\nvar X = 2\n"), 0o644); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	c2 := New(root, testPolicy(), nil) // fresh checker — avoid cache hit from prior hash
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
	c := New(root, testPolicy(), nil)
	// No go.mod means we can't resolve the Go file node to a path, so
	// hashForNode returns "" → treated as "file missing/unreadable" →
	// stale multiplier.
	if got := c.Assess(edge); got != 0.3 {
		t.Errorf("unresolvable Go file with stamped SHA: Assess = %f, want stale 0.3", got)
	}
}

// TestAssess_RustFileNodeResolvesViaGraph — a rs:<crate>/<rel>.rs
// node resolves to a filesystem path by looking up the crate's
// manifest dir in the graph. SHA match/mismatch behaves identically
// to Solidity and Go.
func TestAssess_RustFileNodeResolvesViaGraph(t *testing.T) {
	root := t.TempDir()

	// Lay out a single-crate workspace: <root>/rust/crates/kona-derive/src/lib.rs
	crateDir := filepath.Join(root, "rust", "crates", "kona-derive")
	if err := os.MkdirAll(filepath.Join(crateDir, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	filePath := filepath.Join(crateDir, "src", "lib.rs")
	if err := os.WriteFile(filePath, []byte("pub fn hello() {}\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Build a graph with crate node carrying manifest dir, file node
	// for lib.rs.
	g := graph.NewGraph()
	_ = g.AddNode(&graph.Node{
		ID: "rs:kona-derive", Kind: graph.KindSource, Granularity: "crate",
		Properties: map[string]any{"dir": crateDir},
	})
	_ = g.AddNode(&graph.Node{
		ID: "rs:kona-derive/src/lib.rs", Kind: graph.KindSource, Granularity: "file",
	})

	// Stamp a matching SHA on an edge and verify.
	sha, err := HashFile(filePath)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	edge := &graph.Edge{
		From: "rs:kona-derive", To: "rs:kona-derive/src/lib.rs",
		Properties: map[string]any{"source_sha": sha},
	}
	c := New(root, testPolicy(), g)
	if got := c.Assess(edge); got != 1.0 {
		t.Errorf("matching Rust file SHA: Assess = %f, want 1.0", got)
	}

	// Rewrite the file → SHA mismatches → stale multiplier.
	if err := os.WriteFile(filePath, []byte("pub fn hello() { println!(\"hi\"); }\n"), 0o644); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	c2 := New(root, testPolicy(), g)
	if got := c2.Assess(edge); got != 0.3 {
		t.Errorf("mismatched Rust file SHA: Assess = %f, want 0.3", got)
	}
}

// TestAssess_RustFileNodeNoGraph — without a graph, rs: file nodes
// can't resolve, so stamped edges fall through to the stale path.
func TestAssess_RustFileNodeNoGraph(t *testing.T) {
	root := t.TempDir()
	edge := &graph.Edge{
		To: "rs:kona-derive/src/lib.rs",
		Properties: map[string]any{"source_sha": "abc"},
	}
	c := New(root, testPolicy(), nil)
	if got := c.Assess(edge); got != 0.3 {
		t.Errorf("rs: with nil graph and stamped SHA: Assess = %f, want stale 0.3", got)
	}
}

// TestAssess_PerHunkGradient_MostLinesStillPresent — SHA mismatch
// but most covered lines are unchanged → gradient exceeds the stale
// floor and is returned.
func TestAssess_PerHunkGradient_MostLinesStillPresent(t *testing.T) {
	root := initGitRepo(t)

	// Commit original content with 6 lines, record its blob SHA.
	original := "line1\nline2\nline3\nline4\nline5\nline6\n"
	filePath := filepath.Join(root, "file.sol")
	writeAndCommit(t, root, "file.sol", original, "initial")
	oldSha := blobSha(t, root, "file.sol")

	// Now rewrite only line6 on disk (uncommitted). Lines 1-5
	// unchanged; line 6 replaced. Coverage stamped lines 1-4.
	newContent := "line1\nline2\nline3\nline4\nline5\nline6_REPLACED\n"
	if err := os.WriteFile(filePath, []byte(newContent), 0o644); err != nil {
		t.Fatalf("rewrite: %v", err)
	}

	edge := &graph.Edge{
		To: "sol:file.sol",
		Properties: map[string]any{
			"source_sha":  oldSha,
			"line_ranges": [][2]int{{1, 4}},
		},
	}

	// The node resolver uses packages/contracts-bedrock/ prefix for
	// sol: IDs. Our test file sits at root/file.sol; put it under
	// that prefix for the resolver to find.
	must(t, os.MkdirAll(filepath.Join(root, "packages/contracts-bedrock"), 0o755))
	must(t, os.Rename(filePath, filepath.Join(root, "packages/contracts-bedrock/file.sol")))
	// Re-commit at the new path so git cat-file still works on the
	// stored sha (blobs are path-agnostic — same content, same sha).
	// Nothing to do here — the blob is already in the object db.

	pol := &policy.Policy{Freshness: policy.FreshnessConfig{StaleMultiplier: 0.3, MaxAgeDays: 30}}
	c := New(root, pol, nil)
	got := c.Assess(edge)
	// 4 of 4 covered lines still present in new content → fraction 1.0.
	// Actually exactly 4/4 since lines 1-4 all survived. Floor is
	// 0.3; gradient lifts above it.
	if got < 0.9 {
		t.Errorf("per-hunk gradient: Assess = %f, want ≥ 0.9 (all covered lines still present)", got)
	}
}

// TestAssess_PerHunkGradient_AllCoveredLinesGone — SHA mismatch and
// none of the covered lines survive → gradient is 0, floor
// (staleMultiplier) applies.
func TestAssess_PerHunkGradient_AllCoveredLinesGone(t *testing.T) {
	root := initGitRepo(t)
	must(t, os.MkdirAll(filepath.Join(root, "packages/contracts-bedrock"), 0o755))

	// Commit "A\nB\nC\n".
	writeAndCommit(t, root, "packages/contracts-bedrock/file.sol", "A\nB\nC\n", "initial")
	oldSha := blobSha(t, root, "packages/contracts-bedrock/file.sol")

	// Rewrite to completely different lines.
	must(t, os.WriteFile(filepath.Join(root, "packages/contracts-bedrock/file.sol"),
		[]byte("X\nY\nZ\n"), 0o644))

	edge := &graph.Edge{
		To: "sol:file.sol",
		Properties: map[string]any{
			"source_sha":  oldSha,
			"line_ranges": [][2]int{{1, 3}},
		},
	}
	pol := &policy.Policy{Freshness: policy.FreshnessConfig{StaleMultiplier: 0.3, MaxAgeDays: 30}}
	c := New(root, pol, nil)
	got := c.Assess(edge)
	if got != 0.3 {
		t.Errorf("all covered lines gone: Assess = %f, want 0.3 (floor)", got)
	}
}

// TestAssess_PerHunkGradient_UncommittedBlob — source_sha was never
// committed (git cat-file fails) → fall back to binary stale.
func TestAssess_PerHunkGradient_UncommittedBlob(t *testing.T) {
	root := initGitRepo(t)
	must(t, os.MkdirAll(filepath.Join(root, "packages/contracts-bedrock"), 0o755))
	must(t, os.WriteFile(filepath.Join(root, "packages/contracts-bedrock/file.sol"),
		[]byte("current content\n"), 0o644))

	edge := &graph.Edge{
		To: "sol:file.sol",
		Properties: map[string]any{
			"source_sha":  "deadbeef00000000000000000000000000000000",
			"line_ranges": [][2]int{{1, 1}},
		},
	}
	pol := &policy.Policy{Freshness: policy.FreshnessConfig{StaleMultiplier: 0.3, MaxAgeDays: 30}}
	c := New(root, pol, nil)
	got := c.Assess(edge)
	if got != 0.3 {
		t.Errorf("uncommitted old blob: Assess = %f, want 0.3 (binary stale)", got)
	}
}

// --- helpers ---

func initGitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
		{"config", "commit.gpgsign", "false"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return root
}

func writeAndCommit(t *testing.T, root, rel, content, msg string) {
	t.Helper()
	full := filepath.Join(root, rel)
	must(t, os.MkdirAll(filepath.Dir(full), 0o755))
	must(t, os.WriteFile(full, []byte(content), 0o644))
	for _, args := range [][]string{
		{"add", rel},
		{"commit", "-m", msg, "--no-gpg-sign"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

func blobSha(t *testing.T, root, rel string) string {
	t.Helper()
	cmd := exec.Command("git", "hash-object", rel)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git hash-object: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
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
