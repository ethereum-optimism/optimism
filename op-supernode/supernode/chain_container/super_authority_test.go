package chain_container

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// mockVerificationActivityForSuperAuthority provides controlled test data for SuperAuthority tests
type mockVerificationActivityForSuperAuthority struct {
	latestVerifiedBlock eth.BlockID
	latestVerifiedTS    uint64
}

func (m *mockVerificationActivityForSuperAuthority) Start(ctx context.Context) error { return nil }
func (m *mockVerificationActivityForSuperAuthority) Stop(ctx context.Context) error  { return nil }
func (m *mockVerificationActivityForSuperAuthority) Name() string                    { return "mock" }
func (m *mockVerificationActivityForSuperAuthority) CurrentL1() eth.BlockID {
	return eth.BlockID{}
}
func (m *mockVerificationActivityForSuperAuthority) VerifiedAtTimestamp(ts uint64) (bool, error) {
	return false, nil
}
func (m *mockVerificationActivityForSuperAuthority) LatestVerifiedL2Block(chainID eth.ChainID) (eth.BlockID, uint64) {
	return m.latestVerifiedBlock, m.latestVerifiedTS
}
func (m *mockVerificationActivityForSuperAuthority) Reset(eth.ChainID, uint64, eth.BlockRef) {}

var _ activity.VerificationActivity = (*mockVerificationActivityForSuperAuthority)(nil)

// newTestChainContainer creates a simpleChainContainer for testing with a test logger
func newTestChainContainer(t *testing.T, chainID eth.ChainID) *simpleChainContainer {
	return &simpleChainContainer{
		chainID:   chainID,
		verifiers: []activity.VerificationActivity{},
		log:       testlog.Logger(t, log.LevelDebug),
	}
}

// TestChainContainer_FullyVerifiedL2Head_MultipleVerifiers tests that FullyVerifiedL2Head
// returns the block with the minimum (oldest) timestamp across all verifiers
func TestChainContainer_FullyVerifiedL2Head_MultipleVerifiers(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	// Setup three verifiers with different timestamps
	verifier1 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{1}, Number: 100},
		latestVerifiedTS:    1000, // oldest
	}
	verifier2 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{2}, Number: 200},
		latestVerifiedTS:    2000, // middle
	}
	verifier3 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{3}, Number: 300},
		latestVerifiedTS:    3000, // newest
	}

	cc.verifiers = []activity.VerificationActivity{verifier1, verifier2, verifier3}

	// Should return the block with minimum timestamp (verifier1)
	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, verifier1.latestVerifiedBlock, result, "should return oldest verified block")
	require.False(t, useLocalSafe, "should not signal fallback when verifiers have verified blocks")
}

// TestChainContainer_FullyVerifiedL2Head_NoVerifiers tests that FullyVerifiedL2Head
// returns an empty BlockID and signals fallback when there are no verification activities
func TestChainContainer_FullyVerifiedL2Head_NoVerifiers(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, eth.BlockID{}, result, "should return empty BlockID with no verifiers")
	require.True(t, useLocalSafe, "should signal fallback to local-safe when no verifiers registered")
}

// TestChainContainer_FullyVerifiedL2Head_OneUnverified tests that FullyVerifiedL2Head
// returns an empty BlockID without signaling fallback if any verifier returns an unverified state
func TestChainContainer_FullyVerifiedL2Head_OneUnverified(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	// Setup verifiers where one is unverified (empty BlockID)
	verifier1 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{1}, Number: 100},
		latestVerifiedTS:    1000,
	}
	verifier2 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{}, // unverified
		latestVerifiedTS:    0,             // zero timestamp
	}
	verifier3 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{3}, Number: 300},
		latestVerifiedTS:    3000,
	}

	cc.verifiers = []activity.VerificationActivity{verifier1, verifier2, verifier3}

	// Should return empty BlockID (conservative approach) but NOT signal fallback
	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, eth.BlockID{}, result, "should return empty BlockID when any verifier is unverified")
	require.False(t, useLocalSafe, "should not signal fallback when verifiers exist but are unverified")
}

// TestChainContainer_FullyVerifiedL2Head_SameTimestamp tests that FullyVerifiedL2Head
// panics when multiple verifiers report the same timestamp but different block hashes
func TestChainContainer_FullyVerifiedL2Head_SameTimestamp(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	// Setup verifiers with same timestamp but different block hashes
	verifier1 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{1}, Number: 100},
		latestVerifiedTS:    1000,
	}
	verifier2 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{2}, Number: 100},
		latestVerifiedTS:    1000, // same timestamp, different hash
	}

	cc.verifiers = []activity.VerificationActivity{verifier1, verifier2}

	// Should panic because verifiers disagree on block hash for same timestamp
	require.Panics(t, func() {
		_, _ = cc.FullyVerifiedL2Head()
	}, "should panic when verifiers disagree on block hash for same timestamp")
}

// TestChainContainer_FullyVerifiedL2Head_SingleVerifier tests the simple case
// with just one verification activity
func TestChainContainer_FullyVerifiedL2Head_SingleVerifier(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	verifier := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{1}, Number: 100},
		latestVerifiedTS:    1000,
	}

	cc.verifiers = []activity.VerificationActivity{verifier}

	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, verifier.latestVerifiedBlock, result, "should return the single verifier's block")
	require.False(t, useLocalSafe, "should not signal fallback when verifier has verified blocks")
}

// TestChainContainer_FullyVerifiedL2Head_AllUnverified tests that an empty BlockID
// is returned without signaling fallback when all verifiers are unverified
func TestChainContainer_FullyVerifiedL2Head_AllUnverified(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := newTestChainContainer(t, chainID)

	// All verifiers unverified
	verifier1 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{},
		latestVerifiedTS:    0,
	}
	verifier2 := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{},
		latestVerifiedTS:    0,
	}

	cc.verifiers = []activity.VerificationActivity{verifier1, verifier2}

	result, useLocalSafe := cc.FullyVerifiedL2Head()
	require.Equal(t, eth.BlockID{}, result, "should return empty BlockID when all verifiers are unverified")
	require.False(t, useLocalSafe, "should not signal fallback when verifiers exist but are unverified")
}
