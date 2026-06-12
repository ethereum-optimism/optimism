package interop

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// staleL1Checker reports every head as non-canonical, simulating an offline L1
// reorg: the recorded L1Inclusion no longer matches canonical L1.
type staleL1Checker struct{}

func (staleL1Checker) SameL1Chain(context.Context, []eth.BlockID) (bool, error) {
	return false, nil
}

// resumeTestState commits a verified entry at ts and returns the chainID and
// the committed head. The head's hash follows the mock convention
// (BigToHash(number)) so the mock's OutputV0AtBlockNumber reports it as
// canonical unless a test overrides it.
func resumeTestState(t *testing.T, h *interopTestHarness, ts uint64) (eth.ChainID, eth.BlockID) {
	t.Helper()
	chainID := eth.ChainIDFromUInt64(10)
	head := eth.BlockID{Number: ts, Hash: common.BigToHash(new(big.Int).SetUint64(ts))}
	require.NoError(t, h.interop.verifiedDB.Commit(VerifiedResult{
		Timestamp:   ts,
		L1Inclusion: eth.BlockID{Number: 5, Hash: common.HexToHash("0x11")},
		L2Heads:     map[eth.ChainID]eth.BlockID{chainID: head},
	}))
	return chainID, head
}

func TestCheckResumeConsistency_PassesWhenConsistent(t *testing.T) {
	h := newInteropTestHarness(t).WithChain(10, nil).Build()
	chainID, head := resumeTestState(t, h, 1001)
	require.NoError(t, h.interop.logsDBs[chainID].SealBlock(common.Hash{}, head, 1001))

	sleep, err := h.interop.checkResumeConsistency()
	require.NoError(t, err)
	require.Zero(t, sleep)
	require.True(t, h.interop.resumeChecked)
}

func TestCheckResumeConsistency_HaltsOnEmptyLogsDB(t *testing.T) {
	h := newInteropTestHarness(t).WithChain(10, nil).Build()
	resumeTestState(t, h, 1001) // verified history exists, logsDB left empty

	_, err := h.interop.checkResumeConsistency()
	require.ErrorIs(t, err, ErrResumeDivergence)
	require.False(t, h.interop.resumeChecked)
	requireHalted(t, h)
}

func TestCheckResumeConsistency_HaltsOnTipMismatch(t *testing.T) {
	h := newInteropTestHarness(t).WithChain(10, nil).Build()
	chainID, head := resumeTestState(t, h, 1001)
	// Seal a tip that is not the verified head: local stores disagree.
	require.NoError(t, h.interop.logsDBs[chainID].SealBlock(
		common.Hash{}, eth.BlockID{Number: head.Number, Hash: common.HexToHash("0xeeee")}, 1001))

	_, err := h.interop.checkResumeConsistency()
	require.ErrorIs(t, err, ErrResumeDivergence)
	requireHalted(t, h)
}

func TestCheckResumeConsistency_DeferredWhenPendingTransition(t *testing.T) {
	h := newInteropTestHarness(t).WithChain(10, nil).Build()
	resumeTestState(t, h, 1001)
	// Mid-transition state: the strict tip==head equality legitimately does
	// not hold; convergence belongs to the WAL replay path.
	require.NoError(t, h.interop.verifiedDB.SetPendingTransition(PendingTransition{
		Decision: DecisionAdvance,
		Result:   &Result{Timestamp: 1002},
	}))

	sleep, err := h.interop.checkResumeConsistency()
	require.NoError(t, err)
	require.Zero(t, sleep)
	require.True(t, h.interop.resumeChecked)
}

// Divergence under a still-canonical L1 is the determinism-violation signature
// (no protocol reorg can change a verified block while its L1Inclusion stays
// canonical) and must halt before the first round can act on divergent data.
func TestCheckResumeConsistency_HaltsOnChainDivergenceWithCanonicalL1(t *testing.T) {
	h := newInteropTestHarness(t).WithChain(10, nil).Build()
	chainID := eth.ChainIDFromUInt64(10)
	// Verified head whose hash differs from what the chain now reports
	// canonical at that height (mock reports BigToHash(number)).
	head := eth.BlockID{Number: 1001, Hash: common.HexToHash("0xffff")}
	require.NoError(t, h.interop.verifiedDB.Commit(VerifiedResult{
		Timestamp:   1001,
		L1Inclusion: eth.BlockID{Number: 5, Hash: common.HexToHash("0x11")},
		L2Heads:     map[eth.ChainID]eth.BlockID{chainID: head},
	}))
	// Local stores agree with each other — only the chain moved.
	require.NoError(t, h.interop.logsDBs[chainID].SealBlock(common.Hash{}, head, 1001))
	// noopL1Checker (set by the harness) reports L1Inclusion canonical.

	_, err := h.interop.checkResumeConsistency()
	require.ErrorIs(t, err, ErrResumeDivergence)
	requireHalted(t, h)
}

// The same chain divergence under a reorged L1 is an offline protocol reorg:
// the round loop's L1Inclusion check will fire and rewind, so the resume
// check must let it through.
func TestCheckResumeConsistency_ProceedsOnOfflineL1Reorg(t *testing.T) {
	h := newInteropTestHarness(t).WithChain(10, nil).Build()
	chainID := eth.ChainIDFromUInt64(10)
	head := eth.BlockID{Number: 1001, Hash: common.HexToHash("0xffff")}
	require.NoError(t, h.interop.verifiedDB.Commit(VerifiedResult{
		Timestamp:   1001,
		L1Inclusion: eth.BlockID{Number: 5, Hash: common.HexToHash("0x11")},
		L2Heads:     map[eth.ChainID]eth.BlockID{chainID: head},
	}))
	require.NoError(t, h.interop.logsDBs[chainID].SealBlock(common.Hash{}, head, 1001))
	h.interop.l1Checker = staleL1Checker{}

	sleep, err := h.interop.checkResumeConsistency()
	require.NoError(t, err)
	require.Zero(t, sleep)
	require.True(t, h.interop.resumeChecked)
}

func TestCheckResumeConsistency_RetriesWhileChainNotReady(t *testing.T) {
	h := newInteropTestHarness(t).WithChain(10, func(m *mockChainContainer) {
		m.outputV0Override = func(ctx context.Context, l2BlockNum uint64) (*eth.OutputV0, error) {
			return nil, errors.New("EL not up yet")
		}
	}).Build()
	chainID, head := resumeTestState(t, h, 1001)
	require.NoError(t, h.interop.logsDBs[chainID].SealBlock(common.Hash{}, head, 1001))

	sleep, err := h.interop.checkResumeConsistency()
	require.NoError(t, err, "chain startup races must retry, not fail")
	require.Equal(t, backoffPeriod, sleep)
	require.False(t, h.interop.resumeChecked)
}

// TestRunLoop_ResumeHaltsOnDivergence drives the full Start path: a resume
// with verified history but divergent local state must halt Start with the
// resume sentinel before any verification round runs.
func TestRunLoop_ResumeHaltsOnDivergence(t *testing.T) {
	dataDir := t.TempDir()
	db, err := OpenVerifiedDB(dataDir)
	require.NoError(t, err)
	require.NoError(t, db.Commit(VerifiedResult{
		Timestamp:   500,
		L1Inclusion: eth.BlockID{Number: 1},
		L2Heads: map[eth.ChainID]eth.BlockID{
			eth.ChainIDFromUInt64(10): {Number: 500, Hash: common.BigToHash(big.NewInt(500))},
		},
	}))
	require.NoError(t, db.Close())

	h := newInteropTestHarness(t).WithDataDir(dataDir).WithChain(10, nil).Build()
	// logsDB deliberately left empty: verified history with no sealed blocks.

	done := make(chan error, 1)
	go func() { done <- h.interop.Start(context.Background()) }()

	select {
	case err = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not halt within 5s on resume divergence")
	}
	require.ErrorIs(t, err, ErrResumeDivergence)
}

// TestRunLoop_ResumePassesCheckThenRuns proves the loop traverses the resume
// check and reaches the verification rounds: with consistent local state, the
// first round's ErrHistoryUnavailable is what halts Start — not the check.
func TestRunLoop_ResumePassesCheckThenRuns(t *testing.T) {
	dataDir := t.TempDir()
	head := eth.BlockID{Number: 500, Hash: common.BigToHash(big.NewInt(500))}
	db, err := OpenVerifiedDB(dataDir)
	require.NoError(t, err)
	require.NoError(t, db.Commit(VerifiedResult{
		Timestamp:   500,
		L1Inclusion: eth.BlockID{Number: 1},
		L2Heads:     map[eth.ChainID]eth.BlockID{eth.ChainIDFromUInt64(10): head},
	}))
	require.NoError(t, db.Close())

	h := newInteropTestHarness(t).WithDataDir(dataDir).WithChain(10, func(m *mockChainContainer) {
		m.optimisticAtErr = cc.ErrHistoryUnavailable
	}).Build()
	require.NoError(t, h.interop.logsDBs[eth.ChainIDFromUInt64(10)].SealBlock(common.Hash{}, head, 500))

	done := make(chan error, 1)
	go func() { done <- h.interop.Start(context.Background()) }()

	select {
	case err = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not halt within 5s")
	}
	require.NotErrorIs(t, err, ErrResumeDivergence, "resume check must pass on consistent state")
	require.ErrorIs(t, err, cc.ErrHistoryUnavailable, "first verification round must be reached")
}
