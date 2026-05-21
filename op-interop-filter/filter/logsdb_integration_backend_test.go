package filter

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

func TestIntegration_Backend_NoChains_FailsafeOn(t *testing.T) {
	t.Parallel()

	mtr := newCapturingMetrics()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	logger := testlog.Logger(t, log.LevelError)
	bk := NewBackend(ctx, BackendParams{
		Logger:         logger,
		Metrics:        mtr,
		Chains:         map[eth.ChainID]ChainIngester{},
		CrossValidator: NewLockstepCrossValidator(ctx, logger, mtr, 1<<30, defaultStartTs, 100, map[eth.ChainID]ChainIngester{}),
	})

	require.False(t, bk.Ready(), "empty chain map -> Backend.Ready() is false")
	err := bk.CheckAccessList(ctx, nil, safety.LocalUnsafe, messages.ExecutingDescriptor{ChainID: executingChain()})
	require.ErrorIs(t, err, types.ErrUninitialized)
}

func TestIntegration_Backend_ManualFailsafe_RejectsAll(t *testing.T) {
	t.Parallel()

	bk := twoChainBackend(t, 1)
	bk.SetFailsafeEnabled(true)
	bk.requireRejection(executingChain(), inclusionTs, "failsafe", bk.sourceAccess(100, 0))
}

func TestIntegration_Backend_IngesterErrorTripsFailsafe(t *testing.T) {
	t.Parallel()

	bk := twoChainBackend(t, 1)
	require.False(t, bk.FailsafeEnabled())

	bk.ingesters[eth.ChainIDFromUInt64(901)].SetError(ErrorConflict, "forced")
	require.True(t, bk.FailsafeEnabled())
	bk.requireRejection(executingChain(), inclusionTs, "failsafe", bk.sourceAccess(100, 0))
}

func TestIntegration_Backend_RecoverReorg_ClearsFailsafe(t *testing.T) {
	t.Parallel()

	bk, si := reorgRecoveryBackend(t)
	si.eth.SetLabelBlock(eth.Finalized, si.blockInfo[101])

	putIntoReorg(t, si, 103, 1206)
	require.True(t, bk.FailsafeEnabled())

	bk.tryResolveReorgs(context.Background())
	require.False(t, bk.FailsafeEnabled())
}

func TestIntegration_Backend_RecoverConflict_DoesNotClearFailsafe(t *testing.T) {
	t.Parallel()

	bk, si := reorgRecoveryBackend(t)
	si.eth.SetLabelBlock(eth.Finalized, si.blockInfo[101])

	si.SetError(ErrorConflict, "forced")
	require.True(t, bk.FailsafeEnabled())

	bk.tryResolveReorgs(context.Background())
	require.True(t, bk.FailsafeEnabled(),
		"ErrorConflict is not auto-recoverable; failsafe must stay on")
}

func TestIntegration_Backend_UnsupportedSafetyLevel_Rejected(t *testing.T) {
	t.Parallel()

	bk := twoChainBackend(t, 1)
	err := bk.CheckAccessList(context.Background(), nil, safety.Finalized,
		messages.ExecutingDescriptor{ChainID: executingChain(), Timestamp: inclusionTs})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported safety level")
}

func TestIntegration_Backend_EmptyAccessList_LocalUnsafe_Accepted(t *testing.T) {
	t.Parallel()

	bk := twoChainBackend(t, 1)
	require.NoError(t, bk.CheckAccessList(context.Background(), nil, safety.LocalUnsafe,
		messages.ExecutingDescriptor{ChainID: executingChain(), Timestamp: inclusionTs}))
}

func TestIntegration_Backend_Ready_FalseUntilAllChainsReady(t *testing.T) {
	t.Parallel()

	bk := newSeededBackend(t, backendOpts{
		Specs: []seedSpec{
			{ChainID: 901, AnchorNumber: 99, AnchorTime: 1198,
				Blocks: []seedBlock{{Num: 100, Ts: 1200, Logs: []seedLog{{}}}}},
			{ChainID: 902, AnchorNumber: 99, AnchorTime: 1198,
				StartTimestamp: 1 << 30, // unreachable by seeded timestamps -> Ready=false
				NoIngest:       true},
		},
	})

	require.False(t, bk.Ready(), "Backend.Ready requires all ingesters Ready")

	err := bk.CheckAccessList(context.Background(), nil, safety.LocalUnsafe,
		messages.ExecutingDescriptor{ChainID: executingChain(), Timestamp: inclusionTs})
	require.Error(t, err)
	require.True(t, errors.Is(err, types.ErrUninitialized) || errors.Is(err, types.ErrFailsafeEnabled),
		"expected ErrUninitialized or ErrFailsafeEnabled, got %v", err)
}
