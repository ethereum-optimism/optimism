package coverage

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// RustCollector collects coverage from Rust tests using cargo-llvm-cov.
//
// LCOV output from cargo-llvm-cov keys each source file by absolute
// path. To fit the graph's rs:<crate>/<rel>.rs node schema, the
// collector uses `cargo metadata` to get workspace crate boundaries
// and rewrites each absolute path to <crate>/<rel-to-manifest>. Keys
// outside any workspace crate (stdlib, external deps) are dropped.
// Source SHAs are stamped using the same path mapping.
type RustCollector struct {
	// WorkspaceDir is the path to the Rust workspace relative to rootDir.
	// Default: "rust".
	WorkspaceDir string

	// CratesFor returns workspace members given an absolute workspace
	// directory. Default shells to `cargo metadata --no-deps`; tests
	// can substitute.
	CratesFor func(workDir string) ([]rustCrate, error)
}

// rustCrate mirrors adapter/rust.Crate locally to avoid a cross-
// package dependency. Name + absolute manifest dir.
type rustCrate struct {
	Name        string
	ManifestDir string
}

func NewRustCollector() *RustCollector {
	return &RustCollector{WorkspaceDir: "rust"}
}

func (c *RustCollector) Language() string { return "rust" }

// Collect runs cargo llvm-cov for a specific crate/test and parses LCOV output.
// testPath is a crate name (e.g. "kona-derive") or test binary name.
func (c *RustCollector) Collect(rootDir string, testPath string, profile Profile) (*Report, error) {
	workDir := filepath.Join(rootDir, c.WorkspaceDir)

	if _, err := exec.LookPath("cargo-llvm-cov"); err != nil {
		return nil, fmt.Errorf("cargo-llvm-cov not found: install with 'cargo install cargo-llvm-cov'")
	}

	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("checks-rust-coverage-%d.lcov", os.Getpid()))
	defer os.Remove(tmpFile)

	cmd := exec.Command("cargo", "llvm-cov",
		"--lcov",
		"--output-path", tmpFile,
		"--no-cfg-coverage",
		"--all-features",
		"--ignore-run-fail",
		"--package", testPath,
	)
	cmd.Dir = workDir
	cmd.Env = os.Environ()
	for k, v := range profile.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	cmd.Stderr = os.Stderr
	_ = cmd.Run()

	rawCovers, err := parseLCOVFile(tmpFile)
	if err != nil {
		return nil, fmt.Errorf("parsing LCOV output (cargo may have failed): %w", err)
	}

	crateLookup, err := c.loadCrates(workDir)
	if err != nil {
		// Degrade: without crate boundaries we can't rewrite keys, so
		// emit the report with LCOV absolute paths and no SHA stamps.
		// Callers get time-based staleness only for these edges.
		report := &Report{Test: testPath, Language: "rust", Profile: profile.Name, Covers: rawCovers}
		stampReport(report, "", nil)
		return report, nil
	}

	covers, sourceAbsFor := rewriteCoversToCrateRelative(rawCovers, crateLookup)

	report := &Report{
		Test:     testPath,
		Language: "rust",
		Profile:  profile.Name,
		Covers:   covers,
	}
	stampReport(report, "", sourceAbsFor)
	return report, nil
}

// rewriteCoversToCrateRelative maps absolute-path-keyed coverage data
// to <crate>/<rel> keys using crate boundaries. Returns:
//   - the rewritten Covers map (keys now match rs:<crate>/<rel> node IDs)
//   - a sourceAbsFor callback that maps each new key back to the
//     absolute path so stampReport can hash it
//
// Paths outside every crate's manifest dir are silently dropped.
func rewriteCoversToCrateRelative(
	raw map[string][][2]int,
	crates []rustCrate,
) (map[string][][2]int, func(string) string) {
	// Sort longest-manifest-dir-first so nested crates win over their
	// workspace root when prefix-matching.
	sort.Slice(crates, func(i, j int) bool {
		return len(crates[i].ManifestDir) > len(crates[j].ManifestDir)
	})

	out := make(map[string][][2]int, len(raw))
	absByKey := make(map[string]string, len(raw))
	for abs, ranges := range raw {
		for _, crate := range crates {
			if abs == crate.ManifestDir || strings.HasPrefix(abs, crate.ManifestDir+"/") {
				rel, err := filepath.Rel(crate.ManifestDir, abs)
				if err != nil {
					break
				}
				key := crate.Name + "/" + rel
				out[key] = ranges
				absByKey[key] = abs
				break
			}
		}
	}
	return out, func(key string) string {
		return absByKey[key]
	}
}

// loadCrates resolves workspace members via the configured CratesFor
// callback (default: cargo metadata).
func (c *RustCollector) loadCrates(workDir string) ([]rustCrate, error) {
	if c.CratesFor != nil {
		return c.CratesFor(workDir)
	}
	return cargoMetadataCrates(workDir)
}

func cargoMetadataCrates(workDir string) ([]rustCrate, error) {
	cmd := exec.Command("cargo", "metadata", "--no-deps", "--format-version", "1")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cargo metadata: %w", err)
	}
	var meta struct {
		Packages []struct {
			Name         string `json:"name"`
			ManifestPath string `json:"manifest_path"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(out, &meta); err != nil {
		return nil, fmt.Errorf("parsing cargo metadata: %w", err)
	}
	crates := make([]rustCrate, 0, len(meta.Packages))
	for _, p := range meta.Packages {
		crates = append(crates, rustCrate{
			Name:        p.Name,
			ManifestDir: filepath.Dir(p.ManifestPath),
		})
	}
	return crates, nil
}
