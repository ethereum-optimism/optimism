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
func New(rootDir string, p *policy.Policy) Checker {
	return &repoChecker{
		rootDir:         rootDir,
		staleMultiplier: p.Freshness.StaleMultiplier,
		maxAge:          time.Duration(p.Freshness.MaxAgeDays) * 24 * time.Hour,
		cache:           make(map[string]string),
	}
}

type repoChecker struct {
	rootDir         string
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
	rel := NodeIDToPath(nodeID)
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
