package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"
	"time"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/blobstore"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/el"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/fakebeacon"
	e2egeth "github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/opnode"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/services"
	"github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	nodesync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	"github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

const pcdL1BeaconBlockTime = 12

type pcdBootConfig struct {
	l1RPC             string
	l1Client          *ethclient.Client
	genesisPath       string
	rollupPath        string
	dependencySetPath string
}

type pcdBootResult struct {
	expectedGenesisHash common.Hash
	genesisTime         uint64
	l2BlockTime         uint64
	block0              *types.Block
	block1              *types.Block
	bootToBlock1        time.Duration
}

func requirePCDOpReth(t *testing.T) string {
	t.Helper()
	lgr := testlog.Logger(t, log.LevelCrit)
	execPath, err := (rustbin.Spec{
		SrcDir:  "rust",
		Package: "op-reth",
		Binary:  "op-reth",
	}).EnsureExists(t.Context(), lgr)
	if err == nil {
		return execPath
	}

	message := "op-reth binary is required; run 'cd rust && mise exec -- just build-op-reth' or set RUST_BINARY_PATH_OP_RETH"
	if os.Getenv("CI") == "true" {
		require.NoError(t, err, message)
	}
	t.Skipf("%s: %v", message, err)
	return ""
}

func bootPCDFromArtifacts(t *testing.T, cfg pcdBootConfig) pcdBootResult {
	t.Helper()
	lgr := testlog.Logger(t, log.LevelCrit)

	genesisData, err := os.ReadFile(cfg.genesisPath)
	require.NoErrorf(t, err, "read committed genesis artifact %s", cfg.genesisPath)
	var genesis core.Genesis
	require.NoErrorf(t, json.Unmarshal(genesisData, &genesis), "decode committed genesis artifact %s", cfg.genesisPath)
	expectedGenesisHash := genesis.ToBlock().Hash()

	rollupConfig, err := readPCDRollupConfig(cfg.rollupPath)
	require.NoErrorf(t, err, "read committed rollup artifact %s", cfg.rollupPath)
	dependencySetData, err := os.ReadFile(cfg.dependencySetPath)
	require.NoErrorf(t, err, "read committed dependency-set artifact %s", cfg.dependencySetPath)
	dependencySet, err := depset.ParseJSONDependencySet(bytes.NewReader(dependencySetData))
	require.NoErrorf(t, err, "parse committed dependency-set artifact %s", cfg.dependencySetPath)

	l1Genesis, err := cfg.l1Client.BlockByNumber(t.Context(), big.NewInt(0))
	require.NoError(t, err)
	l1ChainID, err := cfg.l1Client.ChainID(t.Context())
	require.NoError(t, err)
	l1ChainConfig := pcdPragueL1ChainConfig(l1ChainID)
	require.Equal(t, l1ChainID, l1ChainConfig.ChainID, "L1 chain config must match eth_chainId")
	require.Nil(t, l1ChainConfig.OsakaTime, "Prague Anvil must not use the post-Prague Osaka fork")
	require.NotNil(t, l1ChainConfig.BlobScheduleConfig)
	require.Nil(t, l1ChainConfig.BlobScheduleConfig.Osaka, "Prague Anvil must not use the Osaka blob schedule")

	// Anvil provides only the L1 execution endpoint, so use a fake beacon for the
	// L1 consensus endpoint.
	beacon := fakebeacon.NewBeacon(lgr.New("role", "l1_cl"), blobstore.New(), l1Genesis.Time(), pcdL1BeaconBlockTime)
	require.NoError(t, beacon.Start("127.0.0.1:0"))
	t.Cleanup(func() {
		_ = beacon.Close()
	})

	jwtPath := e2eutils.WriteDefaultJWT(t)
	jwtSecret := readPCDJWTSecret(t, jwtPath)
	elCtx, cancelEL := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancelEL()
	l2EL, err := el.InitL2(elCtx, el.L2Config{
		Kind:            services.ELKindOpReth,
		Name:            "pcd-boot",
		GenesisJSONPath: cfg.genesisPath,
		JWTPath:         jwtPath,
		Logger:          lgr.New("role", "l2_el"),
		DataDir:         t.TempDir(),
	})
	require.NoErrorf(t, err, "start op-reth from committed genesis artifact %s", cfg.genesisPath)
	t.Cleanup(func() {
		require.NoError(t, l2EL.Close())
	})

	l2Client, err := ethclient.DialContext(t.Context(), l2EL.UserRPC().RPC())
	require.NoError(t, err)
	t.Cleanup(l2Client.Close)

	// This node config follows newSupernodeNodeConfig in
	// op-e2e/interop/supersystem_l2.go. The PCD journey supplies the endpoints and artifacts.
	nodeConfig := &config.Config{
		L1: &config.L1EndpointConfig{
			L1NodeAddr:       cfg.l1RPC,
			L1TrustRPC:       false,
			L1RPCKind:        sources.RPCKindBasic,
			BatchSize:        20,
			HttpPollInterval: 100 * time.Millisecond,
			MaxConcurrency:   10,
		},
		L1ChainConfig: l1ChainConfig,
		L2: &config.L2EndpointConfig{
			L2EngineAddr:      l2EL.AuthRPC().RPC(),
			L2EngineJWTSecret: jwtSecret,
		},
		Beacon: &config.L1BeaconEndpointConfig{
			BeaconAddr: beacon.BeaconAddr(),
		},
		Driver: driver.Config{
			SequencerEnabled:   true,
			SequencerConfDepth: 2,
		},
		Rollup:                 *rollupConfig,
		DependencySet:          dependencySet,
		RPC:                    rpc.CLIConfig{ListenAddr: "127.0.0.1", ListenPort: 0, EnableAdmin: true},
		P2P:                    nil,
		Metrics:                metrics.CLIConfig{},
		Pprof:                  oppprof.CLIConfig{},
		L1EpochPollInterval:    2 * time.Second,
		ConfigPersistence:      config.DisabledConfigPersistence{},
		Sync:                   nodesync.Config{SyncMode: nodesync.CLSync},
		AltDA:                  altda.CLIConfig{},
		ExperimentalOPStackAPI: true,
	}
	require.NoError(t, nodeConfig.Check(), "committed artifacts must produce a valid op-node configuration")

	bootStarted := time.Now()
	node, err := opnode.NewOpnode(lgr.New("role", "l2_cl"), nodeConfig, clock.SystemClock, func(err error) {
		t.Errorf("op-node requested early shutdown: %v", err)
	})
	require.NoErrorf(t, err, "start op-node from committed rollup artifact %s and dependency-set artifact %s", cfg.rollupPath, cfg.dependencySetPath)
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		require.NoError(t, node.Stop(stopCtx))
	})

	block1, err := e2egeth.WaitForBlock(big.NewInt(1), l2Client, e2egeth.WithAbsoluteTimeout(2*time.Minute))
	require.NoErrorf(t, err, "wait for L2 block 1 from artifacts genesis=%s rollup=%s depset=%s", cfg.genesisPath, cfg.rollupPath, cfg.dependencySetPath)
	bootToBlock1 := time.Since(bootStarted)
	block0, err := l2Client.BlockByNumber(t.Context(), big.NewInt(0))
	require.NoErrorf(t, err, "read L2 block 0 from committed genesis artifact %s", cfg.genesisPath)

	return pcdBootResult{
		expectedGenesisHash: expectedGenesisHash,
		genesisTime:         rollupConfig.Genesis.L2Time,
		l2BlockTime:         rollupConfig.BlockTime,
		block0:              block0,
		block1:              block1,
		bootToBlock1:        bootToBlock1,
	}
}

// pcdPragueL1ChainConfig copies params.AllDevChainProtocolChanges and removes
// Osaka to match the Anvil Prague chain. It also copies BlobScheduleConfig so
// this test does not change shared state.
func pcdPragueL1ChainConfig(chainID *big.Int) *params.ChainConfig {
	chainConfig := *params.AllDevChainProtocolChanges
	chainConfig.ChainID = new(big.Int).Set(chainID)
	chainConfig.OsakaTime = nil
	if chainConfig.BlobScheduleConfig != nil {
		blobSchedule := *chainConfig.BlobScheduleConfig
		blobSchedule.Osaka = nil
		chainConfig.BlobScheduleConfig = &blobSchedule
	}
	return &chainConfig
}

func readPCDJWTSecret(t *testing.T, path string) [32]byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	decoded, err := hexutil.Decode(string(data))
	require.NoError(t, err)
	require.Len(t, decoded, 32)
	var secret [32]byte
	copy(secret[:], decoded)
	return secret
}
