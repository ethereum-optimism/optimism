package sysgo

import (
	"context"
	"sync"
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
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

type L2Batcher struct {
	id          stack.L2BatcherID
	service     *bss.BatcherService
	rpc         string
	l1RPC       string
	l2CLRPC     string
	l2ELRPC     string
	testControl *batcherTestControl
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

// proxyingEndpointProvider wraps an L2EndpointProvider and injects a rollup client proxy.
type proxyingEndpointProvider struct {
	inner      dial.L2EndpointProvider
	proxy      *rollupClientProxy
	realClient dial.RollupClientInterface
	clientOnce sync.Once
	clientErr  error
}

func (p *proxyingEndpointProvider) RollupClient(ctx context.Context) (dial.RollupClientInterface, error) {
	// Create the real clients once and wrap them with the proxy
	p.clientOnce.Do(func() {
		p.realClient, p.clientErr = p.inner.RollupClient(ctx)
		if p.clientErr != nil {
			return
		}
		// Update the proxy's inner client to be the real rollup client
		p.proxy.inner = p.realClient

		// Also fetch the L2 eth client for the proxy
		l2EthClient, err := p.inner.EthClient(ctx)
		if err != nil {
			p.clientErr = err
			return
		}
		p.proxy.l2Client = l2EthClient
	})
	if p.clientErr != nil {
		return nil, p.clientErr
	}
	return p.proxy, nil
}

func (p *proxyingEndpointProvider) EthClient(ctx context.Context) (dial.EthClientInterface, error) {
	return p.inner.EthClient(ctx)
}

func (p *proxyingEndpointProvider) Close() {
	if p.inner != nil {
		p.inner.Close()
	}
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
		require.False(orch.batchers.Has(batcherID), "batcher must not already exist")

		l2Net, ok := orch.l2Nets.Get(l2CLID.ChainID())
		require.True(ok)

		l1Net, ok := orch.l1Nets.Get(l1ELID.ChainID())
		require.True(ok)

		require.Equal(l2Net.l1ChainID, l1Net.id.ChainID(), "expecting L1EL on L1 of L2CL")

		require.Equal(l2CLID.ChainID(), l2ELID.ChainID(), "L2 CL and EL must be on same L2 chain")

		l1EL, ok := orch.l1ELs.Get(l1ELID)
		require.True(ok)

		l2CL, ok := orch.l2CLs.Get(l2CLID)
		require.True(ok)

		l2EL, ok := orch.l2ELs.Get(l2ELID)
		require.True(ok)

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

		// Variables to store proxy and test control
		var rollupProxy *rollupClientProxy
		var testControl *batcherTestControl

		// Driver setup option that creates and injects the proxy
		withProxyOption := func(setup *bss.DriverSetup) {
			// Create rollup client proxy for test control
			// The inner client and l2Client will be set lazily when RollupClient() is called
			rollupProxy = &rollupClientProxy{}

			// Create proxying endpoint provider that wraps the original
			proxyingProvider := &proxyingEndpointProvider{
				inner: setup.EndpointProvider,
				proxy: rollupProxy,
			}

			// Replace the endpoint provider with our proxying version
			setup.EndpointProvider = proxyingProvider

			// Create test control
			testControl = &batcherTestControl{
				proxy: rollupProxy,
				log:   logger,
			}
		}

		batcher, err := bss.BatcherServiceFromCLIConfig(
			batcherContext, closeAppFn, "0.0.1", batcherCLIConfig,
			logger, withProxyOption)
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
			id:          batcherID,
			service:     batcher,
			rpc:         batcher.HTTPEndpoint(),
			l1RPC:       l1EL.UserRPC(),
			l2CLRPC:     l2CL.UserRPC(),
			l2ELRPC:     l2EL.UserRPC(),
			testControl: testControl,
		}
		orch.batchers.Set(batcherID, b)
	})
}

// batcherTestControl provides test control over the batcher by manipulating
// the rollup client proxy to control what blocks the batcher sees.
type batcherTestControl struct {
	proxy *rollupClientProxy
	log   log.Logger
}

// PauseAtBlock pauses the batcher at the specified block number.
// The batcher will process up to and including blockNum, but won't see
// any blocks beyond it. Returns the highest block number the batcher will see.
func (tc *batcherTestControl) PauseAtBlock(blockNum uint64) uint64 {
	tc.log.Info("pausing batcher at block", "blockNum", blockNum)

	// Set the pause in the proxy
	tc.proxy.setPauseAtBlock(blockNum)

	// The batcher will naturally stop processing when it queries sync status
	// and sees unsafe head capped at blockNum (inclusive)

	return blockNum
}

// Unpause resumes normal batcher operation, allowing it to see all available blocks.
func (tc *batcherTestControl) Unpause() {
	tc.log.Info("unpausing batcher")
	tc.proxy.clearPause()
}
