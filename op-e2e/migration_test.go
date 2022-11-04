package op_e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis/migration"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-service/backoff"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"
)

type migrationTestConfig struct {
	enabled           bool
	l1URL             string
	l2Path            string
	ovmAddrsPath      string
	evmAddrsPath      string
	ovmAllowancesPath string
	ovmMessagesPath   string
	evmMessagePath    string
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
		//panic("must specify an l2 data path")
	}
}

type storageSlot struct {
	addr  string
	slot  string
	value string
}

const (
	hardhatImage = "ethereumoptimism/hardhat-node:latest"
	forkBlock    = 15822707
)

var hardcodedSlots = []storageSlot{
	{
		"0xdE1FCfB0851916CA5101820A69b13a4E276bd81F",
		"0x0",
		"0x000000000000000000000000f39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
	},
	{
		"0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1",
		"0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103",
		"0x000000000000000000000000f39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
	},
	{
		"0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1",
		"0x33",
		"0x000000000000000000000000f39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
	},
}

func tCtx(t *testing.T, parent context.Context) context.Context {
	ctx, cancel := context.WithCancel(parent)
	t.Cleanup(cancel)
	return ctx
}

func TestMigration(t *testing.T) {
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

	ctnr, err := dkr.ContainerCreate(ctx, &container.Config{
		Image: hardhatImage,
		Env: []string{
			fmt.Sprintf("FORK_STARTING_BLOCK=%s", forkBlock),
			fmt.Sprintf("FORK_URL=%s", config.l1URL),
			"FORK_CHAIN_ID=1",
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			"8545/tcp": []nat.PortBinding{
				{
					"127.0.0.1", "8545",
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

	var rpcClient *rpc.Client
	var ethClient *ethclient.Client
	require.NoError(t, backoff.Do(10, backoff.Exponential(), func() error {
		rpcClient, err = rpc.Dial("http://127.0.0.1:8545")
		if err != nil {
			log.Warn("error connecting to forked L1, trying again", "err", err)
			return err
		}

		ethClient = ethclient.NewClient(rpcClient)
		_, err = ethClient.ChainID(ctx)
		if err != nil {
			log.Warn("error connecting to forked L1, trying again", "err", err)
		}
		return err
	}), "error connecting to forked L1")

	for _, slot := range hardcodedSlots {
		lgr.Info("setting storage slot", "addr", slot.addr, "slot", slot.slot)
		require.NoError(t, rpcClient.Call(nil, "hardhat_setStorageAt", slot.addr, slot.slot, slot.value))
	}

	//go func() {
	//	blockTick := time.NewTicker(12 * time.Second)
	//
	//	for {
	//		select {
	//		case <-blockTick.C:
	//			err := rpcClient.CallContext(ctx, nil, "evm_mine")
	//			if err != nil {
	//				lgr.Error("error mining new block", "err", err)
	//				continue
	//			}
	//			lgr.Debug("mined block")
	//		case <-ctx.Done():
	//			return
	//		}
	//	}
	//}()

	workdir := path.Join(os.TempDir(), fmt.Sprintf("migration-test-%d", time.Now().Unix()))
	require.NoError(t, os.MkdirAll(workdir, 0o755), "error creating work directory")
	workdir = "/tmp/migration-tmp-workdir"

	lgr.Info("performing L1 migration")
	t.Cleanup(func() {
		// Clean up the mainnet-forked deployment artifacts
		require.NoError(t, os.RemoveAll(path.Join(cwd, "..", "packages", "contracts-bedrock", "deployments", "mainnet-forked")))
	})
	migrateL1(t)
	lgr.Info("l1 successfully migrated!")

	lgr.Info("extracting L2 datadir")
	//untar(t, config.l2Path, "/tmp/migration-tmp-workdir")

	lgr.Info("performing L2 migration")
	//migrateL2(t, workdir)
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
		"mainnet-forked",
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

func migrateL2(t *testing.T, workdir string) {
	deployConfigFile, err := os.Open(path.Join(cwd, "..", "packages", "contracts-bedrock", "deploy-config", "mainnet-forked.json"))
	require.NoError(t, err, "error opening deploy config")
	defer deployConfigFile.Close()
	deployConfig := new(genesis.DeployConfig)
	dec := json.NewDecoder(deployConfigFile)
	require.NoError(t, dec.Decode(deployConfig))

	_ = &migration.Config{
		DeployConfig:          deployConfig,
		OVMAddressesPath:      config.ovmAddrsPath,
		EVMAddressesPath:      config.evmAddrsPath,
		OVMAllowancesPath:     config.ovmAllowancesPath,
		OVMMessagesPath:       config.ovmMessagesPath,
		EVMMessagesPath:       config.evmMessagePath,
		Network:               "mainnet-forked",
		HardhatDeployments:    nil,
		L1URL:                 config.l1URL,
		StartingL1BlockNumber: forkBlock,
		L2DBPath:              path.Join(workdir, "geth"),
		DryRun:                true,
	}
}
