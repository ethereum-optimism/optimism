// Package freshness assesses whether an evidence edge's content still
// reflects the current repo. Evidence edges (coverage, CI-history, AI-
// annotated) are generated at a point in time from specific file
// contents; when those files change underneath, the edge becomes less
// reliable.
//
// Freshness is represented as a multiplier in [0, 1] applied to an
// edge's signal during Phase 1 aggregation:
//
//	effective_signal = raw_signal * freshness.Assess(edge)
//
// An edge whose stamped content SHAs still match current file content
// returns 1.0. A mismatched SHA drops to policy.Freshness.StaleMultiplier
// (default ~0.3). An edge older than MaxAgeDays without a matching SHA
// is treated as stale. Edges with no stamps are assumed fresh (1.0), so
// legacy graphs work unchanged until they're regenerated.
package freshness

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
	"github.com/ethereum-optimism/optimism/ops/checks/policy"
)

// Checker assesses how fresh an evidence edge is relative to the
// current repo state.
type Checker interface {
	// Assess returns a multiplier in [0, 1]. 1.0 = as-generated;
	// 0 = maximally stale / cannot be verified.
	Assess(edge *graph.Edge) float64
}

// Nop returns a Checker that always reports fresh. Use in tests or
// when file-system access is unavailable.
func Nop() Checker { return nopChecker{} }

type nopChecker struct{}

func (nopChecker) Assess(*graph.Edge) float64 { return 1.0 }

// New returns a Checker that reads files under rootDir to compare
// current content SHAs against stamps stored on edges. Settings come
// from policy.Freshness.
//
// If go.mod is present at rootDir, the checker reads its module
// directive so it can resolve go:<module>/<rel>.go node IDs back to
// filesystem paths. The optional graph enables rs:<crate>/<rel>.rs
// resolution by looking up each Rust file node's containing crate
// (which carries a manifest `dir` property from the Rust adapter).
// Pass nil for the graph when Rust coverage isn't in scope (tests,
// non-Rust repos).
func New(rootDir string, p *policy.Policy, g *graph.Graph) Checker {
	return &repoChecker{
		rootDir:         rootDir,
		graph:           g,
		goModulePath:    readGoModulePath(rootDir),
		staleMultiplier: p.Freshness.StaleMultiplier,
		maxAge:          time.Duration(p.Freshness.MaxAgeDays) * 24 * time.Hour,
		cache:           make(map[string]string),
	}
}

type repoChecker struct {
	rootDir         string
	graph           *graph.Graph // optional; used for rs: resolution
	goModulePath    string       // module path from go.mod; "" if unavailable
	staleMultiplier float64
	maxAge          time.Duration

	mu    sync.Mutex
	cache map[string]string // abs path → git blob SHA (or "" for not-readable)
}

func (c *repoChecker) Assess(edge *graph.Edge) float64 {
	if edge == nil || edge.Properties == nil {
		return 1.0
	}
	props := edge.Properties

	// Prefer SHA-based assessment when stamps are present.
	testSha := asString(props["test_sha"])
	sourceSha := asString(props["source_sha"])
	if testSha != "" || sourceSha != "" {
		if testSha != "" {
			cur := c.hashForNode(edge.From)
			if cur == "" {
				// File missing or unreadable: treat as stale.
				return c.staleMultiplier
			}
			if cur != testSha {
				return c.staleMultiplier
			}
		}
		if sourceSha != "" {
			cur := c.hashForNode(edge.To)
			if cur == "" {
				return c.staleMultiplier
			}
			if cur != sourceSha {
				return c.staleMultiplier
			}
		}
		return 1.0
	}

	// SHA stamps absent; fall back to age-based decay.
	if ts := asString(props["generated_at"]); ts != "" && c.maxAge > 0 {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			if time.Since(t) > c.maxAge {
				return c.staleMultiplier
			}
		}
	}

	return 1.0
}

// hashForNode converts a node ID to a file path and returns its git
// blob SHA. Empty string indicates the file cannot be read.
func (c *repoChecker) hashForNode(nodeID string) string {
	rel := c.resolveNodeID(nodeID)
	if rel == "" {
		return ""
	}
	abs := filepath.Join(c.rootDir, rel)
	c.mu.Lock()
	if h, ok := c.cache[abs]; ok {
		c.mu.Unlock()
		return h
	}
	c.mu.Unlock()
	h, err := HashFile(abs)
	if err != nil {
		h = ""
	}
	c.mu.Lock()
	c.cache[abs] = h
	c.mu.Unlock()
	return h
}

// resolveNodeID maps a graph node ID to a repo-relative file path.
// Handles Solidity file nodes via the package-level NodeIDToPath,
// Go file nodes (go:<module>/<rel>.go) by stripping the module prefix,
// and Rust file nodes (rs:<crate>/<rel>.rs) by looking up the crate's
// manifest dir in the graph and composing it with rootDir.
//
// Returns "" for nodes that don't identify a single file (package
// nodes, crate nodes, module nodes, check nodes) or when the required
// resolver state (goModulePath, graph) is unavailable.
func (c *repoChecker) resolveNodeID(nodeID string) string {
	if rel := NodeIDToPath(nodeID); rel != "" {
		return rel
	}
	if c.goModulePath != "" && strings.HasPrefix(nodeID, "go:") && strings.HasSuffix(nodeID, ".go") {
		path := strings.TrimPrefix(nodeID, "go:")
		if strings.HasPrefix(path, c.goModulePath+"/") {
			return strings.TrimPrefix(path, c.goModulePath+"/")
		}
	}
	if c.graph != nil && strings.HasPrefix(nodeID, "rs:") && strings.HasSuffix(nodeID, ".rs") {
		trimmed := strings.TrimPrefix(nodeID, "rs:")
		if slash := strings.Index(trimmed, "/"); slash > 0 {
			crateName := trimmed[:slash]
			rel := trimmed[slash+1:]
			crateNode := c.graph.GetNode("rs:" + crateName)
			if crateNode != nil {
				if dir, ok := crateNode.Properties["dir"].(string); ok && dir != "" {
					abs := filepath.Join(dir, rel)
					if r, err := filepath.Rel(c.rootDir, abs); err == nil && !strings.HasPrefix(r, "..") {
						return r
					}
				}
			}
		}
	}
	return ""
}

// readGoModulePath returns the module directive from rootDir/go.mod,
// or "" if the file is missing or malformed.
func readGoModulePath(rootDir string) string {
	data, err := os.ReadFile(filepath.Join(rootDir, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

// NodeIDToPath converts a selector graph node ID to a repo-relative
// file path, or "" if the node does not identify a single file.
//
//	sol:test/L1/X.t.sol → packages/contracts-bedrock/test/L1/X.t.sol
//	sol:src/L1/X.sol    → packages/contracts-bedrock/src/L1/X.sol
//	go:<package>        → "" (packages are not single files)
//	rs:<crate>          → "" (crates are not single files)
func NodeIDToPath(nodeID string) string {
	switch {
	case strings.HasPrefix(nodeID, "sol:"):
		return filepath.Join("packages", "contracts-bedrock", strings.TrimPrefix(nodeID, "sol:"))
	default:
		return ""
	}
}

// HashFile computes the git blob SHA of a file on disk. Matches
// `git hash-object <path>` — git prepends "blob <length>\0" before
// SHA-1 hashing.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", err
	}

	h := sha1.New()
	fmt.Fprintf(h, "blob %d\x00", info.Size())
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// HashContent computes the git blob SHA of a byte slice.
func HashContent(data []byte) string {
	h := sha1.New()
	fmt.Fprintf(h, "blob %d\x00", len(data))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func asString(v interface{}) string {
	s, _ := v.(string)
	return s
}
