package superroot

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/interop"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// mockCC is a ChainContainer test fake. Field semantics:
//   - optL2/optL1: returned from OptimisticAt.
//   - optOutput:   the OutputV0 returned by OptimisticOutputAtTimestamp.
//   - byHashOutputs: per-block-hash output roots returned by OutputRootAtL2BlockHash.
//     If a hash is missing, byHashFallback is returned.
//   - byHashFallback: default OutputRoot.
//   - byHashErr:     if set, OutputRootAtL2BlockHash returns this error.
//   - optimisticErr: if set, both OptimisticOutputAtTimestamp and OptimisticAt return it.
//   - syncStatusErr: if set, SyncStatus returns it.
//   - status: returned from SyncStatus.
//   - verifierL1s: returned from VerifierCurrentL1s.
//   - byHashCalled: incremented per OutputRootAtL2BlockHash call (for assertion).
type mockCC struct {
	optL2          eth.BlockID
	optL1          eth.BlockID
	optOutput      *eth.OutputV0
	byHashOutputs  map[common.Hash]eth.Bytes32
	byHashFallback eth.Bytes32
	byHashErr      error
	optimisticErr  error
	syncStatusErr  error
	status         *eth.SyncStatus
	verifierL1s    []eth.BlockID
	byHashCalled   int
}

func (m *mockCC) Start(ctx context.Context) error          { return nil }
func (m *mockCC) Stop(ctx context.Context) error           { return nil }
func (m *mockCC) Pause(ctx context.Context) error          { return nil }
func (m *mockCC) Resume(ctx context.Context) error         { return nil }
func (m *mockCC) PauseAndStopVN(ctx context.Context) error { return nil }
func (m *mockCC) ELFinalizedHead(ctx context.Context) (eth.L2BlockRef, error) {
	return eth.L2BlockRef{}, nil
}
func (m *mockCC) RegisterVerifier(v activity.VerificationActivity) {}
func (m *mockCC) VerifierCurrentL1s() []eth.BlockID                { return m.verifierL1s }
func (m *mockCC) LocalSafeBlockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error) {
	return eth.L2BlockRef{}, nil
}
func (m *mockCC) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	if m.syncStatusErr != nil {
		return nil, m.syncStatusErr
	}
	return m.status, nil
}
func (m *mockCC) OptimisticAt(ctx context.Context, ts uint64) (eth.BlockID, eth.BlockID, error) {
	if m.optimisticErr != nil {
		return eth.BlockID{}, eth.BlockID{}, m.optimisticErr
	}
	return m.optL2, m.optL1, nil
}
func (m *mockCC) OutputRootAtL2BlockHash(ctx context.Context, blockHash common.Hash) (eth.Bytes32, error) {
	m.byHashCalled++
	if m.byHashErr != nil {
		return eth.Bytes32{}, m.byHashErr
	}
	if out, ok := m.byHashOutputs[blockHash]; ok {
		return out, nil
	}
	return m.byHashFallback, nil
}
func (m *mockCC) OptimisticOutputAtTimestamp(ctx context.Context, ts uint64) (*eth.OutputV0, error) {
	if m.optimisticErr != nil {
		return nil, m.optimisticErr
	}
	if m.optOutput != nil {
		return m.optOutput, nil
	}
	return &eth.OutputV0{}, nil
}
func (m *mockCC) FetchReceipts(ctx context.Context, blockID eth.BlockID) (eth.BlockInfo, types.Receipts, error) {
	return nil, nil, nil
}
func (m *mockCC) ID() eth.ChainID   { return eth.ChainIDFromUInt64(10) }
func (m *mockCC) BlockTime() uint64 { return 1 }
func (m *mockCC) OutputV0AtBlockNumber(ctx context.Context, l2BlockNum uint64) (*eth.OutputV0, error) {
	return &eth.OutputV0{}, nil
}
func (m *mockCC) GetDeniedOutput(height uint64, payloadHash common.Hash) (*eth.OutputV0, error) {
	return nil, nil
}
func (m *mockCC) PruneDeniedAtOrAfterTimestamp(timestamp uint64) (map[uint64][]common.Hash, error) {
	return nil, nil
}
func (m *mockCC) IsDenied(height uint64, payloadHash common.Hash) (bool, error) { return false, nil }
func (m *mockCC) SetResetCallback(cb cc.ResetCallback)                          {}
func (m *mockCC) TimestampToBlockNumber(ctx context.Context, ts uint64) (uint64, error) {
	return ts, nil
}
func (m *mockCC) BlockNumberToTimestamp(ctx context.Context, blocknum uint64) (uint64, error) {
	return 0, nil
}
func (m *mockCC) FirstSafeHeadTimestamp(ctx context.Context) (uint64, error) {
	return 0, cc.ErrSafeDBNotReady
}
func (m *mockCC) Generation() uint64 { return 0 }

var _ cc.ChainContainer = (*mockCC)(nil)

// mockVerifiedReader is a test fake for interop.VerifiedResultReader.
// Set result to return a VerifiedResult; set err to return an error
// (ethereum.NotFound for "active but not yet verified", interop.ErrNotActive
// for pre-interop fallback, interop.ErrBeforeVerifiedDB for the
// hard-error regime, or any other error for the default branch).
type mockVerifiedReader struct {
	result    interop.VerifiedResult
	currentL1 eth.BlockID
	err       error
}

func (m *mockVerifiedReader) VerifiedResultAtTimestamp(ts uint64) (interop.VerifiedResult, eth.BlockID, error) {
	return m.result, m.currentL1, m.err
}

// preInteropReader always reports the pre-interop fallback regime.
func preInteropReader() interop.VerifiedResultReader {
	return interop.NoopVerifiedResultReader{}
}

// activeUnverifiedReader reports the active-but-not-yet-verified regime
// (handler returns Data == nil but a successful response).
func activeUnverifiedReader() interop.VerifiedResultReader {
	return &mockVerifiedReader{err: ethereum.NotFound}
}

func newSuperroot(chains map[eth.ChainID]cc.ChainContainer, reader interop.VerifiedResultReader) *Superroot {
	if reader == nil {
		reader = preInteropReader()
	}
	return New(gethlog.New(), chains, reader)
}

// ------ Aggregate sync-status tests (regime-agnostic) ------

func TestSuperroot_AtTimestamp_AggregatesSyncStatus(t *testing.T) {
	t.Parallel()
	hashA := common.HexToHash("0xaaaa")
	hashB := common.HexToHash("0xbbbb")
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			optL2: eth.BlockID{Number: 100, Hash: hashA},
			optL1: eth.BlockID{Number: 1000},
			status: &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 2000},
				SafeL2:      eth.L2BlockRef{Time: 190},
				LocalSafeL2: eth.L2BlockRef{Time: 200},
				FinalizedL2: eth.L2BlockRef{Time: 150},
			},
		},
		eth.ChainIDFromUInt64(420): &mockCC{
			optL2: eth.BlockID{Number: 200, Hash: hashB},
			optL1: eth.BlockID{Number: 1100},
			status: &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 2100},
				SafeL2:      eth.L2BlockRef{Time: 170},
				LocalSafeL2: eth.L2BlockRef{Time: 180},
				FinalizedL2: eth.L2BlockRef{Time: 140},
			},
		},
	}
	s := newSuperroot(chains, preInteropReader())
	out, err := (&superrootAPI{s: s}).AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	require.Equal(t, uint64(2000), out.CurrentL1.Number)
	require.Equal(t, uint64(170), out.CurrentSafeTimestamp)
	require.Equal(t, uint64(180), out.CurrentLocalSafeTimestamp)
	require.Equal(t, uint64(140), out.CurrentFinalizedTimestamp)
	require.Len(t, out.OptimisticAtTimestamp, 2)
}

func TestSuperroot_AtTimestamp_ErrorOnCurrentL1(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{syncStatusErr: assertErr()},
	}
	s := newSuperroot(chains, preInteropReader())
	_, err := (&superrootAPI{s: s}).AtTimestamp(context.Background(), 123)
	require.Error(t, err)
}

func TestSuperroot_AtTimestamp_VerifierL1ReducesCurrentL1(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			optL2:       eth.BlockID{Number: 100, Hash: common.HexToHash("0x01")},
			optL1:       eth.BlockID{Number: 1000},
			status:      &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2000}},
			verifierL1s: []eth.BlockID{{Number: 1500}},
		},
	}
	s := newSuperroot(chains, preInteropReader())
	out, err := (&superrootAPI{s: s}).AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	require.Equal(t, uint64(1500), out.CurrentL1.Number)
}

func TestSuperroot_AtTimestamp_VerifierL1HigherThanDerivationDoesNotIncrease(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			optL2:       eth.BlockID{Number: 100, Hash: common.HexToHash("0x01")},
			optL1:       eth.BlockID{Number: 1000},
			status:      &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2000}},
			verifierL1s: []eth.BlockID{{Number: 3000}},
		},
	}
	s := newSuperroot(chains, preInteropReader())
	out, err := (&superrootAPI{s: s}).AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	require.Equal(t, uint64(2000), out.CurrentL1.Number)
}

func TestSuperroot_AtTimestamp_EmptyChains(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{}
	s := newSuperroot(chains, preInteropReader())
	out, err := (&superrootAPI{s: s}).AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	require.Len(t, out.OptimisticAtTimestamp, 0)
}

// ------ Post-interop verified-branch tests ------

func TestSuperroot_AtTimestamp_VerifiedFromDB(t *testing.T) {
	t.Parallel()
	hashA := common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000000a")
	hashB := common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000000b")
	chainA := eth.ChainIDFromUInt64(10)
	chainB := eth.ChainIDFromUInt64(420)
	outA := eth.Bytes32{0xaa}
	outB := eth.Bytes32{0xbb}
	chains := map[eth.ChainID]cc.ChainContainer{
		chainA: &mockCC{
			optL2:          eth.BlockID{Number: 100, Hash: hashA},
			optL1:          eth.BlockID{Number: 900},
			byHashOutputs:  map[common.Hash]eth.Bytes32{hashA: outA},
			byHashFallback: eth.Bytes32{0xee},
			status:         &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2000}},
		},
		chainB: &mockCC{
			optL2:          eth.BlockID{Number: 200, Hash: hashB},
			optL1:          eth.BlockID{Number: 950},
			byHashOutputs:  map[common.Hash]eth.Bytes32{hashB: outB},
			byHashFallback: eth.Bytes32{0xee},
			status:         &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2100}},
		},
	}
	reader := &mockVerifiedReader{
		result: interop.VerifiedResult{
			Timestamp:   123,
			L1Inclusion: eth.BlockID{Number: 1100},
			L2Heads: map[eth.ChainID]eth.BlockID{
				chainA: {Number: 100, Hash: hashA},
				chainB: {Number: 200, Hash: hashB},
			},
		},
		currentL1: eth.BlockID{Number: 2000},
	}
	s := newSuperroot(chains, reader)
	resp, err := (&superrootAPI{s: s}).AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	require.NotNil(t, resp.Data)
	expected := eth.SuperRoot(eth.NewSuperV1(123,
		eth.ChainIDAndOutput{ChainID: chainA, Output: outA},
		eth.ChainIDAndOutput{ChainID: chainB, Output: outB}))
	require.Equal(t, expected, resp.Data.SuperRoot)
	require.Equal(t, uint64(1100), resp.Data.VerifiedRequiredL1.Number)
}

func TestSuperroot_AtTimestamp_VerifiedDBMiss(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			optL2:  eth.BlockID{Number: 100, Hash: common.HexToHash("0x01")},
			optL1:  eth.BlockID{Number: 900},
			status: &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2000}},
		},
	}
	mock := chains[eth.ChainIDFromUInt64(10)].(*mockCC)
	s := newSuperroot(chains, activeUnverifiedReader())
	resp, err := (&superrootAPI{s: s}).AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	require.Nil(t, resp.Data, "Data must be nil for active-but-not-yet-verified")
	require.Equal(t, 0, mock.byHashCalled, "OutputRootAtL2BlockHash must not be called when Data is nil")
}

func TestSuperroot_AtTimestamp_OutputRootByHashFails(t *testing.T) {
	t.Parallel()
	hashA := common.HexToHash("0x0a")
	chainA := eth.ChainIDFromUInt64(10)
	chains := map[eth.ChainID]cc.ChainContainer{
		chainA: &mockCC{
			optL2:     eth.BlockID{Number: 100, Hash: hashA},
			optL1:     eth.BlockID{Number: 900},
			byHashErr: ethereum.NotFound,
			status:    &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2000}},
		},
	}
	reader := &mockVerifiedReader{
		result: interop.VerifiedResult{
			Timestamp:   123,
			L1Inclusion: eth.BlockID{Number: 1000},
			L2Heads:     map[eth.ChainID]eth.BlockID{chainA: {Number: 100, Hash: hashA}},
		},
		currentL1: eth.BlockID{Number: 2000},
	}
	s := newSuperroot(chains, reader)
	_, err := (&superrootAPI{s: s}).AtTimestamp(context.Background(), 123)
	require.Error(t, err, "transient EL-by-hash failure must surface as transport error")
}

func TestSuperroot_AtTimestamp_DepSetSizeMismatch(t *testing.T) {
	t.Parallel()
	chainA := eth.ChainIDFromUInt64(10)
	chainB := eth.ChainIDFromUInt64(420)
	chains := map[eth.ChainID]cc.ChainContainer{
		chainA: &mockCC{
			optL2:  eth.BlockID{Number: 100, Hash: common.HexToHash("0x01")},
			optL1:  eth.BlockID{Number: 900},
			status: &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2000}},
		},
		chainB: &mockCC{
			optL2:  eth.BlockID{Number: 200, Hash: common.HexToHash("0x02")},
			optL1:  eth.BlockID{Number: 950},
			status: &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2100}},
		},
	}
	// VerifiedResult covers only one of the two chains in s.chains.
	reader := &mockVerifiedReader{
		result: interop.VerifiedResult{
			Timestamp:   123,
			L1Inclusion: eth.BlockID{Number: 1000},
			L2Heads:     map[eth.ChainID]eth.BlockID{chainA: {Number: 100, Hash: common.HexToHash("0x01")}},
		},
		currentL1: eth.BlockID{Number: 2000},
	}
	s := newSuperroot(chains, reader)
	_, err := (&superrootAPI{s: s}).AtTimestamp(context.Background(), 123)
	require.Error(t, err)
	require.Contains(t, err.Error(), "dep-set size mismatch")
}

func TestSuperroot_AtTimestamp_DepSetKeyMismatch(t *testing.T) {
	t.Parallel()
	chainA := eth.ChainIDFromUInt64(10)
	chainB := eth.ChainIDFromUInt64(420)
	chainC := eth.ChainIDFromUInt64(999)
	chains := map[eth.ChainID]cc.ChainContainer{
		chainA: &mockCC{
			optL2:  eth.BlockID{Number: 100, Hash: common.HexToHash("0x01")},
			optL1:  eth.BlockID{Number: 900},
			status: &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2000}},
		},
		chainB: &mockCC{
			optL2:  eth.BlockID{Number: 200, Hash: common.HexToHash("0x02")},
			optL1:  eth.BlockID{Number: 950},
			status: &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2100}},
		},
	}
	// Same length as s.chains, but a different chain ID.
	reader := &mockVerifiedReader{
		result: interop.VerifiedResult{
			Timestamp:   123,
			L1Inclusion: eth.BlockID{Number: 1000},
			L2Heads: map[eth.ChainID]eth.BlockID{
				chainA: {Number: 100, Hash: common.HexToHash("0x01")},
				chainC: {Number: 200, Hash: common.HexToHash("0x02")},
			},
		},
		currentL1: eth.BlockID{Number: 2000},
	}
	s := newSuperroot(chains, reader)
	_, err := (&superrootAPI{s: s}).AtTimestamp(context.Background(), 123)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing chain")
}

// Data is returned even when CurrentL1 sits below VerifiedRequiredL1 (e.g.
// mid-rewind, where the verifier has rolled CurrentL1 back below T's
// required L1 but the entry has not yet been deleted). Callers gate trust
// on CurrentL1 themselves.
func TestSuperroot_AtTimestamp_VerifiedDataReturnedBelowRequiredL1(t *testing.T) {
	t.Parallel()
	hashA := common.HexToHash("0x0a")
	chainA := eth.ChainIDFromUInt64(10)
	chains := map[eth.ChainID]cc.ChainContainer{
		chainA: &mockCC{
			optL2:          eth.BlockID{Number: 100, Hash: hashA},
			optL1:          eth.BlockID{Number: 900},
			byHashOutputs:  map[common.Hash]eth.Bytes32{hashA: {0xaa}},
			byHashFallback: eth.Bytes32{0xee},
			status:         &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2000}},
		},
	}
	reader := &mockVerifiedReader{
		result: interop.VerifiedResult{
			Timestamp:   123,
			L1Inclusion: eth.BlockID{Number: 1000},
			L2Heads:     map[eth.ChainID]eth.BlockID{chainA: {Number: 100, Hash: hashA}},
		},
		// Snapshot caught currentL1 already at zero (mid-rewind).
		currentL1: eth.BlockID{Number: 0},
	}
	s := newSuperroot(chains, reader)
	resp, err := (&superrootAPI{s: s}).AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	require.NotNil(t, resp.Data)
	require.Equal(t, uint64(1000), resp.Data.VerifiedRequiredL1.Number)
	require.Equal(t, uint64(0), resp.CurrentL1.Number,
		"response CurrentL1 must be min(aggregate, snapshot) so it doesn't overstate verifier progress")
}

func TestSuperroot_AtTimestamp_VerifiedReaderError(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			optL2:  eth.BlockID{Number: 100, Hash: common.HexToHash("0x01")},
			optL1:  eth.BlockID{Number: 900},
			status: &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2000}},
		},
	}
	reader := &mockVerifiedReader{err: errors.New("verifiedDB bbolt corrupted")}
	s := newSuperroot(chains, reader)
	_, err := (&superrootAPI{s: s}).AtTimestamp(context.Background(), 123)
	require.Error(t, err)
}

func TestSuperroot_AtTimestamp_OptimisticErrorFailsRPC_VerifiedHit(t *testing.T) {
	t.Parallel()
	hashA := common.HexToHash("0x0a")
	chainA := eth.ChainIDFromUInt64(10)
	chains := map[eth.ChainID]cc.ChainContainer{
		chainA: &mockCC{
			optimisticErr:  fmt.Errorf("EL transient: %w", context.DeadlineExceeded),
			byHashOutputs:  map[common.Hash]eth.Bytes32{hashA: {0xaa}},
			byHashFallback: eth.Bytes32{0xee},
			status:         &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2000}},
		},
	}
	reader := &mockVerifiedReader{
		result: interop.VerifiedResult{
			Timestamp:   123,
			L1Inclusion: eth.BlockID{Number: 1000},
			L2Heads:     map[eth.ChainID]eth.BlockID{chainA: {Number: 100, Hash: hashA}},
		},
		currentL1: eth.BlockID{Number: 2000},
	}
	s := newSuperroot(chains, reader)
	_, err := (&superrootAPI{s: s}).AtTimestamp(context.Background(), 123)
	require.Error(t, err, "optimistic-branch transport error must fail the RPC even with a verified hit (silently emitting a partial map would corrupt op-challenger step>0 logic)")
}

func TestSuperroot_AtTimestamp_OptimisticErrorFailsRPC_UnverifiedMiss(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			optimisticErr: fmt.Errorf("EL transient: %w", context.DeadlineExceeded),
			status:        &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2000}},
		},
	}
	s := newSuperroot(chains, activeUnverifiedReader())
	_, err := (&superrootAPI{s: s}).AtTimestamp(context.Background(), 123)
	require.Error(t, err)
}

// ------ Pre-interop fallback tests ------

func TestSuperroot_AtTimestamp_PreInteropFallback_NoopReader(t *testing.T) {
	t.Parallel()
	chainA := eth.ChainIDFromUInt64(10)
	chainB := eth.ChainIDFromUInt64(420)
	chains := map[eth.ChainID]cc.ChainContainer{
		chainA: &mockCC{
			optL2:     eth.BlockID{Number: 100, Hash: common.HexToHash("0x01")},
			optL1:     eth.BlockID{Number: 900},
			optOutput: &eth.OutputV0{StateRoot: eth.Bytes32{0x01}},
			status:    &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2000}},
		},
		chainB: &mockCC{
			optL2:     eth.BlockID{Number: 200, Hash: common.HexToHash("0x02")},
			optL1:     eth.BlockID{Number: 950},
			optOutput: &eth.OutputV0{StateRoot: eth.Bytes32{0x02}},
			status:    &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2100}},
		},
	}
	mockA := chains[chainA].(*mockCC)
	mockB := chains[chainB].(*mockCC)
	s := newSuperroot(chains, preInteropReader())
	resp, err := (&superrootAPI{s: s}).AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	require.NotNil(t, resp.Data)
	// VerifiedRequiredL1 is max(optL1).
	require.Equal(t, uint64(950), resp.Data.VerifiedRequiredL1.Number)
	// Data.Super.Chains[c].Output must byte-equal OptimisticAtTimestamp[c].OutputRoot
	// because the pre-interop path reuses the optimistic map.
	for _, c := range resp.Data.Super.(*eth.SuperV1).Chains {
		require.Equal(t, resp.OptimisticAtTimestamp[c.ChainID].OutputRoot, c.Output,
			"pre-interop Data.Super.Chains[c].Output must byte-equal OptimisticAtTimestamp[c].OutputRoot")
	}
	// No second per-chain by-hash fetch — pre-interop reuses optimistic data.
	require.Equal(t, 0, mockA.byHashCalled)
	require.Equal(t, 0, mockB.byHashCalled)
}

func TestSuperroot_AtTimestamp_PreInteropFallback_OptimisticMapShort(t *testing.T) {
	t.Parallel()
	chainA := eth.ChainIDFromUInt64(10)
	chainB := eth.ChainIDFromUInt64(420)
	chains := map[eth.ChainID]cc.ChainContainer{
		chainA: &mockCC{
			optL2:     eth.BlockID{Number: 100, Hash: common.HexToHash("0x01")},
			optL1:     eth.BlockID{Number: 900},
			optOutput: &eth.OutputV0{StateRoot: eth.Bytes32{0x01}},
			status:    &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2000}},
		},
		chainB: &mockCC{
			// Chain B hasn't derived T yet.
			optimisticErr: ethereum.NotFound,
			status:        &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2100}},
		},
	}
	s := newSuperroot(chains, preInteropReader())
	resp, err := (&superrootAPI{s: s}).AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	require.Nil(t, resp.Data, "Data must be nil when at least one chain hasn't derived T")
	require.Contains(t, resp.OptimisticAtTimestamp, chainA)
	require.NotContains(t, resp.OptimisticAtTimestamp, chainB)
}

func TestSuperroot_AtTimestamp_PreInteropFallback_BelowActivation(t *testing.T) {
	t.Parallel()
	chainA := eth.ChainIDFromUInt64(10)
	chains := map[eth.ChainID]cc.ChainContainer{
		chainA: &mockCC{
			optL2:     eth.BlockID{Number: 100, Hash: common.HexToHash("0x01")},
			optL1:     eth.BlockID{Number: 900},
			optOutput: &eth.OutputV0{StateRoot: eth.Bytes32{0x01}},
			status:    &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2000}},
		},
	}
	reader := &mockVerifiedReader{err: interop.ErrNotActive}
	s := newSuperroot(chains, reader)
	resp, err := (&superrootAPI{s: s}).AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	require.NotNil(t, resp.Data)
}

func TestSuperroot_AtTimestamp_PreInteropFallback_NoSecondFetch(t *testing.T) {
	t.Parallel()
	chainA := eth.ChainIDFromUInt64(10)
	chains := map[eth.ChainID]cc.ChainContainer{
		chainA: &mockCC{
			optL2:     eth.BlockID{Number: 100, Hash: common.HexToHash("0x01")},
			optL1:     eth.BlockID{Number: 900},
			optOutput: &eth.OutputV0{StateRoot: eth.Bytes32{0x01}},
			// Setting byHashErr proves the pre-interop fallback does NOT call
			// OutputRootAtL2BlockHash — otherwise the test would fail.
			byHashErr: fmt.Errorf("BUG: pre-interop fallback must not call by-hash"),
			status:    &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2000}},
		},
	}
	s := newSuperroot(chains, preInteropReader())
	resp, err := (&superrootAPI{s: s}).AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	require.NotNil(t, resp.Data)
}

// ErrNotStarted routes to the optimistic-composition fallback.
func TestSuperroot_AtTimestamp_InteropNotStarted_ComposesFromOptimistic(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			optL2:     eth.BlockID{Number: 100, Hash: common.HexToHash("0x01")},
			optL1:     eth.BlockID{Number: 900},
			optOutput: &eth.OutputV0{StateRoot: eth.Bytes32{0x01}},
			status:    &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2000}},
		},
	}
	reader := &mockVerifiedReader{err: interop.ErrNotStarted}
	s := newSuperroot(chains, reader)
	resp, err := (&superrootAPI{s: s}).AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	require.NotNil(t, resp.Data)
}

// ------ Below-verified-db handoff tests ------

// Below firstVerifiable, the safe-head startup handoff guarantees the
// optimistic outputs are canonical, so Data is composed from them rather
// than returning an error.
func TestSuperroot_AtTimestamp_BelowVerifiedDB_ComposesFromOptimistic(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			optL2:     eth.BlockID{Number: 100, Hash: common.HexToHash("0x01")},
			optL1:     eth.BlockID{Number: 900},
			optOutput: &eth.OutputV0{StateRoot: eth.Bytes32{0x01}},
			status:    &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 2000}},
		},
	}
	reader := &mockVerifiedReader{err: interop.ErrBeforeVerifiedDB}
	s := newSuperroot(chains, reader)
	resp, err := (&superrootAPI{s: s}).AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	require.NotNil(t, resp.Data)
}

// assertErr returns a generic error instance used to signal mock failures.
func assertErr() error { return fmt.Errorf("mock error") }
