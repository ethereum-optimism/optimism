package op_e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	batchermetrics "github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	proposermetrics "github.com/ethereum-optimism/optimism/op-proposer/metrics"
	l2os "github.com/ethereum-optimism/optimism/op-proposer/proposer"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-bindings/hardhat"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis/migration_action"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-service/backoff"
)

type migrationTestConfig struct {
	enabled           bool
	l1URL             string
	l2Path            string
	ovmAddrsPath      string
	evmAddrsPath      string
	ovmAllowancesPath string
	ovmMessagesPath   string
	evmMessagesPath   string
}

var config migrationTestConfig

var cwd string

func init() {
	if os.Getenv("OP_E2E_MIGRATION_ENABLED") != "true" {
		return
	}

	iCwd, err := os.Getwd()
	if err != nil {
		panic("failed to get cwd")
	}
	cwd = iCwd

	config.enabled = true
	config.l1URL = os.Getenv("OP_E2E_MIGRATION_L1_URL")
	if config.l1URL == "" {
		panic("must specify an L1 url")
	}
	config.l2Path = os.Getenv("OP_E2E_MIGRATION_L2_DATA_PATH")
	if config.l2Path == "" {
		panic("must specify an l2 data path")
	}

	migrationDataDir := path.Join(cwd, "..", "packages", "migration-data", "data")
	config.ovmAddrsPath = path.Join(migrationDataDir, "ovm-addresses.json")
	config.evmAddrsPath = path.Join(migrationDataDir, "evm-addresses.json")
	config.ovmAllowancesPath = path.Join(migrationDataDir, "ovm-allowances.json")
	config.ovmMessagesPath = path.Join(migrationDataDir, "ovm-messages.json")
	config.evmMessagesPath = path.Join(migrationDataDir, "evm-messages.json")
}

type storageSlot struct {
	addr  string
	slot  string
	value string
}

const (
	networkName  = "mainnet-forked"
	hardhatImage = "docker.io/ethereumoptimism/hardhat-node:latest"
	forkedL1URL  = "http://127.0.0.1:8545"
)

var hardcodedSlots = []storageSlot{
	// Address manager owner
	{
		"0xdE1FCfB0851916CA5101820A69b13a4E276bd81F",
		"0x0",
		"0x000000000000000000000000f39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
	},
	// L1SB Proxy Owner
	{
		"0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1",
		"0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103",
		"0x000000000000000000000000f39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
	},
	// L1XDM Owner
	{
		"0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1",
		"0x33",
		"0x000000000000000000000000f39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
	},
}

func TestMigration(t *testing.T) {
	InitParallel(t)
	if !config.enabled {
		t.Skipf("skipping migration tests")
		return
	}

	lgr := testlog.Logger(t, log.LvlDebug)
	lgr.Info("starting forked L1")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dkr, err := client.NewClientWithOpts(client.FromEnv)
	require.NoError(t, err, "error connecting to Docker")

	//_, err = dkr.ImagePull(context.Background(), hardhatImage, types.ImagePullOptions{})
	//require.NoError(t, err, "error pulling hardhat image")

	realL1Client, err := ethclient.Dial(config.l1URL)
	require.NoError(t, err)
	headBlock, err := realL1Client.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	// Have to specify a small confirmation depth here to prevent the Hardhat fork
	// from timing out in the middle of contract deployments.
	forkBlock, err := realL1Client.BlockByNumber(ctx, new(big.Int).Sub(headBlock.Number, big.NewInt(10)))
	require.NoError(t, err)
	forkBlockNumber := forkBlock.NumberU64()

	lgr.Info("writing deploy config")
	deployCfg := e2eutils.ForkedDeployConfig(t, e2eutils.DefaultMnemonicConfig, forkBlock)
	deployCfgPath := path.Join(cwd, "..", "packages", "contracts-bedrock", "deploy-config", "mainnet-forked.json")
	f, err := os.OpenFile(deployCfgPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o744)
	require.NoError(t, err)
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	require.NoError(t, enc.Encode(deployCfg))

	ctnr, err := dkr.ContainerCreate(ctx, &container.Config{
		Image: hardhatImage,
		Env: []string{
			fmt.Sprintf("FORK_STARTING_BLOCK=%d", forkBlockNumber),
			fmt.Sprintf("FORK_URL=%s", config.l1URL),
			"FORK_CHAIN_ID=1",
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			"8545/tcp": []nat.PortBinding{
				{
					HostIP: "127.0.0.1", HostPort: "8545",
				},
			},
		},
	}, nil, nil, "")
	require.NoError(t, err, "error creating hardhat container")

	err = dkr.ContainerStart(ctx, ctnr.ID, types.ContainerStartOptions{})
	require.NoError(t, err)

	t.Cleanup(func() {
		timeout := 5 * time.Second
		err = dkr.ContainerStop(context.Background(), ctnr.ID, &timeout)
		require.NoError(t, err)
	})

	var forkedL1RPC *rpc.Client
	var forkedL1Client *ethclient.Client
	require.NoError(t, backoff.Do(10, backoff.Exponential(), func() error {
		forkedL1RPC, err = rpc.Dial(forkedL1URL)
		if err != nil {
			lgr.Warn("error connecting to forked L1, trying again", "err", err)
			return err
		}

		forkedL1Client = ethclient.NewClient(forkedL1RPC)
		_, err = forkedL1Client.ChainID(ctx)
		if err != nil {
			lgr.Warn("error connecting to forked L1, trying again", "err", err)
		}
		return err
	}), "error connecting to forked L1")

	for _, slot := range hardcodedSlots {
		lgr.Info("setting storage slot", "addr", slot.addr, "slot", slot.slot)
		require.NoError(t, forkedL1RPC.Call(nil, "hardhat_setStorageAt", slot.addr, slot.slot, slot.value))
	}

	tag := rpc.BlockNumberOrHash(*deployCfg.L1StartingBlockTag)
	l1BlockHash, ok := tag.Hash()
	require.True(t, ok, "invalid l1 starting block tag")
	l1Block, err := forkedL1Client.BlockByHash(ctx, l1BlockHash)
	require.NoError(t, err)

	workdir := "/tmp/migration-tmp-workdir"
	require.NoError(t, os.MkdirAll(workdir, 0o755))

	lgr.Info("performing L1 migration")
	t.Cleanup(func() {
		// Clean up the mainnet-forked deployment artifacts
		require.NoError(t, os.RemoveAll(path.Join(cwd, "..", "packages", "contracts-bedrock", "deployments", networkName)))
	})
	migrateL1(t)
	lgr.Info("l1 successfully migrated!")

	hh, err := hardhat.New(networkName, []string{}, []string{
		path.Join(cwd, "..", "packages", "contracts-bedrock", "deployments"),
		path.Join(cwd, "..", "packages", "contracts-periphery", "deployments"),
		path.Join(cwd, "..", "packages", "contracts", "deployments"),
	})
	require.NoError(t, err)

	require.NoError(t, deployCfg.GetDeployedAddresses(hh))

	go makeBlocks(ctx, forkedL1RPC, lgr)

	lgr.Info("extracting L2 datadir")
	untar(t, config.l2Path, workdir)

	lgr.Info("performing L2 migration")
	migRes := migrateL2(t, workdir, deployCfg, l1Block.NumberU64())

	lgr.Info("starting new L2 system")

	portal, err := hh.GetDeployment("OptimismPortalProxy")
	require.NoError(t, err)
	sysConfig, err := hh.GetDeployment("SystemConfigProxy")
	require.NoError(t, err)
	l2OS, err := hh.GetDeployment("L2OutputOracleProxy")
	require.NoError(t, err)

	jwt := writeDefaultJWT(t)
	nodeCfg := defaultNodeConfig("geth", jwt)
	nodeCfg.DataDir = workdir
	ethCfg := &ethconfig.Config{
		NetworkId: deployCfg.L2ChainID,
	}
	gethNode, _, err := createGethNode(true, nodeCfg, ethCfg, nil)
	require.NoError(t, err)

	require.NoError(t, gethNode.Start())
	t.Cleanup(func() {
		require.NoError(t, gethNode.Close())
	})

	secrets, err := e2eutils.DefaultMnemonicConfig.Secrets()
	require.NoError(t, err)

	// Don't log state snapshots in test output
	snapLog := log.New()
	snapLog.SetHandler(log.DiscardHandler())
	rollupNodeConfig := &node.Config{
		L1: &node.L1EndpointConfig{
			L1NodeAddr:       forkedL1URL,
			L1TrustRPC:       false,
			L1RPCKind:        sources.RPCKindBasic,
			RateLimit:        0,
			BatchSize:        20,
			HttpPollInterval: 12 * time.Second,
		},
		L2: &node.L2EndpointConfig{
			L2EngineAddr:      gethNode.HTTPAuthEndpoint(),
			L2EngineJWTSecret: testingJWTSecret,
		},
		L2Sync: &node.PreparedL2SyncEndpoint{Client: nil, TrustRPC: false},
		Driver: driver.Config{
			VerifierConfDepth:  0,
			SequencerConfDepth: 0,
			SequencerEnabled:   true,
		},
		Rollup: rollup.Config{
			Genesis: rollup.Genesis{
				L1: eth.BlockID{
					Hash:   forkBlock.Hash(),
					Number: forkBlock.NumberU64(),
				},
				L2: eth.BlockID{
					Hash:   migRes.TransitionBlockHash,
					Number: migRes.TransitionHeight,
				},
				L2Time:       migRes.TransitionTimestamp,
				SystemConfig: e2eutils.SystemConfigFromDeployConfig(deployCfg),
			},
			BlockTime:              deployCfg.L2BlockTime,
			MaxSequencerDrift:      deployCfg.MaxSequencerDrift,
			SeqWindowSize:          deployCfg.SequencerWindowSize,
			ChannelTimeout:         deployCfg.ChannelTimeout,
			L1ChainID:              new(big.Int).SetUint64(deployCfg.L1ChainID),
			L2ChainID:              new(big.Int).SetUint64(deployCfg.L2ChainID),
			BatchInboxAddress:      deployCfg.BatchInboxAddress,
			DepositContractAddress: portal.Address,
			L1SystemConfigAddress:  sysConfig.Address,
		},
		P2PSigner: &p2p.PreparedSigner{Signer: p2p.NewLocalSigner(secrets.SequencerP2P)},
		RPC: node.RPCConfig{
			ListenAddr:  "127.0.0.1",
			ListenPort:  0,
			EnableAdmin: true,
		},
		L1EpochPollInterval: 4 * time.Second,
	}
	rollupLog := log.New()
	rollupNodeConfig.Rollup.LogDescription(rollupLog, chaincfg.L2ChainIDToNetworkName)
	rollupNode, err := node.New(ctx, rollupNodeConfig, rollupLog, snapLog, "", metrics.NewMetrics(""))
	require.NoError(t, err)

	require.NoError(t, rollupNode.Start(ctx))
	t.Cleanup(func() {
		require.NoError(t, rollupNode.Close())
	})

	batcher, err := bss.NewBatchSubmitterFromCLIConfig(bss.CLIConfig{
		L1EthRpc:           forkedL1URL,
		L2EthRpc:           gethNode.WSEndpoint(),
		RollupRpc:          rollupNode.HTTPEndpoint(),
		MaxChannelDuration: 1,
		MaxL1TxSize:        120_000,
		TargetL1TxSize:     100_000,
		TargetNumFrames:    1,
		ApproxComprRatio:   0.4,
		SubSafetyMargin:    4,
		PollInterval:       50 * time.Millisecond,
		TxMgrConfig:        newTxMgrConfig(forkedL1URL, secrets.Batcher),
		LogConfig: oplog.CLIConfig{
			Level:  "info",
			Format: "text",
		},
	}, lgr.New("module", "batcher"), batchermetrics.NoopMetrics)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		batcher.StopIfRunning(ctx)
	})

	proposer, err := l2os.NewL2OutputSubmitterFromCLIConfig(l2os.CLIConfig{
		L1EthRpc:          forkedL1URL,
		RollupRpc:         rollupNode.HTTPEndpoint(),
		L2OOAddress:       l2OS.Address.String(),
		PollInterval:      50 * time.Millisecond,
		AllowNonFinalized: true,
		TxMgrConfig:       newTxMgrConfig(forkedL1URL, secrets.Proposer),
		LogConfig: oplog.CLIConfig{
			Level:  "info",
			Format: "text",
		},
	}, lgr.New("module", "proposer"), proposermetrics.NoopMetrics)
	require.NoError(t, err)
	t.Cleanup(func() {
		proposer.Stop()
	})
}

func untar(t *testing.T, src, dst string) {
	cmd := exec.Command("tar", "-xzvf", src, "--strip-components=6", "-C", dst)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run(), "error untarring data")
}

func migrateL1(t *testing.T) {
	cmd := exec.Command(
		"yarn",
		"hardhat",
		"--network",
		networkName,
		"deploy",
		"--tags",
		"migration",
	)
	cmd.Env = os.Environ()
	cmd.Env = append(
		cmd.Env,
		"CHAIN_ID=1",
		"L1_RPC=http://127.0.0.1:8545",
		"PRIVATE_KEY_DEPLOYER=ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = path.Join(cwd, "..", "packages", "contracts-bedrock")
	require.NoError(t, cmd.Run(), "error migrating L1")
}

func migrateL2(t *testing.T, workdir string, deployConfig *genesis.DeployConfig, startingBlockNumber uint64) *genesis.MigrationResult {
	migCfg := &migration_action.Config{
		DeployConfig:      deployConfig,
		OVMAddressesPath:  config.ovmAddrsPath,
		EVMAddressesPath:  config.evmAddrsPath,
		OVMAllowancesPath: config.ovmAllowancesPath,
		OVMMessagesPath:   config.ovmMessagesPath,
		EVMMessagesPath:   config.evmMessagesPath,
		Network:           "mainnet",
		HardhatDeployments: []string{
			path.Join(cwd, "..", "packages", "contracts", "deployments"),
			path.Join(cwd, "..", "packages", "contracts-periphery", "deployments"),
		},
		L1URL:                 config.l1URL,
		StartingL1BlockNumber: startingBlockNumber,
		L2DBPath:              workdir,
		DryRun:                false,
	}

	res, err := migration_action.Migrate(migCfg)
	require.NoError(t, err)
	return res
}

func makeBlocks(ctx context.Context, rpcClient *rpc.Client, lgr log.Logger) {
	blockTick := time.NewTicker(12 * time.Second)

	for {
		select {
		case <-blockTick.C:
			err := rpcClient.CallContext(ctx, nil, "evm_mine")
			if err != nil {
				lgr.Error("error mining new block", "err", err)
				continue
			}
			lgr.Debug("mined block")
		case <-ctx.Done():
			return
		}
	}
}
