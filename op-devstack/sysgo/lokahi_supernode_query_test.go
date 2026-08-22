package sysgo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// lokahiQueryFixture names the wire-parity fixture this package shares with lokahi's own
// tests.
//
// The Rust side pulls this exact file in with include_str! (rust/lokahi/src/query/mod.rs) and
// asserts that serializing its two responses produces it. This test asserts the other half: that
// what is in the file decodes into the Go types the real consumers use, and that the two
// commitments in it are the ones Go computes over the same preimages.
//
// Neither half is worth much alone. Together they say that a super root lokahi publishes is
// bit-identical to the one op-supernode would publish over the same chain outputs, which is the
// only property that makes lokahi safe to put behind op-proposer or op-challenger — a super root
// that is well-formed but computed differently is worse than a missing one, because it looks
// right.
const lokahiQueryFixture = "lokahi_supernode_query.json"

// readLokahiQueryFixture decodes one method's response out of the fixture.
func readLokahiQueryFixture(t *testing.T, method string, out any) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", lokahiQueryFixture))
	require.NoError(t, err, "read the lokahi query parity fixture")

	var doc map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &doc), "the fixture is a JSON object of responses")
	body, ok := doc[method]
	require.True(t, ok, "the fixture has a %s response", method)
	require.NoError(t, json.Unmarshal(body, out), "decode the %s response", method)
}

// TestLokahiSyncStatusDecodesIntoGoTypes checks that what lokahi serves for
// supernode_syncStatus is what eth.SuperNodeSyncStatusResponse reads.
func TestLokahiSyncStatusDecodesIntoGoTypes(t *testing.T) {
	t.Parallel()

	var status eth.SuperNodeSyncStatusResponse
	readLokahiQueryFixture(t, "supernode_syncStatus", &status)

	chainA, chainB := eth.ChainIDFromUInt64(901), eth.ChainIDFromUInt64(902)
	require.Equal(t, []eth.ChainID{chainA, chainB}, status.ChainIDs, "chain ids, ascending")
	require.Len(t, status.Chains, 2)

	// The aggregate is the minimum over the set, including the verifier's L1 progress: the
	// verifier here is at 896, below both chains, so it is what current_l1 resolves to.
	require.Equal(t, uint64(896), status.CurrentL1.Number)
	require.Equal(t, uint64(1_700_000_030), status.SafeTimestamp, "the lower chain's safe head")
	require.Equal(t, uint64(1_700_000_046), status.LocalSafeTimestamp)
	require.Equal(t, uint64(1_700_000_000), status.FinalizedTimestamp)

	// Each chain's own status decodes whole, which is what an operator reading `chains` needs.
	a := status.Chains[chainA]
	require.Equal(t, uint64(900), a.CurrentL1.Number)
	require.Equal(t, uint64(4_900), a.SafeL2.Number)
	require.Equal(t, uint64(1_700_000_050), a.LocalSafeL2.Time)
	require.Equal(t, uint64(3), a.LocalSafeL2.SequenceNumber, "l2 refs carry their seq number")
	require.Equal(t, uint64(898), a.LocalSafeL2.L1Origin.Number, "l1origin decodes")

	// One field of eth.SyncStatus has no counterpart in kona's, so it decodes as zero. That
	// is the same thing a Go consumer sees reading a kona-node's own optimism_syncStatus, and it
	// is recorded here so a future consumer that starts depending on it finds this test rather
	// than a silent zero. (CrossUnsafeL2 used to be asserted the same way, until #22555 removed
	// the cross-unsafe chain-head notion from eth.SyncStatus itself.)
	require.Zero(t, a.PendingSafeL2.Number, "kona tracks no pending-safe head")
}

// TestLokahiSuperRootDecodesIntoGoTypes checks the superroot_atTimestamp response, and
// recomputes both of its commitments with the Go implementations.
func TestLokahiSuperRootDecodesIntoGoTypes(t *testing.T) {
	t.Parallel()

	var response eth.SuperRootAtTimestampResponse
	readLokahiQueryFixture(t, "superroot_atTimestamp", &response)

	chainA, chainB := eth.ChainIDFromUInt64(901), eth.ChainIDFromUInt64(902)
	require.Equal(t, []eth.ChainID{chainA, chainB}, response.ChainIDs)
	require.Equal(t, uint64(896), response.CurrentL1.Number)
	require.Equal(t, uint64(1_700_000_030), response.CurrentSafeTimestamp)
	require.Equal(t, uint64(1_700_000_046), response.CurrentLocalSafeTimestamp)
	require.Equal(t, uint64(1_700_000_000), response.CurrentFinalizedTimestamp)

	// The optimistic branch carries each output-root preimage next to its hash. Recomputing the
	// hash from the preimage is what proves lokahi's output-root encoding is eth.OutputRoot's:
	// op-challenger hashes the preimage itself at every step of a dispute.
	require.Len(t, response.OptimisticAtTimestamp, 2)
	for _, chainID := range response.ChainIDs {
		entry, ok := response.OptimisticAtTimestamp[chainID]
		require.True(t, ok, "chain %s is in the optimistic branch", chainID)
		require.NotNil(t, entry.Output, "the output-root preimage is present")
		require.Equal(t, eth.OutputRoot(entry.Output), entry.OutputRoot,
			"chain %s: output_root must be the keccak of the OutputV0 it shows", chainID)
		require.NotZero(t, entry.RequiredL1.Number)
	}

	// The super root itself. eth.SuperRoot over the decoded preimage has to reproduce the hash
	// lokahi published, or a proposer would commit on chain to a root nobody else computes.
	require.NotNil(t, response.Data, "the verified branch publishes a super root")
	require.Equal(t, uint64(890), response.Data.VerifiedRequiredL1.Number)
	require.NotNil(t, response.Data.Super)
	require.Equal(t, eth.SuperRootVersionV1, response.Data.Super.Version())
	require.Equal(t, eth.SuperRoot(response.Data.Super), response.Data.SuperRoot,
		"super_root must be the keccak of the SuperV1 preimage it shows")

	superV1, ok := response.Data.Super.(*eth.SuperV1)
	require.True(t, ok, "the super root is a V1")
	require.Equal(t, uint64(1_700_000_040), superV1.Timestamp)
	require.Equal(t, []eth.ChainID{chainA, chainB},
		[]eth.ChainID{superV1.Chains[0].ChainID, superV1.Chains[1].ChainID},
		"the preimage is ordered by chain id, which is what the hash commits to")

	// The verified branch reads its output roots at the heads the verifier committed to, which
	// are not the blocks the optimistic branch reports. If lokahi ever served the optimistic
	// roots here the super root would be over the wrong branch, so the fixture keeps the two
	// distinguishable and this asserts they stayed distinct.
	for i, chainID := range response.ChainIDs {
		require.NotEqual(t, response.OptimisticAtTimestamp[chainID].OutputRoot,
			superV1.Chains[i].Output,
			"chain %s: the verified root is not the optimistic one", chainID)
	}
}

// lokahi needs no Go client of its own: op-service/sources.SuperNodeClient already implements
// the query API, so the Go side of this feature is plumbing and the wire is where the parity
// question lives.
var _ apis.SupernodeQueryAPI = (*sources.SuperNodeClient)(nil)
