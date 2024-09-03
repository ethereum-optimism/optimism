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

// rather than pass around a string for rpc, pass around an rpc interface
// this interface should be able to have a simple version
// and a copmlex version
// a utility in the middle can decide which to use based on an interface check,
// plus preferences if they can be applied
type rpcClient interface {
}

// ignore unused variable warning
var _ rpcClient = &rpc.Client{}

// example: ask for a chainID plus the actor,
// get the actor back
// system2 should provide a complex rpcClient
// or could return the simple rpcClient url

// TestInterop stands up a basic L1
// and multiple L2 states

// system2 is a struct for creating resources
// rather than make a big-bang of resources like system 1,
// system 2 will create resources in a more modular way
// and allows calls
type system2 struct {
	recipe          *interopgen.InteropDevRecipe
	logger          log.Logger
	hdWallet        *devkeys.MnemonicDevKeys
	worldDeployment *interopgen.WorldDeployment
	worldOutput     *interopgen.WorldOutput
	beacon          *fakebeacon.FakeBeacon
	l1              *geth.GethInstance
	l2s             map[string]l2Set
}

type l2Set struct {
	opNode   *opnode.Opnode
	l2Geth   *geth.GethInstance
	proposer *l2os.ProposerService
	batcher  *bss.BatcherService
}

// prepareHDWallet creates a new HD wallet to store the keys
// and stores it in the system
func (s *system2) prepareHDWallet(t *testing.T) *devkeys.MnemonicDevKeys {
	hdWallet, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	return hdWallet
}

func (s *system2) prepareWorld(t *testing.T) (*interopgen.WorldDeployment, *interopgen.WorldOutput) {
	// Build the world configuration from the recipe and the HD wallet
	worldCfg, err := s.recipe.Build(s.hdWallet)
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

	return worldDeployment, worldOutput
}

func (s *system2) prepareL1(t *testing.T) (*fakebeacon.FakeBeacon, *geth.GethInstance) {
	// Create a fake Beacon node to hold on to blobs created by the L1 miner, and to serve them to L2
	genesisTimestampL1 := s.worldOutput.L1.Genesis.Timestamp
	blockTimeL1 := uint64(6)
	blobPath := t.TempDir()
	bcn := fakebeacon.NewBeacon(s.logger.New("role", "l1_cl"),
		filepath.Join(blobPath, "l1_cl"), genesisTimestampL1, blockTimeL1)
	t.Cleanup(func() {
		_ = bcn.Close()
	})
	require.NoError(t, bcn.Start("127.0.0.1:0"))
	beaconApiAddr := bcn.BeaconAddr()
	require.NotEmpty(t, beaconApiAddr, "beacon API listener must be up")

	l1FinalizedDistance := uint64(3)
	l1Clock := clock.SystemClock
	// Start the L1 chain
	l1Geth, err := geth.InitL1(
		blockTimeL1,
		l1FinalizedDistance,
		s.worldOutput.L1.Genesis,
		l1Clock,
		filepath.Join(blobPath, "l1_el"),
		bcn)

	require.NoError(t, err)
	require.NoError(t, l1Geth.Node.Start())
	t.Cleanup(func() {
		_ = l1Geth.Close()
	})
	return bcn, l1Geth
}

// TODO: break up this monolith
func (s *system2) prepareL2(t *testing.T, id string, l2Out *interopgen.L2Output) l2Set {
	logger := s.logger.New("role", "op-node-"+id)
	jwtPath := writeDefaultJWT(t)
	name := "l2-" + id
	l2Geth, err := geth.InitL2(name, l2Out.Genesis, jwtPath)
	require.NoError(t, err)
	require.NoError(t, l2Geth.Node.Start())
	t.Cleanup(func() {
		_ = l2Geth.Close()
	})
	// TODO register the op-geth node

	seqP2PSecret, err := s.hdWallet.Secret(devkeys.ChainOperatorKey{
		ChainID: l2Out.Genesis.Config.ChainID,
		Role:    devkeys.SequencerP2PRole,
	})
	require.NoError(t, err)

	nodeCfg := &node.Config{
		L1: &node.PreparedL1Endpoint{
			Client: client.NewBaseRPCClient(endpoint.DialRPC(
				endpoint.PreferAnyRPC,
				s.l1.UserRPC(),
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
			BeaconAddr: s.beacon.BeaconAddr(),
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
	proposerSecret, err := s.hdWallet.Secret(devkeys.ChainOperatorKey{
		ChainID: l2Out.Genesis.Config.ChainID,
		Role:    devkeys.ProposerRole,
	})
	require.NoError(t, err)
	proposerCLIConfig := &l2os.CLIConfig{
		L1EthRpc:          s.l1.UserRPC().RPC(),
		RollupRpc:         opNode.UserRPC().RPC(),
		DGFAddress:        s.worldDeployment.L2s[id].DisputeGameFactoryProxy.Hex(),
		ProposalInterval:  6 * time.Second,
		DisputeGameType:   254, // Fast game type
		PollInterval:      500 * time.Millisecond,
		TxMgrConfig:       setuputils.NewTxMgrConfig(s.l1.UserRPC(), proposerSecret),
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

	// op-batcher
	batcherSecret, err := s.hdWallet.Secret(devkeys.ChainOperatorKey{
		ChainID: l2Out.Genesis.Config.ChainID,
		Role:    devkeys.BatcherRole,
	})
	require.NoError(t, err)
	batcherCLIConfig := &bss.CLIConfig{
		L1EthRpc:                 s.l1.UserRPC().RPC(),
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
		TxMgrConfig:              setuputils.NewTxMgrConfig(s.l1.UserRPC(), batcherSecret),
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

	return l2Set{opNode, l2Geth, proposer, batcher}
}

// prepareL2s creates the L2s for the system. it is a simple loop that adds the L2s
// by calling addL2 for each L2 in the world output
func (s *system2) prepareL2s(t *testing.T) map[string]l2Set {
	l2s := make(map[string]l2Set)
	for id, l2Out := range s.worldOutput.L2s {
		l2s[id] = s.prepareL2(t, id, l2Out)
	}
	return l2s
}

// addL2 adds an L2 to the system by creating the resources for it
// and then assigning them to the system's map of L2s
func (s *system2) addL2(t *testing.T, id string, output *interopgen.L2Output) {
	if s.l2s == nil {
		s.l2s = make(map[string]l2Set)
	}
	s.l2s[id] = s.prepareL2(t, id, output)
}

// prepare sets up the system for testing
// it is a monlith that creates all the resources by calling the other prepare functions
func (s *system2) prepare(t *testing.T) {

	s.logger = testlog.Logger(t, log.LevelInfo)

	s.hdWallet = s.prepareHDWallet(t)

	s.worldDeployment, s.worldOutput = s.prepareWorld(t)

	s.beacon, s.l1 = s.prepareL1(t)

	s.l2s = s.prepareL2s(t)

	// TODO op-supervisor
}

// gettter functions for the individual L2s
// TODO: maybe the caller is just better off using the map directly, it reads nicely
func (s *system2) getBatcher(id string) *bss.BatcherService {
	return s.l2s[id].batcher
}
func (s *system2) getProposer(id string) *l2os.ProposerService {
	return s.l2s[id].proposer
}
func (s *system2) getOpNode(id string) *opnode.Opnode {
	return s.l2s[id].opNode
}
func (s *system2) getL2Geth(id string) *geth.GethInstance {
	return s.l2s[id].l2Geth
}

func (s *system2) getL2IDs() []string {
	ids := make([]string, 0, len(s.l2s))
	for id := range s.l2s {
		ids = append(ids, id)
	}
	return ids
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
