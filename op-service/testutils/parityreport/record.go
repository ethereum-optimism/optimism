// Package parityreport records how kona compares with op-node on each derivation parity case, and
// renders the records as a report.
//
// The parity suite has three layers, and each writes one JSON line per case to
// <$DERIVATION_PARITY_REPORT_DIR>/<layer>.jsonl:
//   - vectors: op-node writes inputs and its outputs as JSON, and kona's crates decode the same
//     inputs (rust/kona/crates/protocol/derive/tests/parity.rs);
//   - proof: op-node derives crafted L1 data and kona-client proves op-node's safe head
//     (rust/kona/tests/proofs);
//   - node: op-node and kona-node derive the same crafted L1 data side by side
//     (op-acceptance-tests/tests/derivation_parity).
//
// The Rust layer writes the same JSON shape as Record.
package parityreport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// EnvDir names the directory that parity tests write their records to. When it is unset, Write
// does nothing, so the tests run unchanged in CI. It must be an absolute path, because each Go
// test binary runs in its own package directory.
const EnvDir = "DERIVATION_PARITY_REPORT_DIR"

// The layers of the parity suite.
const (
	LayerVectors = "vectors"
	LayerProof   = "proof"
	LayerNode    = "node"
)

// Record is the outcome of one parity case.
type Record struct {
	Layer string `json:"layer"`
	// Group names what the case exercises, such as a decoding step or a test.
	Group string `json:"group"`
	Case  string `json:"case"`
	// Forks names the forks active for the case, when the case runs on more than one.
	Forks string `json:"forks,omitempty"`
	Agree bool   `json:"agree"`
	// Divergence says how kona's behaviour differs from op-node's. It is set only when Agree is
	// false, and cases that show the same difference use the same text.
	Divergence string `json:"divergence,omitempty"`
}

var writeMu sync.Mutex

// Write appends r to <$DERIVATION_PARITY_REPORT_DIR>/<layer>.jsonl, or does nothing when the
// variable is unset.
func Write(t testing.TB, r Record) {
	t.Helper()
	require.Contains(t, []string{LayerVectors, LayerProof, LayerNode}, r.Layer, "unknown layer")
	require.Equal(t, r.Agree, r.Divergence == "", "a record carries a divergence exactly when the clients disagree")
	dir := os.Getenv(EnvDir)
	if dir == "" {
		return
	}
	line, err := json.Marshal(r)
	require.NoError(t, err)

	writeMu.Lock()
	defer writeMu.Unlock()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	f, err := os.OpenFile(filepath.Join(dir, r.Layer+".jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	require.NoError(t, err)
}
