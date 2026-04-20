package diff

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// Hunk represents a changed region within a file.
type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Added    []string // lines added (without leading +)
	Removed  []string // lines removed (without leading -)
	Context  []string // unchanged context lines (without leading space)
}

// FileDiff represents the parsed diff for a single file.
type FileDiff struct {
	Path    string // file path (from b/ side, or a/ side for deletes)
	OldPath string // original path (for renames)
	IsNew   bool
	IsDelete bool
	IsRename bool
	Hunks   []Hunk
}

// ParseUnifiedDiff parses standard unified diff output (from `git diff`,
// `git show`, `git log -p`, etc.) into structured FileDiff entries. Also
// works with `git diff --name-only` (falls back to bare file-list
// parsing when no `diff --git` line is found).
//
// Commit-message preamble (from `git show` / `git log -p`) is tolerated:
// the parser scans for the first `diff --git` block and starts there.
func ParseUnifiedDiff(input string) []FileDiff {
	lines := strings.Split(input, "\n")

	// Scan for the first unified-diff block. `git show` and `git log -p`
	// both prefix the diff with a commit header + message, which would
	// otherwise trigger the name-only fallback.
	start := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "diff --git ") {
			start = i
			break
		}
	}
	if start < 0 {
		// No unified-diff block anywhere — treat as a bare file list.
		return nameOnlyToDiffs(lines)
	}
	lines = lines[start:]

	var diffs []FileDiff
	var current *FileDiff
	var currentHunk *Hunk

	for _, line := range lines {
		// New file diff header
		if strings.HasPrefix(line, "diff --git ") {
			if current != nil {
				if currentHunk != nil {
					current.Hunks = append(current.Hunks, *currentHunk)
					currentHunk = nil
				}
				diffs = append(diffs, *current)
			}
			current = &FileDiff{}
			currentHunk = nil

			// Parse "diff --git a/path b/path"
			parts := strings.SplitN(line, " b/", 2)
			if len(parts) == 2 {
				current.Path = parts[1]
			}
			// Also extract a/ path for renames/deletes
			aParts := strings.SplitN(line, " a/", 2)
			if len(aParts) == 2 {
				aPath := strings.SplitN(aParts[1], " ", 2)
				current.OldPath = aPath[0]
			}
			continue
		}

		if current == nil {
			continue
		}

		// File mode indicators
		if strings.HasPrefix(line, "new file") {
			current.IsNew = true
			continue
		}
		if strings.HasPrefix(line, "deleted file") {
			current.IsDelete = true
			continue
		}
		if strings.HasPrefix(line, "rename from ") {
			current.IsRename = true
			current.OldPath = strings.TrimPrefix(line, "rename from ")
			continue
		}
		if strings.HasPrefix(line, "rename to ") {
			current.Path = strings.TrimPrefix(line, "rename to ")
			continue
		}

		// --- and +++ lines refine the path
		if strings.HasPrefix(line, "--- a/") {
			current.OldPath = strings.TrimPrefix(line, "--- a/")
			continue
		}
		if strings.HasPrefix(line, "+++ b/") {
			current.Path = strings.TrimPrefix(line, "+++ b/")
			continue
		}
		if line == "--- /dev/null" {
			current.IsNew = true
			continue
		}
		if line == "+++ /dev/null" {
			current.IsDelete = true
			// For deletes, use the a/ path
			if current.OldPath != "" {
				current.Path = current.OldPath
			}
			continue
		}

		// Hunk header: @@ -oldStart,oldCount +newStart,newCount @@
		if strings.HasPrefix(line, "@@") {
			if currentHunk != nil {
				current.Hunks = append(current.Hunks, *currentHunk)
			}
			currentHunk = parseHunkHeader(line)
			continue
		}

		// Hunk content
		if currentHunk != nil {
			if strings.HasPrefix(line, "+") {
				currentHunk.Added = append(currentHunk.Added, strings.TrimPrefix(line, "+"))
			} else if strings.HasPrefix(line, "-") {
				currentHunk.Removed = append(currentHunk.Removed, strings.TrimPrefix(line, "-"))
			} else if strings.HasPrefix(line, " ") {
				currentHunk.Context = append(currentHunk.Context, strings.TrimPrefix(line, " "))
			}
		}
	}

	// Flush last file
	if current != nil {
		if currentHunk != nil {
			current.Hunks = append(current.Hunks, *currentHunk)
		}
		diffs = append(diffs, *current)
	}

	return diffs
}

// ChangedFiles extracts file paths from either unified diff or --name-only output.
func ChangedFiles(input string) []string {
	diffs := ParseUnifiedDiff(input)
	var files []string
	seen := make(map[string]bool)
	for _, d := range diffs {
		if d.Path != "" && !seen[d.Path] {
			files = append(files, d.Path)
			seen[d.Path] = true
		}
	}
	return files
}

// DiffSummary returns a high-level summary of changes for scoring purposes.
type DiffSummary struct {
	Files        []FileDiff
	TotalAdded   int
	TotalRemoved int
	HasNewFiles  bool
	HasDeletes   bool
	HasRenames   bool
}

// Summarize computes aggregate statistics from parsed diffs.
func Summarize(diffs []FileDiff) DiffSummary {
	s := DiffSummary{Files: diffs}
	for _, d := range diffs {
		if d.IsNew {
			s.HasNewFiles = true
		}
		if d.IsDelete {
			s.HasDeletes = true
		}
		if d.IsRename {
			s.HasRenames = true
		}
		for _, h := range d.Hunks {
			s.TotalAdded += len(h.Added)
			s.TotalRemoved += len(h.Removed)
		}
	}
	return s
}

func nameOnlyToDiffs(lines []string) []FileDiff {
	var diffs []FileDiff
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			diffs = append(diffs, FileDiff{Path: line})
		}
	}
	return diffs
}

func parseHunkHeader(line string) *Hunk {
	// @@ -oldStart,oldCount +newStart,newCount @@ optional context
	h := &Hunk{}
	line = strings.TrimPrefix(line, "@@")
	idx := strings.Index(line[1:], "@@")
	if idx < 0 {
		return h
	}
	line = strings.TrimSpace(line[:idx+1])

	parts := strings.Fields(line)
	for _, p := range parts {
		if strings.HasPrefix(p, "-") {
			nums := strings.SplitN(strings.TrimPrefix(p, "-"), ",", 2)
			h.OldStart, _ = strconv.Atoi(nums[0])
			if len(nums) > 1 {
				h.OldCount, _ = strconv.Atoi(nums[1])
			}
		} else if strings.HasPrefix(p, "+") {
			nums := strings.SplitN(strings.TrimPrefix(p, "+"), ",", 2)
			h.NewStart, _ = strconv.Atoi(nums[0])
			if len(nums) > 1 {
				h.NewCount, _ = strconv.Atoi(nums[1])
			}
		}
	}
	return h
}

// FilesToNodeIDs maps changed file paths to graph node IDs.
func FilesToNodeIDs(g *graph.Graph, files []string) (nodeIDs []string, unknown []string) {
	seen := make(map[string]bool)

	for _, f := range files {
		matched := false

		// Try Go: map file to its file node AND package node. Same
		// reasoning as Rust: file nodes are coverage leaves with no
		// outgoing import edges, so the import-walk must start from
		// the package. Emitting both keeps coverage-edge matching
		// (file granularity) working while also feeding transitive
		// consumers (package granularity) into the scoping path.
		if strings.HasSuffix(f, ".go") {
			dir := filepath.Dir(f)
			// Collect every matching node, then emit package(s) first.
			var matches []*graph.Node
			for _, node := range g.NodesOfKind(graph.KindSource) {
				if !strings.HasPrefix(node.ID, "go:") {
					continue
				}
				nodeDir, _ := node.Properties["dir"].(string)
				if nodeDir != "" && strings.HasSuffix(nodeDir, "/"+dir) {
					matches = append(matches, node)
					continue
				}
				// Fallback: match by package name in path when dir prop is missing.
				goPath := strings.TrimPrefix(node.ID, "go:")
				if matchesPath(goPath, dir) {
					matches = append(matches, node)
				}
			}
			// Package-or-unspecified granularity first so import-walk
			// starts from the package node; file nodes come after and
			// still anchor coverage-edge lookups.
			for _, n := range matches {
				if n.Granularity != "file" && !seen[n.ID] {
					nodeIDs = append(nodeIDs, n.ID)
					seen[n.ID] = true
					matched = true
				}
			}
			for _, n := range matches {
				if n.Granularity == "file" && !seen[n.ID] {
					nodeIDs = append(nodeIDs, n.ID)
					seen[n.ID] = true
					matched = true
				}
			}
		}

		// Try Solidity: map file to sol: node
		if strings.HasSuffix(f, ".sol") {
			// Try with and without "packages/contracts-bedrock/" prefix
			candidates := []string{f}
			if strings.HasPrefix(f, "packages/contracts-bedrock/") {
				candidates = append(candidates, strings.TrimPrefix(f, "packages/contracts-bedrock/"))
			}
			for _, c := range candidates {
				nodeID := "sol:" + c
				if g.GetNode(nodeID) != nil && !seen[nodeID] {
					nodeIDs = append(nodeIDs, nodeID)
					seen[nodeID] = true
					matched = true
					break
				}
			}
		}

		// Try Rust: map file to rs: file node AND its parent crate
		// node. File nodes are coverage-target leaves with no outgoing
		// import edges, so the import-walk can't start from them.
		// Emitting the crate node makes the walk pick up every crate
		// that transitively depends on this one, which drives
		// per-crate rust-test scoping.
		if strings.HasSuffix(f, ".rs") && !matched {
			dir := filepath.Dir(f)
			for _, node := range g.NodesOfKind(graph.KindSource) {
				if !strings.HasPrefix(node.ID, "rs:") {
					continue
				}
				nodeDir, _ := node.Properties["dir"].(string)
				if nodeDir == "" {
					continue
				}
				if !strings.HasSuffix(nodeDir, "/"+dir) {
					continue
				}
				if !seen[node.ID] {
					nodeIDs = append(nodeIDs, node.ID)
					seen[node.ID] = true
				}
				// Also emit the owning crate node. Package prop on file
				// nodes isn't carried today, so derive from ID prefix
				// ("rs:<crate>/..." → "rs:<crate>").
				if node.Granularity == "file" {
					if idx := strings.Index(node.ID[3:], "/"); idx > 0 {
						crateID := "rs:" + node.ID[3:3+idx]
						if g.GetNode(crateID) != nil && !seen[crateID] {
							nodeIDs = append(nodeIDs, crateID)
							seen[crateID] = true
						}
					}
				}
				matched = true
				break
			}
		}

		if !matched {
			unknown = append(unknown, f)
		}
	}

	return
}

func matchesPath(goImportPath, relDir string) bool {
	// Check if the Go import path ends with the relative directory path
	return strings.HasSuffix(goImportPath, "/"+relDir) || strings.HasSuffix(goImportPath, relDir)
}
