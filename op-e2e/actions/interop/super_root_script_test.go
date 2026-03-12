package interop

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/interop/dsl"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const latestFinalizedLookupTimestamp = ^uint64(0)

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
	expectedChainIDs := []eth.ChainID{actors.ChainA.ChainID, actors.ChainB.ChainID}
	superNode := dsl.NewSuperNode(t, testlog.Logger(t, log.LevelInfo), actors.L1Miner, actors.ChainA, actors.ChainB)

	gt.Run("SuppliedTimestamp", func(gt *testing.T) {
		t := helpers.NewDefaultTesting(gt)
		safeTime := actors.ChainA.Sequencer.L2Safe().Time
		if otherSafeTime := actors.ChainB.Sequencer.L2Safe().Time; otherSafeTime < safeTime {
			safeTime = otherSafeTime
		}
		migrator, err := script.NewSuperRootMigratorWithClient(testlog.Logger(t, log.LevelInfo), superNode, &safeTime)
		require.NoError(t, err)

		_, err = migrator.Run(t.Ctx())
		require.Error(t, err)
	})

	gt.Run("LatestFinalized", func(gt *testing.T) {
		t := helpers.NewDefaultTesting(gt)
		initialResp := waitForSuperRootAtTimestamp(t, superNode, latestFinalizedLookupTimestamp, func(collect *assert.CollectT, resp eth.SuperRootAtTimestampResponse) {
			require.NotZero(collect, resp.CurrentFinalizedTimestamp)
			require.Nil(collect, resp.Data)
		})
		finalizedTime := initialResp.CurrentFinalizedTimestamp
		superNodeResp := waitForSuperRootAtTimestamp(t, superNode, finalizedTime, func(collect *assert.CollectT, resp eth.SuperRootAtTimestampResponse) {
			require.Equal(collect, finalizedTime, resp.CurrentFinalizedTimestamp)
			require.NotNil(collect, resp.Data)
			for _, chainID := range expectedChainIDs {
				require.Contains(collect, resp.ChainIDs, chainID)
			}
		})
		require.Equal(t, finalizedTime, superNodeResp.CurrentFinalizedTimestamp)
		require.NotNil(t, superNodeResp.Data)
		supervisorResp, err := actors.Supervisor.SuperRootAtTimestamp(t.Ctx(), hexutil.Uint64(finalizedTime))
		require.NoError(t, err)
		require.Equal(t, common.Hash(supervisorResp.SuperRoot), common.Hash(superNodeResp.Data.SuperRoot))

		migrator, err := script.NewSuperRootMigratorWithClient(testlog.Logger(t, log.LevelInfo), superNode, nil)
		require.NoError(t, err)

		actual, err := migrator.Run(t.Ctx())
		require.NoError(t, err)
		require.Equal(t, common.Hash(supervisorResp.SuperRoot), actual)
	})
}

func waitForSuperRootAtTimestamp(
	t helpers.Testing,
	superNode apis.SupernodeQueryAPI,
	timestamp uint64,
	check func(*assert.CollectT, eth.SuperRootAtTimestampResponse),
) eth.SuperRootAtTimestampResponse {
	t.Helper()

	var resp eth.SuperRootAtTimestampResponse
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		var err error
		resp, err = superNode.SuperRootAtTimestamp(t.Ctx(), timestamp)
		require.NoError(collect, err)
		check(collect, resp)
	}, 10*time.Second, 100*time.Millisecond)

	return resp
}
