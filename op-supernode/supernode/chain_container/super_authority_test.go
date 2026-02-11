package chain_container

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
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

var _ activity.VerificationActivity = (*mockVerificationActivityForSuperAuthority)(nil)

// TestChainContainer_FullyVerifiedL2Head_MultipleVerifiers tests that FullyVerifiedL2Head
// returns the block with the minimum (oldest) timestamp across all verifiers
func TestChainContainer_FullyVerifiedL2Head_MultipleVerifiers(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := &simpleChainContainer{
		chainID:   chainID,
		verifiers: []activity.VerificationActivity{},
	}

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
	result := cc.FullyVerifiedL2Head()
	require.Equal(t, verifier1.latestVerifiedBlock, result, "should return oldest verified block")
}

// TestChainContainer_FullyVerifiedL2Head_NoVerifiers tests that FullyVerifiedL2Head
// returns an empty BlockID when there are no verification activities
func TestChainContainer_FullyVerifiedL2Head_NoVerifiers(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := &simpleChainContainer{
		chainID:   chainID,
		verifiers: []activity.VerificationActivity{},
	}

	result := cc.FullyVerifiedL2Head()
	require.Equal(t, eth.BlockID{}, result, "should return empty BlockID with no verifiers")
}

// TestChainContainer_FullyVerifiedL2Head_OneUnverified tests that FullyVerifiedL2Head
// returns an empty BlockID if any verifier returns an unverified state
func TestChainContainer_FullyVerifiedL2Head_OneUnverified(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := &simpleChainContainer{
		chainID:   chainID,
		verifiers: []activity.VerificationActivity{},
	}

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

	// Should return empty BlockID (conservative approach)
	result := cc.FullyVerifiedL2Head()
	require.Equal(t, eth.BlockID{}, result, "should return empty BlockID when any verifier is unverified")
}

// TestChainContainer_FullyVerifiedL2Head_SameTimestamp tests that FullyVerifiedL2Head
// panics when multiple verifiers report the same timestamp but different block hashes
func TestChainContainer_FullyVerifiedL2Head_SameTimestamp(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := &simpleChainContainer{
		chainID:   chainID,
		verifiers: []activity.VerificationActivity{},
	}

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
		cc.FullyVerifiedL2Head()
	}, "should panic when verifiers disagree on block hash for same timestamp")
}

// TestChainContainer_FullyVerifiedL2Head_SingleVerifier tests the simple case
// with just one verification activity
func TestChainContainer_FullyVerifiedL2Head_SingleVerifier(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := &simpleChainContainer{
		chainID:   chainID,
		verifiers: []activity.VerificationActivity{},
	}

	verifier := &mockVerificationActivityForSuperAuthority{
		latestVerifiedBlock: eth.BlockID{Hash: [32]byte{1}, Number: 100},
		latestVerifiedTS:    1000,
	}

	cc.verifiers = []activity.VerificationActivity{verifier}

	result := cc.FullyVerifiedL2Head()
	require.Equal(t, verifier.latestVerifiedBlock, result, "should return the single verifier's block")
}

// TestChainContainer_FullyVerifiedL2Head_AllUnverified tests that an empty BlockID
// is returned when all verifiers are unverified
func TestChainContainer_FullyVerifiedL2Head_AllUnverified(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(420)
	cc := &simpleChainContainer{
		chainID:   chainID,
		verifiers: []activity.VerificationActivity{},
	}

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

	result := cc.FullyVerifiedL2Head()
	require.Equal(t, eth.BlockID{}, result, "should return empty BlockID when all verifiers are unverified")
}
