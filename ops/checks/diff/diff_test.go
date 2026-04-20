package diff

import (
	"testing"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

const sampleUnifiedDiff = `diff --git a/op-node/rollup/derive/batch_queue.go b/op-node/rollup/derive/batch_queue.go
index abc1234..def5678 100644
--- a/op-node/rollup/derive/batch_queue.go
+++ b/op-node/rollup/derive/batch_queue.go
@@ -10,7 +10,8 @@ func (bq *BatchQueue) AddBatch(batch *Batch) error {
 	if batch == nil {
 		return errors.New("nil batch")
 	}
-	bq.batches = append(bq.batches, batch)
+	// Validate before appending
+	if err := batch.Validate(); err != nil {
+		return fmt.Errorf("invalid batch: %w", err)
+	}
 	return nil
 }
diff --git a/op-node/rollup/derive/batch_queue_test.go b/op-node/rollup/derive/batch_queue_test.go
index 1111111..2222222 100644
--- a/op-node/rollup/derive/batch_queue_test.go
+++ b/op-node/rollup/derive/batch_queue_test.go
@@ -5,6 +5,12 @@ func TestAddBatch(t *testing.T) {
 	bq := NewBatchQueue()
 	err := bq.AddBatch(nil)
 	require.Error(t, err)
+}
+
+func TestAddBatch_Invalid(t *testing.T) {
+	bq := NewBatchQueue()
+	err := bq.AddBatch(&Batch{Invalid: true})
+	require.Error(t, err)
 }
`

func TestParseUnifiedDiff_TwoFiles(t *testing.T) {
	diffs := ParseUnifiedDiff(sampleUnifiedDiff)
	if len(diffs) != 2 {
		t.Fatalf("expected 2 file diffs, got %d", len(diffs))
	}

	if diffs[0].Path != "op-node/rollup/derive/batch_queue.go" {
		t.Errorf("expected batch_queue.go, got %q", diffs[0].Path)
	}
	if diffs[1].Path != "op-node/rollup/derive/batch_queue_test.go" {
		t.Errorf("expected batch_queue_test.go, got %q", diffs[1].Path)
	}
}

func TestParseUnifiedDiff_HunkContent(t *testing.T) {
	diffs := ParseUnifiedDiff(sampleUnifiedDiff)

	// First file should have 1 hunk with added and removed lines
	if len(diffs[0].Hunks) != 1 {
		t.Fatalf("expected 1 hunk in first file, got %d", len(diffs[0].Hunks))
	}

	hunk := diffs[0].Hunks[0]
	if hunk.OldStart != 10 {
		t.Errorf("expected OldStart=10, got %d", hunk.OldStart)
	}
	if len(hunk.Removed) != 1 {
		t.Errorf("expected 1 removed line, got %d", len(hunk.Removed))
	}
	if len(hunk.Added) != 4 {
		t.Errorf("expected 4 added lines, got %d", len(hunk.Added))
	}
}

func TestParseUnifiedDiff_Summary(t *testing.T) {
	diffs := ParseUnifiedDiff(sampleUnifiedDiff)
	summary := Summarize(diffs)

	if summary.TotalAdded != 10 {
		t.Errorf("expected 10 added lines, got %d", summary.TotalAdded)
	}
	if summary.TotalRemoved != 1 {
		t.Errorf("expected 1 removed line, got %d", summary.TotalRemoved)
	}
	if summary.HasNewFiles {
		t.Error("no new files expected")
	}
}

const newFileDiff = `diff --git a/op-node/rollup/new_feature.go b/op-node/rollup/new_feature.go
new file mode 100644
index 0000000..abcdef1
--- /dev/null
+++ b/op-node/rollup/new_feature.go
@@ -0,0 +1,5 @@
+package rollup
+
+func NewFeature() string {
+	return "hello"
+}
`

func TestParseUnifiedDiff_NewFile(t *testing.T) {
	diffs := ParseUnifiedDiff(newFileDiff)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if !diffs[0].IsNew {
		t.Error("expected IsNew=true")
	}
	if diffs[0].Path != "op-node/rollup/new_feature.go" {
		t.Errorf("expected new_feature.go, got %q", diffs[0].Path)
	}
	summary := Summarize(diffs)
	if !summary.HasNewFiles {
		t.Error("expected HasNewFiles=true")
	}
}

const deletedFileDiff = `diff --git a/op-node/rollup/old.go b/op-node/rollup/old.go
deleted file mode 100644
index abcdef1..0000000
--- a/op-node/rollup/old.go
+++ /dev/null
@@ -1,3 +0,0 @@
-package rollup
-
-func Old() {}
`

func TestParseUnifiedDiff_DeletedFile(t *testing.T) {
	diffs := ParseUnifiedDiff(deletedFileDiff)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if !diffs[0].IsDelete {
		t.Error("expected IsDelete=true")
	}
	if diffs[0].Path != "op-node/rollup/old.go" {
		t.Errorf("expected old.go, got %q", diffs[0].Path)
	}
}

// TestParseUnifiedDiff_GitShowPreamble — git show and git log -p
// prefix the diff with a commit header. The parser should skip past
// that and start at the first `diff --git` block, not fall back to
// name-only mode.
func TestParseUnifiedDiff_GitShowPreamble(t *testing.T) {
	input := `commit abc123
Author: Someone <someone@example.com>
Date:   Wed Apr 8 09:42:54 2026 -0400

    fix: some change

    This is a multi-line commit message describing what changed
    and why it matters to the project.

diff --git a/pkg/foo.go b/pkg/foo.go
--- a/pkg/foo.go
+++ b/pkg/foo.go
@@ -1,3 +1,3 @@
-old line
+new line
 context
`
	diffs := ParseUnifiedDiff(input)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 file diff past the commit header, got %d: %+v", len(diffs), diffs)
	}
	if diffs[0].Path != "pkg/foo.go" {
		t.Errorf("Path = %q, want pkg/foo.go", diffs[0].Path)
	}
	if len(diffs[0].Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(diffs[0].Hunks))
	}
	h := diffs[0].Hunks[0]
	if len(h.Added) != 1 || h.Added[0] != "new line" {
		t.Errorf("Added = %v, want [new line]", h.Added)
	}
	if len(h.Removed) != 1 || h.Removed[0] != "old line" {
		t.Errorf("Removed = %v, want [old line]", h.Removed)
	}
}

func TestChangedFiles_UnifiedDiff(t *testing.T) {
	files := ChangedFiles(sampleUnifiedDiff)
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d: %v", len(files), files)
	}
}

func TestChangedFiles_NameOnly(t *testing.T) {
	output := `op-node/rollup/derive/batch_queue.go
op-node/rollup/derive/batch_queue_test.go
packages/contracts-bedrock/src/L1/OptimismPortal2.sol
`
	files := ChangedFiles(output)
	if len(files) != 3 {
		t.Errorf("expected 3 files, got %d", len(files))
	}
}

func TestChangedFiles_EmptyInput(t *testing.T) {
	files := ChangedFiles("")
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestFilesToNodeIDs_GoFile(t *testing.T) {
	g := graph.NewGraph()
	_ = g.AddNode(&graph.Node{
		ID: "go:github.com/org/repo/op-node/rollup/derive", Kind: graph.KindSource,
		Name: "derive", Properties: map[string]any{"dir": "/repo/op-node/rollup/derive"},
	})

	ids, unknown := FilesToNodeIDs(g, []string{"op-node/rollup/derive/batch_queue.go"})
	if len(ids) != 1 {
		t.Errorf("expected 1 node ID, got %d (unknown: %v)", len(ids), unknown)
	}
}

func TestFilesToNodeIDs_SolidityFile(t *testing.T) {
	g := graph.NewGraph()
	_ = g.AddNode(&graph.Node{
		ID: "sol:src/L1/OptimismPortal2.sol", Kind: graph.KindSource,
		Name: "OptimismPortal2.sol",
	})

	ids, _ := FilesToNodeIDs(g, []string{"packages/contracts-bedrock/src/L1/OptimismPortal2.sol"})
	if len(ids) != 1 {
		t.Errorf("expected 1 node ID, got %d", len(ids))
	}
	if len(ids) > 0 && ids[0] != "sol:src/L1/OptimismPortal2.sol" {
		t.Errorf("expected sol:src/L1/OptimismPortal2.sol, got %s", ids[0])
	}
}

func TestFilesToNodeIDs_UnknownFile(t *testing.T) {
	g := graph.NewGraph()
	_, unknown := FilesToNodeIDs(g, []string{"README.md"})
	if len(unknown) != 1 {
		t.Errorf("expected 1 unknown file, got %d", len(unknown))
	}
}

