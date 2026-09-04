package silhouette

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestV1ExampleConfigLoads keeps docs/RUNNING-V1.md honest.
//
// The example under example/v1 is what a reader copies, so it is loaded through the SAME
// LoadManifest/LoadConfig path the supernode uses at startup, with DisallowUnknownFields and Check
// both in force. A documented config that has quietly stopped parsing is worse than no example: it
// costs the reader an afternoon before they suspect the document rather than themselves.
//
// The assertions below are about the SHAPE of a v1 deployment rather than about these particular
// addresses. Placeholders are fine; a placeholder in the wrong field is not.
func TestV1ExampleConfigLoads(t *testing.T) {
	m, err := LoadManifest("example/v1/manifest.json")
	require.NoError(t, err, "the v1 example must load through the real manifest path")

	require.Len(t, m.Chains, 1,
		"a two-chain v1 deployment declares ONE silhouette chain: chain A's absence from this file is "+
			"what makes it an ordinary derived chain, and is the whole shape of the example")

	decl, ok := m.Lookup(eth.ChainIDFromUInt64(424247))
	require.True(t, ok)
	require.NoError(t, decl.CheckRole(), "a silhouette supernode has one verifier-only role")

	cfg := decl.Config()
	require.NotNil(t, cfg)
	require.Equal(t, ProofTypeAttested, cfg.ProofType,
		"the v1 example must be attested, or the document it illustrates is about something else")
	require.True(t, cfg.DependenciesVerified(),
		"the example must pin a wire version that carries the import list, so P's dependencies are judged")
	require.NotEqual(t, cfg.Submitter, cfg.Inbox)
}
