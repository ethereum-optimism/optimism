package coverage

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/rustmeta"
)

// RustCollector collects coverage from Rust tests using cargo-llvm-cov.
//
// LCOV output from cargo-llvm-cov keys each source file by absolute
// path. To fit the graph's rs:<crate>/<rel>.rs node schema, the
// collector uses cargo metadata to get workspace crate boundaries and
// rewrites each absolute path to <crate>/<rel-to-manifest>. Keys
// outside any workspace crate (stdlib, external deps) are dropped.
// Source SHAs are stamped using the same path mapping.
//
// The metadata Loader is held on the collector, not fetched per call,
// so batch coverage runs invoke cargo metadata once for the workspace
// instead of once per test.
type RustCollector struct {
	// WorkspaceDir is the path to the Rust workspace relative to rootDir.
	// Default: "rust".
	WorkspaceDir string

	// Loader is the crate metadata resolver. Defaults to the global
	// cached loader backed by cargo metadata; tests can swap it for a
	// deterministic fetcher via rustmeta.NewLoaderWith.
	Loader *rustmeta.Loader
}

// NewRustCollector returns a collector with a fresh cached Loader.
// Callers that want metadata reuse across multiple collectors (e.g.
// in a batch run) should share a single RustCollector instance
// rather than constructing one per Collect call.
func NewRustCollector() *RustCollector {
	return &RustCollector{
		WorkspaceDir: "rust",
		Loader:       rustmeta.NewLoader(),
	}
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

	crateLookup, err := c.Loader.Load(workDir)
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
	crates []rustmeta.Crate,
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

