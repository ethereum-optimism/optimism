package sysgo

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/shim"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/setuputils"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

type L2Batcher struct {
	service *bss.BatcherService
	rpc     string
	l1RPC   string
	l2CLRPC string
	l2ELRPC string
}

func WithBatcher(batcherID stack.L2BatcherID, l1ELID stack.L1ELNodeID, l2CLID stack.L2CLNodeID, l2ELID stack.L2ELNodeID) stack.Option {
	return func(setup *stack.Setup) {
		orch := setup.Orchestrator.(*Orchestrator)
		setup.Require.False(orch.batchers.Has(batcherID), "batcher must not already exist")

		l2ID := setup.System.L2NetworkID(l2CLID.ChainID)
		l2Chain := setup.System.L2Network(l2ID).(stack.ExtensibleL2Network)

		l1ID := setup.System.L1NetworkID(l1ELID.ChainID)
		setup.Require.Equal(l2Chain.L1().ID(), l1ID, "expecting L1EL on L1 of L2CL")

		setup.Require.Equal(l2CLID.ChainID, l2ELID.ChainID, "L2 CL and EL must be on same L2 chain")

		l1EL, ok := orch.l1ELs.Get(l1ELID)
		setup.Require.True(ok)

		l2CL, ok := orch.l2CLs.Get(l2CLID)
		setup.Require.True(ok)

		l2EL, ok := orch.l2ELs.Get(l2ELID)
		setup.Require.True(ok)

		batcherSecret, err := orch.keys.Secret(devkeys.BatcherRole.Key(l2ELID.ChainID.ToBig()))
		setup.Require.NoError(err)

		logger := setup.Log.New("id", batcherID)
		logger.Info("Batcher key acquired", "addr", crypto.PubkeyToAddress(batcherSecret.PublicKey))

		batcherCLIConfig := &bss.CLIConfig{
			L1EthRpc:                 l1EL.userRPC,
			L2EthRpc:                 l2EL.userRPC,
			RollupRpc:                l2CL.rpc,
			MaxPendingTransactions:   1,
			MaxChannelDuration:       1,
			MaxL1TxSize:              120_000,
			TestUseMaxTxSizeForBlobs: false,
			TargetNumFrames:          1,
			ApproxComprRatio:         0.4,
			SubSafetyMargin:          4,
			PollInterval:             500 * time.Millisecond,
			TxMgrConfig:              setuputils.NewTxMgrConfig(endpoint.URL(l1EL.userRPC), batcherSecret),
			LogConfig: oplog.CLIConfig{
				Level:  log.LevelInfo,
				Format: oplog.FormatText,
			},
			Stopped:               false,
			BatchType:             derive.SpanBatchType,
			MaxBlocksPerSpanBatch: 10,
			DataAvailabilityType:  batcherFlags.CalldataType,
			CompressionAlgo:       derive.Brotli,
		}

		batcher, err := bss.BatcherServiceFromCLIConfig(
			setup.Ctx, "0.0.1", batcherCLIConfig,
			logger.New("service", "batcher"))
		setup.Require.NoError(err)
		setup.Require.NoError(batcher.Start(setup.Ctx))
		orch.t.Cleanup(func() {
			ctx, cancel := context.WithCancel(setup.Ctx)
			cancel() // force-quit
			logger.Info("Closing batcher")
			_ = batcher.Stop(ctx)
			logger.Info("Closed batcher")
		})

		b := &L2Batcher{
			service: batcher,
			rpc:     batcher.HTTPEndpoint(),
			l1RPC:   l1EL.userRPC,
			l2CLRPC: l2CL.rpc,
			l2ELRPC: l2EL.userRPC,
		}
		orch.batchers.Set(batcherID, b)

		rpcCl, err := client.NewRPC(setup.Ctx, setup.Log, b.rpc, client.WithLazyDial())
		setup.Require.NoError(err)

		bFrontend := shim.NewL2Batcher(shim.L2BatcherConfig{
			CommonConfig: shim.CommonConfigFromSetup(setup),
			ID:           batcherID,
			Client:       rpcCl,
		})
		l2Chain.AddL2Batcher(bFrontend)
	}
}
