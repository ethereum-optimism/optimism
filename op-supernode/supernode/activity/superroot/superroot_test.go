package superroot

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type mockCC struct {
	verL2       eth.BlockID
	verL1       eth.BlockID
	optL2       eth.BlockID
	optL1       eth.BlockID
	output      eth.Bytes32
	status      *eth.SyncStatus
	verifierL1s []eth.BlockID

	verifiedErr   error
	outputErr     error
	optimisticErr error
	syncStatusErr error
}

func (m *mockCC) Start(ctx context.Context) error  { return nil }
func (m *mockCC) Stop(ctx context.Context) error   { return nil }
func (m *mockCC) Pause(ctx context.Context) error  { return nil }
func (m *mockCC) Resume(ctx context.Context) error { return nil }

func (m *mockCC) RegisterVerifier(v activity.VerificationActivity) {}
func (m *mockCC) VerifierCurrentL1s() []eth.BlockID {
	return m.verifierL1s
}

func (m *mockCC) LocalSafeBlockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error) {
	return eth.L2BlockRef{}, nil
}
func (m *mockCC) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	if m.syncStatusErr != nil {
		return nil, m.syncStatusErr
	}
	if m.status == nil {
		return &eth.SyncStatus{}, nil
	}
	return m.status, nil
}
func (m *mockCC) SafeHeadAtL1(ctx context.Context, l1BlockNum uint64) (eth.BlockID, eth.BlockID, error) {
	return eth.BlockID{}, eth.BlockID{}, nil
}
func (m *mockCC) L1AtSafeHead(ctx context.Context, l2 eth.BlockID) (eth.BlockID, error) {
	return eth.BlockID{}, nil
}
func (m *mockCC) VerifiedAt(ctx context.Context, ts uint64) (eth.BlockID, eth.BlockID, error) {
	if m.verifiedErr != nil {
		return eth.BlockID{}, eth.BlockID{}, m.verifiedErr
	}
	return m.verL2, m.verL1, nil
}
func (m *mockCC) OptimisticAt(ctx context.Context, ts uint64) (eth.BlockID, eth.BlockID, error) {
	if m.optimisticErr != nil {
		return eth.BlockID{}, eth.BlockID{}, m.optimisticErr
	}
	return m.optL2, m.optL1, nil
}
func (m *mockCC) OutputRootAtL2BlockNumber(ctx context.Context, l2BlockNum uint64) (eth.Bytes32, error) {
	if m.outputErr != nil {
		return eth.Bytes32{}, m.outputErr
	}
	return m.output, nil
}
func (m *mockCC) OptimisticOutputAtTimestamp(ctx context.Context, ts uint64) (*eth.OutputResponse, error) {
	if m.optimisticErr != nil {
		return nil, m.optimisticErr
	}
	// Return minimal output response; tests only assert presence/count
	return &eth.OutputResponse{}, nil
}
func (m *mockCC) RewindEngine(ctx context.Context, timestamp uint64, invalidatedBlock eth.BlockRef) error {
	return nil
}

func (m *mockCC) L1ForL2(ctx context.Context, l2Block eth.BlockID) (eth.BlockID, error) {
	return eth.BlockID{}, nil
}

func (m *mockCC) FetchReceipts(ctx context.Context, blockID eth.BlockID) (eth.BlockInfo, types.Receipts, error) {
	return nil, nil, nil
}

func (m *mockCC) ID() eth.ChainID {
	return eth.ChainIDFromUInt64(10)
}

func (m *mockCC) BlockTime() uint64 { return 1 }
func (m *mockCC) InvalidateBlock(ctx context.Context, height uint64, payloadHash common.Hash) (bool, error) {
	return false, nil
}
func (m *mockCC) IsDenied(height uint64, payloadHash common.Hash) (bool, error) {
	return false, nil
}
func (m *mockCC) SetResetCallback(cb cc.ResetCallback) {}

var _ cc.ChainContainer = (*mockCC)(nil)

func TestSuperroot_AtTimestamp_Succeeds(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			verL2:  eth.BlockID{Number: 100},
			verL1:  eth.BlockID{Number: 1000},
			optL2:  eth.BlockID{Number: 100},
			optL1:  eth.BlockID{Number: 1000},
			output: eth.Bytes32{},
			status: &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 2000},
				LocalSafeL2: eth.L2BlockRef{Time: 200},
				FinalizedL2: eth.L2BlockRef{Time: 150},
			},
		},
		eth.ChainIDFromUInt64(420): &mockCC{
			verL2:  eth.BlockID{Number: 200},
			verL1:  eth.BlockID{Number: 1100},
			optL2:  eth.BlockID{Number: 200},
			optL1:  eth.BlockID{Number: 1100},
			output: eth.Bytes32{},
			status: &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 2100},
				LocalSafeL2: eth.L2BlockRef{Time: 180},
				FinalizedL2: eth.L2BlockRef{Time: 140},
			},
		},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	out, err := api.AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	require.Len(t, out.OptimisticAtTimestamp, 2)
	// min values
	require.Equal(t, uint64(2000), out.CurrentL1.Number)
	require.Equal(t, uint64(180), out.CurrentSafeTimestamp)
	require.Equal(t, uint64(140), out.CurrentFinalizedTimestamp)
	require.Equal(t, uint64(1000), out.Data.VerifiedRequiredL1.Number)
	// With zero outputs, the superroot will be deterministic, just ensure it's set
	_ = out.Data.SuperRoot
}

func TestSuperroot_AtTimestamp_ComputesSuperRoot(t *testing.T) {
	t.Parallel()
	out1 := eth.Bytes32{1}
	out2 := eth.Bytes32{2}
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			verL2:  eth.BlockID{Number: 100},
			verL1:  eth.BlockID{Number: 1000},
			optL2:  eth.BlockID{Number: 100},
			optL1:  eth.BlockID{Number: 1000},
			output: out1,
			status: &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2000}},
		},
		eth.ChainIDFromUInt64(420): &mockCC{
			verL2:  eth.BlockID{Number: 200},
			verL1:  eth.BlockID{Number: 1100},
			optL2:  eth.BlockID{Number: 200},
			optL1:  eth.BlockID{Number: 1100},
			output: out2,
			status: &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2100}},
		},
	}
	ts := uint64(123)
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	resp, err := api.AtTimestamp(context.Background(), hexutil.Uint64(ts))
	require.NoError(t, err)

	// Compute expected super root
	chainOutputs := []eth.ChainIDAndOutput{
		{ChainID: eth.ChainIDFromUInt64(10), Output: out1},
		{ChainID: eth.ChainIDFromUInt64(420), Output: out2},
	}
	expected := eth.SuperRoot(eth.NewSuperV1(ts, chainOutputs...))
	require.Equal(t, expected, resp.Data.SuperRoot)
}

func TestSuperroot_AtTimestamp_ErrorOnCurrentL1(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			syncStatusErr: assertErr(),
		},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	_, err := api.AtTimestamp(context.Background(), 123)
	require.Error(t, err)
}

func TestSuperroot_AtTimestamp_ErrorOnVerifiedAt(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			verifiedErr: assertErr(),
		},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	_, err := api.AtTimestamp(context.Background(), 123)
	require.Error(t, err)
}

func TestSuperroot_AtTimestamp_NotFoundOnVerifiedAt(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			verifiedErr: fmt.Errorf("nope: %w", ethereum.NotFound),
			optL2:       eth.BlockID{Number: 100},
			optL1:       eth.BlockID{Number: 1000},
		},
		eth.ChainIDFromUInt64(11): &mockCC{
			verL2:  eth.BlockID{Number: 200},
			verL1:  eth.BlockID{Number: 1100},
			optL2:  eth.BlockID{Number: 200},
			optL1:  eth.BlockID{Number: 1100},
			output: eth.Bytes32{0x12},
			status: &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2100}},
		},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	actual, err := api.AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	require.Nil(t, actual.Data)
	// Chain 10 has no verified data but optimistic data is available, so it should be present
	require.Contains(t, actual.OptimisticAtTimestamp, eth.ChainIDFromUInt64(10))
	require.Contains(t, actual.OptimisticAtTimestamp, eth.ChainIDFromUInt64(11))
}

func TestSuperroot_AtTimestamp_NotFoundOnVerifiedAtAndOptimisticAt(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			verifiedErr:   fmt.Errorf("nope: %w", ethereum.NotFound),
			optimisticErr: ethereum.NotFound,
		},
		eth.ChainIDFromUInt64(11): &mockCC{
			verL2:  eth.BlockID{Number: 200},
			verL1:  eth.BlockID{Number: 1100},
			optL2:  eth.BlockID{Number: 200},
			optL1:  eth.BlockID{Number: 1100},
			output: eth.Bytes32{0x12},
			status: &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2100}},
		},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	actual, err := api.AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	require.Nil(t, actual.Data)
	// Chain 10 has neither verified nor optimistic data, so it should be absent
	require.NotContains(t, actual.OptimisticAtTimestamp, eth.ChainIDFromUInt64(10))
	require.Contains(t, actual.OptimisticAtTimestamp, eth.ChainIDFromUInt64(11))
}

func TestSuperroot_AtTimestamp_ErrorOnOptimisticAtWhenVerifiedNotFound(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			verifiedErr:   fmt.Errorf("nope: %w", ethereum.NotFound),
			optimisticErr: assertErr(),
		},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	_, err := api.AtTimestamp(context.Background(), 123)
	require.Error(t, err)
}

func TestSuperroot_AtTimestamp_ErrorOnOutputRoot(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			verL2:     eth.BlockID{Number: 100},
			outputErr: assertErr(),
		},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	_, err := api.AtTimestamp(context.Background(), 123)
	require.Error(t, err)
}

func TestSuperroot_AtTimestamp_ErrorOnOptimisticAt(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			verL2:         eth.BlockID{Number: 100},
			output:        eth.Bytes32{1},
			optimisticErr: assertErr(),
		},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	_, err := api.AtTimestamp(context.Background(), 123)
	require.Error(t, err)
}

func TestSuperroot_AtTimestamp_EmptyChains(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	out, err := api.AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	require.Len(t, out.OptimisticAtTimestamp, 0)
}

func TestSuperroot_AtTimestamp_VerifierL1ReducesCurrentL1(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			verL2:  eth.BlockID{Number: 100},
			verL1:  eth.BlockID{Number: 1000},
			optL2:  eth.BlockID{Number: 100},
			optL1:  eth.BlockID{Number: 1000},
			output: eth.Bytes32{},
			status: &eth.SyncStatus{
				CurrentL1: eth.L1BlockRef{Number: 2000},
			},
			// Verifier has only processed up to L1 block 1500, which is less than derivation's 2000
			verifierL1s: []eth.BlockID{{Number: 1500}},
		},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	out, err := api.AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	// CurrentL1 should be 1500 (verifier), not 2000 (derivation)
	require.Equal(t, uint64(1500), out.CurrentL1.Number)
}

func TestSuperroot_AtTimestamp_VerifierL1HigherThanDerivationDoesNotIncrease(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			verL2:  eth.BlockID{Number: 100},
			verL1:  eth.BlockID{Number: 1000},
			optL2:  eth.BlockID{Number: 100},
			optL1:  eth.BlockID{Number: 1000},
			output: eth.Bytes32{},
			status: &eth.SyncStatus{
				CurrentL1: eth.L1BlockRef{Number: 2000},
			},
			// Verifier is ahead of derivation — should not increase the minimum
			verifierL1s: []eth.BlockID{{Number: 3000}},
		},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	out, err := api.AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	// CurrentL1 should still be 2000 (derivation), since verifier is ahead
	require.Equal(t, uint64(2000), out.CurrentL1.Number)
}

func TestSuperroot_SyncStatus_Succeeds(t *testing.T) {
	t.Parallel()
	chainA := eth.ChainIDFromUInt64(10)
	chainB := eth.ChainIDFromUInt64(420)

	chains := map[eth.ChainID]cc.ChainContainer{
		chainA: &mockCC{
			verL2:  eth.BlockID{Number: 90},
			verL1:  eth.BlockID{Number: 1000},
			optL2:  eth.BlockID{Number: 90},
			optL1:  eth.BlockID{Number: 1000},
			output: eth.Bytes32{0xaa},
			status: &eth.SyncStatus{
				CurrentL1:     eth.L1BlockRef{Number: 2000},
				UnsafeL2:      eth.L2BlockRef{Number: 120, Time: 220},
				CrossUnsafeL2: eth.L2BlockRef{Number: 118, Time: 205},
				SafeL2:        eth.L2BlockRef{Number: 110, Time: 170},
				LocalSafeL2:   eth.L2BlockRef{Number: 111, Time: 180},
				FinalizedL2:   eth.L2BlockRef{Number: 100, Time: 140},
			},
		},
		chainB: &mockCC{
			verL2:  eth.BlockID{Number: 100},
			verL1:  eth.BlockID{Number: 1100},
			optL2:  eth.BlockID{Number: 100},
			optL1:  eth.BlockID{Number: 1100},
			output: eth.Bytes32{0xbb},
			status: &eth.SyncStatus{
				CurrentL1:     eth.L1BlockRef{Number: 2000},
				UnsafeL2:      eth.L2BlockRef{Number: 130, Time: 230},
				CrossUnsafeL2: eth.L2BlockRef{Number: 128, Time: 215},
				SafeL2:        eth.L2BlockRef{Number: 112, Time: 175},
				LocalSafeL2:   eth.L2BlockRef{Number: 113, Time: 190},
				FinalizedL2:   eth.L2BlockRef{Number: 101, Time: 150},
			},
		},
	}

	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	out, err := api.SyncStatus(context.Background())
	require.NoError(t, err)

	require.Len(t, out.Chains, 2)
	require.Contains(t, out.Chains, chainA)
	require.Contains(t, out.Chains, chainB)
	require.Equal(t, []eth.ChainID{chainA, chainB}, out.ChainIDs)
	require.Equal(t, uint64(2000), out.CurrentL1.Number)
	require.Equal(t, out.Chains[chainA].CurrentL1.ID(), out.CurrentL1)
	require.Equal(t, out.Chains[chainB].CurrentL1.ID(), out.CurrentL1)
	require.Equal(t, uint64(170), out.SafeTimestamp)
	require.Equal(t, uint64(180), out.LocalSafeTimestamp)
	require.Equal(t, uint64(140), out.FinalizedTimestamp)

	// Ensure per-chain head ordering can be strictly unsafe > safe > finalized.
	statusA := out.Chains[chainA]
	require.Greater(t, statusA.UnsafeL2.Number, statusA.SafeL2.Number)
	require.Greater(t, statusA.SafeL2.Number, statusA.FinalizedL2.Number)
	require.Greater(t, statusA.LocalSafeL2.Number, statusA.SafeL2.Number)
}

func TestSuperroot_SyncStatus_PanicsOnCurrentL1Mismatch(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			status: &eth.SyncStatus{
				CurrentL1: eth.L1BlockRef{Number: 100, Hash: common.Hash{0x11}},
			},
		},
		eth.ChainIDFromUInt64(11): &mockCC{
			status: &eth.SyncStatus{
				CurrentL1: eth.L1BlockRef{Number: 101, Hash: common.Hash{0x22}},
			},
		},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	require.Panics(t, func() {
		_, _ = api.SyncStatus(context.Background())
	})
}

func TestSuperroot_SyncStatus_ErrorOnCurrentL1(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			syncStatusErr: assertErr(),
		},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	_, err := api.SyncStatus(context.Background())
	require.Error(t, err)
}

func TestSuperroot_SyncStatus_IgnoresUnsafeOutputRootErrors(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			// SyncStatus should not require output-root lookups.
			verifiedErr: fmt.Errorf("not available: %w", ethereum.NotFound),
			outputErr:   assertErr(),
			status: &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 100},
				UnsafeL2:    eth.L2BlockRef{Number: 10, Time: 20},
				LocalSafeL2: eth.L2BlockRef{Number: 9, Time: 18},
				FinalizedL2: eth.L2BlockRef{Number: 8, Time: 16},
			},
		},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	out, err := api.SyncStatus(context.Background())
	require.NoError(t, err)
	require.Equal(t, []eth.ChainID{eth.ChainIDFromUInt64(10)}, out.ChainIDs)
	require.Equal(t, eth.BlockID{Number: 100}, out.CurrentL1)
	require.Equal(t, out.Chains[eth.ChainIDFromUInt64(10)].CurrentL1.ID(), out.CurrentL1)
	require.Equal(t, uint64(0), out.SafeTimestamp)
	require.Equal(t, uint64(18), out.LocalSafeTimestamp)
}

func TestSuperroot_SyncStatus_EmptyChains(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}

	out, err := api.SyncStatus(context.Background())
	require.NoError(t, err)
	require.Len(t, out.Chains, 0)
	require.Len(t, out.ChainIDs, 0)
	require.Equal(t, eth.BlockID{}, out.CurrentL1)
	require.Equal(t, uint64(0), out.SafeTimestamp)
	require.Equal(t, uint64(0), out.LocalSafeTimestamp)
	require.Equal(t, uint64(0), out.FinalizedTimestamp)
}

// assertErr returns a generic error instance used to signal mock failures.
func assertErr() error { return fmt.Errorf("mock error") }
