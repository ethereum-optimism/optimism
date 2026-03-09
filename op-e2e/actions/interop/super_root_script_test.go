package interop

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/interop/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type supervisorSupernodeAdapter struct {
	supervisor *dsl.SupervisorActor
}

func (a *supervisorSupernodeAdapter) SuperRootAtTimestamp(ctx context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error) {
	syncStatus, err := a.supervisor.SyncStatus(ctx)
	if err != nil {
		return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("failed to fetch supervisor sync status: %w", err)
	}

	resp := eth.SuperRootAtTimestampResponse{
		CurrentL1:                 syncStatus.MinSyncedL1.ID(),
		CurrentSafeTimestamp:      syncStatus.SafeTimestamp,
		CurrentFinalizedTimestamp: syncStatus.FinalizedTimestamp,
	}
	if timestamp == 0 {
		return resp, nil
	}

	superRootResp, err := a.supervisor.SuperRootAtTimestamp(ctx, hexutil.Uint64(timestamp))
	if err != nil {
		return eth.SuperRootAtTimestampResponse{}, err
	}
	super, err := superRootResp.ToSuper()
	if err != nil {
		return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("failed to convert supervisor super root response: %w", err)
	}

	chainIDs := make([]eth.ChainID, 0, len(superRootResp.Chains))
	for _, chain := range superRootResp.Chains {
		chainIDs = append(chainIDs, chain.ChainID)
	}
	resp.ChainIDs = chainIDs
	resp.Data = &eth.SuperRootResponseData{
		VerifiedRequiredL1: superRootResp.CrossSafeDerivedFrom,
		Super:              super,
		SuperRoot:          superRootResp.SuperRoot,
	}
	return resp, nil
}

func TestSuperRootScript(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	system := dsl.NewInteropDSL(t)

	system.AddL2Block(system.Actors.ChainA)
	system.AddL2Block(system.Actors.ChainB)

	system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
		opts.SetChains(system.Actors.ChainA)
	})

	system.FinalizeL1()

	system.AddL2Block(system.Actors.ChainA)
	system.AddL2Block(system.Actors.ChainB)

	system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
		opts.SetChains(system.Actors.ChainA)
	})
	system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
		opts.SetChains(system.Actors.ChainB)
	})

	actors := system.Actors
	client := &supervisorSupernodeAdapter{supervisor: actors.Supervisor}

	gt.Run("SuppliedTimestamp", func(gt *testing.T) {
		t := helpers.NewDefaultTesting(gt)
		safeTime := actors.ChainA.Sequencer.L2Safe().Time
		expected, err := actors.Supervisor.SuperRootAtTimestamp(t.Ctx(), hexutil.Uint64(safeTime))
		require.NoError(t, err)

		migrator, err := script.NewSuperRootMigratorWithClient(testlog.Logger(t, log.LevelInfo), client, &safeTime)
		require.NoError(t, err)

		actual, err := migrator.Run(t.Ctx())
		require.NoError(t, err)
		require.Equal(t, common.Hash(expected.SuperRoot), actual)
	})

	gt.Run("LatestFinalized", func(gt *testing.T) {
		t := helpers.NewDefaultTesting(gt)

		syncStatus, err := actors.Supervisor.SyncStatus(t.Ctx())
		require.NoError(t, err)
		finalizedTime := syncStatus.FinalizedTimestamp
		expected, err := actors.Supervisor.SuperRootAtTimestamp(t.Ctx(), hexutil.Uint64(finalizedTime))
		require.NoError(t, err)

		migrator, err := script.NewSuperRootMigratorWithClient(testlog.Logger(t, log.LevelInfo), client, nil)
		require.NoError(t, err)

		actual, err := migrator.Run(t.Ctx())
		require.NoError(t, err)
		require.Equal(t, common.Hash(expected.SuperRoot), actual)
	})
}
