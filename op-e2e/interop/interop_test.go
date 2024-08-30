package interop

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/fakebeacon"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/opnode"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/setuputils"
	"github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	l2os "github.com/ethereum-optimism/optimism/op-proposer/proposer"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestInterop stands up a basic L1
// and multiple L2 states
func TestInterop(t *testing.T) {
	recipe := interopgen.InteropDevRecipe{
		L1ChainID:        900100,
		L2ChainIDs:       []uint64{900200, 900201},
		GenesisTimestamp: uint64(time.Now().Unix() + 3), // start chain 3 seconds from now
	}

	// Create a new HD wallet to store the keys
	hdWallet, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)

	// Build the world configuration from the recipe and the HD wallet
	worldCfg, err := recipe.Build(hdWallet)
	require.NoError(t, err)

	// create a logger for the world configuration
	logger := testlog.Logger(t, log.LevelInfo)
	require.NoError(t, worldCfg.Check(logger))

	// create the foundry artifacts and source map
	foundryArtifacts := foundry.OpenArtifactsDir("../../packages/contracts-bedrock/forge-artifacts")
	sourceMap := foundry.NewSourceMapFS(os.DirFS("../../packages/contracts-bedrock"))

	// deploy the world, using the logger, foundry artifacts, source map, and world configuration
	worldDeployment, worldOutput, err := interopgen.Deploy(logger, foundryArtifacts, sourceMap, worldCfg)
	require.NoError(t, err)

	// acknowledging worldDeployment to avoid unused variable warnings
	_ = worldDeployment

	// set up the L1 chain parameters
	genesisTimestampL1 := worldOutput.L1.Genesis.Timestamp
	require.Equal(t, recipe.GenesisTimestamp, genesisTimestampL1)
	blockTimeL1 := uint64(6)
	l1FinalizedDistance := uint64(3)
	l1Clock := clock.SystemClock

	// Create a fake Beacon node to hold on to blobs created by the L1 miner, and to serve them to L2
	blobPath := t.TempDir()
	bcn := fakebeacon.NewBeacon(logger.New("role", "l1_cl"),
		filepath.Join(blobPath, "l1_cl"), genesisTimestampL1, blockTimeL1)
	t.Cleanup(func() {
		_ = bcn.Close()
	})
	require.NoError(t, bcn.Start("127.0.0.1:0"))
	beaconApiAddr := bcn.BeaconAddr()
	require.NotEmpty(t, beaconApiAddr, "beacon API listener must be up")

	// Start the L1 chain
	l1Geth, err := geth.InitL1(blockTimeL1, l1FinalizedDistance, worldOutput.L1.Genesis, l1Clock,
		filepath.Join(blobPath, "l1_el"), bcn)
	require.NoError(t, err)
	require.NoError(t, l1Geth.Node.Start())
	t.Cleanup(func() {
		_ = l1Geth.Close()
	})

	// Start the L2 chains
	for id, l2Out := range worldOutput.L2s {
		logger := logger.New("role", "op-node-"+id)
		jwtPath := writeDefaultJWT(t)
		name := "l2-" + id
		l2Geth, err := geth.InitL2(name, l2Out.Genesis, jwtPath)
		require.NoError(t, err)
		require.NoError(t, l2Geth.Node.Start())
		t.Cleanup(func() {
			_ = l2Geth.Close()
		})
		// TODO register the op-geth node

		seqP2PSecret, err := hdWallet.Secret(devkeys.ChainOperatorKey{
			ChainID: l2Out.Genesis.Config.ChainID,
			Role:    devkeys.SequencerP2PRole,
		})
		require.NoError(t, err)

		nodeCfg := &node.Config{
			L1: &node.PreparedL1Endpoint{
				Client: client.NewBaseRPCClient(endpoint.DialRPC(
					endpoint.PreferAnyRPC,
					l1Geth.UserRPC(),
					mustDial(t, logger))),
				TrustRPC:        false,
				RPCProviderKind: sources.RPCKindDebugGeth,
			},
			L2: &node.L2EndpointConfig{
				// TODO refactoring this to a PreparedL2Endpoint,
				//  with auth, to utilize in-process RPC would be very nice.
				L2EngineAddr:      l2Geth.AuthRPC().RPC(),
				L2EngineJWTSecret: testingJWTSecret,
			},
			Beacon: &node.L1BeaconEndpointConfig{
				BeaconAddr: bcn.BeaconAddr(),
			},
			Driver: driver.Config{
				SequencerEnabled: true,
			},
			Rollup: *l2Out.RollupCfg,
			P2PSigner: &p2p.PreparedSigner{
				Signer: p2p.NewLocalSigner(seqP2PSecret)},
			RPC: node.RPCConfig{
				ListenAddr:  "127.0.0.1",
				ListenPort:  0,
				EnableAdmin: true,
			},
			P2P:                         nil, // disabled P2P setup for now
			L1EpochPollInterval:         time.Second * 2,
			RuntimeConfigReloadInterval: 0,
			Tracer:                      nil,
			Sync: sync.Config{
				SyncMode:                       sync.CLSync,
				SkipSyncStartCheck:             false,
				SupportsPostFinalizationELSync: false,
			},
			ConfigPersistence: node.DisabledConfigPersistence{},
		}
		opNode, err := opnode.NewOpnode(logger.New("service", "op-node"),
			nodeCfg, func(err error) {
				t.Error(err)
			})
		require.NoError(t, err)
		t.Cleanup(func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // force-quit
			_ = opNode.Stop(ctx)
		})
		// TODO register the op-node

		// op-proposer
		{
			proposerSecret, err := hdWallet.Secret(devkeys.ChainOperatorKey{
				ChainID: l2Out.Genesis.Config.ChainID,
				Role:    devkeys.ProposerRole,
			})
			require.NoError(t, err)
			proposerCLIConfig := &l2os.CLIConfig{
				L1EthRpc:          l1Geth.UserRPC().RPC(),
				RollupRpc:         opNode.UserRPC().RPC(),
				DGFAddress:        worldDeployment.L2s[id].DisputeGameFactoryProxy.Hex(),
				ProposalInterval:  6 * time.Second,
				DisputeGameType:   254, // Fast game type
				PollInterval:      500 * time.Millisecond,
				TxMgrConfig:       setuputils.NewTxMgrConfig(l1Geth.UserRPC(), proposerSecret),
				AllowNonFinalized: false,
				LogConfig: oplog.CLIConfig{
					Level:  log.LvlInfo,
					Format: oplog.FormatText,
				},
			}
			proposer, err := l2os.ProposerServiceFromCLIConfig(
				context.Background(), "0.0.1", proposerCLIConfig,
				logger.New("service", "proposer"))
			require.NoError(t, err, "must start proposer")
			require.NoError(t, proposer.Start(context.Background()))
			t.Cleanup(func() {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // force-quit
				_ = proposer.Stop(ctx)
			})
			// TODO register the proposer
		}

		// op-batcher
		{
			batcherSecret, err := hdWallet.Secret(devkeys.ChainOperatorKey{
				ChainID: l2Out.Genesis.Config.ChainID,
				Role:    devkeys.BatcherRole,
			})
			require.NoError(t, err)
			batcherCLIConfig := &bss.CLIConfig{
				L1EthRpc:                 l1Geth.UserRPC().RPC(),
				L2EthRpc:                 l2Geth.UserRPC().RPC(),
				RollupRpc:                opNode.UserRPC().RPC(),
				MaxPendingTransactions:   1,
				MaxChannelDuration:       1,
				MaxL1TxSize:              120_000,
				TestUseMaxTxSizeForBlobs: false,
				TargetNumFrames:          1,
				ApproxComprRatio:         0.4,
				SubSafetyMargin:          4,
				PollInterval:             50 * time.Millisecond,
				TxMgrConfig:              setuputils.NewTxMgrConfig(l1Geth.UserRPC(), batcherSecret),
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
			// Batch Submitter
			batcher, err := bss.BatcherServiceFromCLIConfig(
				context.Background(), "0.0.1", batcherCLIConfig,
				logger.New("service", "batcher"))
			require.NoError(t, err)
			require.NoError(t, err, "must start batcher")
			require.NoError(t, batcher.Start(context.Background()))
			t.Cleanup(func() {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // force-quit
				_ = batcher.Stop(ctx)
			})
			// TODO register the op-batcher
		}
	}

	// TODO op-supervisor

	// TODO (placeholder) Let the system test-run for a bit
	time.Sleep(time.Second * 30)
}

func mustDial(t *testing.T, logger log.Logger) func(v string) *rpc.Client {
	return func(v string) *rpc.Client {
		cl, err := dial.DialRPCClientWithTimeout(context.Background(), 30*time.Second, logger, v)
		require.NoError(t, err, "failed to dial")
		return cl
	}
}

var testingJWTSecret = [32]byte{123}

func writeDefaultJWT(t testing.TB) string {
	// Sadly the geth node config cannot load JWT secret from memory, it has to be a file
	jwtPath := path.Join(t.TempDir(), "jwt_secret")
	if err := os.WriteFile(jwtPath, []byte(hexutil.Encode(testingJWTSecret[:])), 0o600); err != nil {
		t.Fatalf("failed to prepare jwt file for geth: %v", err)
	}
	return jwtPath
}
