package sysgo

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/setuputils"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

type L2Batcher struct {
	id      stack.L2BatcherID
	service *bss.BatcherService
	rpc     string
	l1RPC   string
	l2CLRPC string
	l2ELRPC string
}

func (b *L2Batcher) hydrate(system stack.ExtensibleSystem) {
	require := system.T().Require()
	rpcCl, err := client.NewRPC(system.T().Ctx(), system.Logger(), b.rpc, client.WithLazyDial())
	require.NoError(err)
	system.T().Cleanup(rpcCl.Close)

	bFrontend := shim.NewL2Batcher(shim.L2BatcherConfig{
		CommonConfig: shim.NewCommonConfig(system.T()),
		ID:           b.id,
		Client:       rpcCl,
	})
	l2Net := system.L2Network(stack.L2NetworkID(b.id.ChainID()))
	l2Net.(stack.ExtensibleL2Network).AddL2Batcher(bFrontend)
}

type BatcherOption func(id stack.L2BatcherID, cfg *bss.CLIConfig)

func WithBatcherOption(opt BatcherOption) stack.Option[*Orchestrator] {
	return stack.Deploy[*Orchestrator](func(orch *Orchestrator) {
		orch.batcherOptions = append(orch.batcherOptions, opt)
	})
}

func WithBatcher(batcherID stack.L2BatcherID, l1ELID stack.L1ELNodeID, l2CLID stack.L2CLNodeID, l2ELID stack.L2ELNodeID) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), batcherID))

		require := p.Require()
		batcherCID := stack.ConvertL2BatcherID(batcherID).ComponentID
		require.False(orch.registry.Has(batcherCID), "batcher must not already exist")

		l2NetComponent, ok := orch.registry.Get(stack.ConvertL2NetworkID(stack.L2NetworkID(l2CLID.ChainID())).ComponentID)
		require.True(ok)
		l2Net := l2NetComponent.(*L2Network)

		l1NetComponent, ok := orch.registry.Get(stack.ConvertL1NetworkID(stack.L1NetworkID(l1ELID.ChainID())).ComponentID)
		require.True(ok)
		l1Net := l1NetComponent.(*L1Network)

		require.Equal(l2Net.l1ChainID, l1Net.id.ChainID(), "expecting L1EL on L1 of L2CL")

		require.Equal(l2CLID.ChainID(), l2ELID.ChainID(), "L2 CL and EL must be on same L2 chain")

		l1ELComponent, ok := orch.registry.Get(stack.ConvertL1ELNodeID(l1ELID).ComponentID)
		require.True(ok)
		l1EL := l1ELComponent.(L1ELNode)

		l2CLComponent, ok := orch.registry.Get(stack.ConvertL2CLNodeID(l2CLID).ComponentID)
		require.True(ok)
		l2CL := l2CLComponent.(L2CLNode)

		l2ELComponent, ok := orch.registry.Get(stack.ConvertL2ELNodeID(l2ELID).ComponentID)
		require.True(ok)
		l2EL := l2ELComponent.(L2ELNode)

		batcherSecret, err := orch.keys.Secret(devkeys.BatcherRole.Key(l2ELID.ChainID().ToBig()))
		require.NoError(err)

		logger := p.Logger()
		logger.SetContext(p.Ctx())
		logger.Info("Batcher key acquired", "addr", crypto.PubkeyToAddress(batcherSecret.PublicKey))

		batcherCLIConfig := &bss.CLIConfig{
			L1EthRpc:                 l1EL.UserRPC(),
			L2EthRpc:                 []string{l2EL.UserRPC()},
			RollupRpc:                []string{l2CL.UserRPC()},
			MaxPendingTransactions:   7,
			MaxChannelDuration:       1,
			MaxL1TxSize:              120_000,
			TestUseMaxTxSizeForBlobs: false,
			TargetNumFrames:          1,
			ApproxComprRatio:         0.4,
			SubSafetyMargin:          4,
			PollInterval:             500 * time.Millisecond,
			TxMgrConfig:              setuputils.NewTxMgrConfig(endpoint.URL(l1EL.UserRPC()), batcherSecret),
			LogConfig: oplog.CLIConfig{
				Level:  log.LevelInfo,
				Format: oplog.FormatText,
			},
			Stopped:               false,
			BatchType:             derive.SpanBatchType,
			MaxBlocksPerSpanBatch: 10,
			DataAvailabilityType:  batcherFlags.CalldataType,
			CompressionAlgo:       derive.Brotli,
			RPC: oprpc.CLIConfig{
				EnableAdmin: true,
			},
		}
		for _, opt := range orch.batcherOptions {
			opt(batcherID, batcherCLIConfig)
		}

		batcherContext, cancelBatcherCtx := context.WithCancel(p.Ctx())
		var closeAppFn context.CancelCauseFunc = func(cause error) {
			p.Errorf("closeAppFn called, batcher hit a critical error: %v", cause)
			cancelBatcherCtx()
		}
		batcher, err := bss.BatcherServiceFromCLIConfig(
			batcherContext, closeAppFn, "0.0.1", batcherCLIConfig,
			logger)
		require.NoError(err)
		require.NoError(batcher.Start(p.Ctx()))
		p.Cleanup(func() {
			ctx, cancel := context.WithCancel(p.Ctx())
			cancel() // force-quit
			logger.Info("Closing batcher")
			_ = batcher.Stop(ctx)
			logger.Info("Closed batcher")
		})

		b := &L2Batcher{
			id:      batcherID,
			service: batcher,
			rpc:     batcher.HTTPEndpoint(),
			l1RPC:   l1EL.UserRPC(),
			l2CLRPC: l2CL.UserRPC(),
			l2ELRPC: l2EL.UserRPC(),
		}
		orch.registry.Register(stack.ConvertL2BatcherID(batcherID).ComponentID, b)
	})
}
