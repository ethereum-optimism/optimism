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
	"github.com/ethereum/go-ethereum/ethclient"
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

	// Submit batch data for each chain in separate L1 blocks so tests can have one chain safe and one unsafe
	system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
		opts.SetChains(system.Actors.ChainA)
	})

	system.FinalizeL1()

	system.AddL2Block(system.Actors.ChainA)
	system.AddL2Block(system.Actors.ChainB)

	// Submit batch data for each chain in separate L1 blocks so tests can have one chain safe and one unsafe
	system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
		opts.SetChains(system.Actors.ChainA)
	})
	system.SubmitBatchData(func(opts *dsl.SubmitBatchDataOpts) {
		opts.SetChains(system.Actors.ChainB)
	})

	actors := system.Actors

	clients := map[string]*ethclient.Client{
		"chainA": actors.ChainA.SequencerEngine.EthClient(),
		"chainB": actors.ChainB.SequencerEngine.EthClient(),
	}
	expectedChainIDs := []eth.ChainID{actors.ChainA.ChainID, actors.ChainB.ChainID}
	superNode := dsl.NewSuperNode(t, testlog.Logger(t, log.LevelInfo), actors.L1Miner, actors.ChainA, actors.ChainB)

	gt.Run("SuppliedTimestamp", func(gt *testing.T) {
		t := helpers.NewDefaultTesting(gt)
		initialResp := waitForSuperRootAtTimestamp(t, superNode, latestFinalizedLookupTimestamp, func(collect *assert.CollectT, resp eth.SuperRootAtTimestampResponse) {
			require.NotZero(collect, resp.CurrentFinalizedTimestamp)
			require.Nil(collect, resp.Data)
		})
		targetTimestamp := initialResp.CurrentFinalizedTimestamp
		expected := waitForSuperRootAtTimestamp(t, superNode, targetTimestamp, func(collect *assert.CollectT, resp eth.SuperRootAtTimestampResponse) {
			require.Equal(collect, targetTimestamp, resp.CurrentFinalizedTimestamp)
			require.NotNil(collect, resp.Data)
			for _, chainID := range expectedChainIDs {
				require.Contains(collect, resp.ChainIDs, chainID)
			}
		})
		require.NotNil(t, expected.Data)

		migrator, err := script.NewSuperRootMigratorWithClients(testlog.Logger(t, log.LevelInfo), clients, &targetTimestamp)
		require.NoError(t, err)
		actual, err := migrator.Run(t.Ctx())
		require.NoError(t, err)
		require.Equal(t, common.Hash(expected.Data.SuperRoot), actual)
	})

	gt.Run("LatestFinalized", func(gt *testing.T) {
		t := helpers.NewDefaultTesting(gt)
		initialResp := waitForSuperRootAtTimestamp(t, superNode, latestFinalizedLookupTimestamp, func(collect *assert.CollectT, resp eth.SuperRootAtTimestampResponse) {
			require.NotZero(collect, resp.CurrentFinalizedTimestamp)
			require.Nil(collect, resp.Data)
		})
		finalizedTimestamp := initialResp.CurrentFinalizedTimestamp
		expected := waitForSuperRootAtTimestamp(t, superNode, finalizedTimestamp, func(collect *assert.CollectT, resp eth.SuperRootAtTimestampResponse) {
			require.Equal(collect, finalizedTimestamp, resp.CurrentFinalizedTimestamp)
			require.NotNil(collect, resp.Data)
			for _, chainID := range expectedChainIDs {
				require.Contains(collect, resp.ChainIDs, chainID)
			}
		})
		require.NotNil(t, expected.Data)

		migrator, err := script.NewSuperRootMigratorWithClients(testlog.Logger(t, log.LevelInfo), clients, nil)
		require.NoError(t, err)
		actual, err := migrator.Run(t.Ctx())
		require.NoError(t, err)
		require.NotNil(t, migrator.TargetTimestamp)
		require.Equal(t, finalizedTimestamp, *migrator.TargetTimestamp)
		require.Equal(t, common.Hash(expected.Data.SuperRoot), actual)
	})
}

func waitForSuperRootAtTimestamp(
	t helpers.Testing,
	superNode apis.SupernodeQueryAPI,
	timestamp uint64,
	check func(*assert.CollectT, eth.SuperRootAtTimestampResponse),
) eth.SuperRootAtTimestampResponse {
	var resp eth.SuperRootAtTimestampResponse
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		var err error
		resp, err = superNode.SuperRootAtTimestamp(t.Ctx(), timestamp)
		require.NoError(collect, err)
		check(collect, resp)
	}, 10*time.Second, 100*time.Millisecond)

	return resp
}
