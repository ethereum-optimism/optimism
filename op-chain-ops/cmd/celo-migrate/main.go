package main

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"runtime/debug"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slog"
)

var (
	deployConfigFlag = &cli.PathFlag{
		Name:     "deploy-config",
		Usage:    "Path to the JSON file that was used for the bedrock contracts deployment. A test example can be found here 'op-chain-ops/genesis/testdata/test-deploy-config-full.json' and documentation for the fields is at https://docs.optimism.io/builders/chain-operators/management/configuration",
		Required: true,
	}
	l1DeploymentsFlag = &cli.PathFlag{
		Name:     "l1-deployments",
		Usage:    "Path to L1 deployments JSON file, the output of running the bedrock contracts deployment for the given 'deploy-config'",
		Required: true,
	}
	l1RPCFlag = &cli.StringFlag{
		Name:     "l1-rpc",
		Usage:    "RPC URL for a node of the L1 defined in the 'deploy-config'",
		Required: true,
	}
	l2AllocsFlag = &cli.PathFlag{
		Name:     "l2-allocs",
		Usage:    "Path to L2 genesis allocs file",
		Required: true,
	}
	outfileRollupConfigFlag = &cli.PathFlag{
		Name:     "outfile.rollup-config",
		Usage:    "Path to write the rollup config JSON file, to be provided to op-node with the 'rollup.config' flag",
		Required: true,
	}
	oldDBPathFlag = &cli.StringFlag{
		Name:     "old-db",
		Usage:    "Path to the old Celo chaindata",
		Required: true,
	}
	newDBPathFlag = &cli.StringFlag{
		Name:     "new-db",
		Usage:    "Path to write migrated Celo chaindata",
		Required: true,
	}
	batchSizeFlag = &cli.Uint64Flag{
		Name:        "batch-size",
		Usage:       "Dry run the upgrade by not committing the database",
		DefaultText: "10000",
	}
	memoryLimitFlag = &cli.Int64Flag{
		Name:        "memory-limit",
		Usage:       "Memory limit in MB",
		DefaultText: "7500",
	}
	dryRunFlag = &cli.BoolFlag{ // TODO
		Name:  "dry-run",
		Usage: "Dry run the upgrade by not committing the database",
	}
	clearAllFlag = &cli.BoolFlag{
		Name:  "clear-all",
		Usage: "Use this to start with a fresh new database",
	}
	clearNonAncientsFlag = &cli.BoolFlag{
		Name:  "clear-nonAncients",
		Usage: "Use to keep migrated ancients, but not non-ancients",
	}

	flags = []cli.Flag{
		deployConfigFlag,
		l1DeploymentsFlag,
		l1RPCFlag,
		l2AllocsFlag,
		outfileRollupConfigFlag,
		oldDBPathFlag,
		newDBPathFlag,
		batchSizeFlag,
		memoryLimitFlag,
		dryRunFlag,
		clearAllFlag,
		clearNonAncientsFlag,
	}

	// TODO: read those form the deploy config
	// TODO(pl): select values
	EIP1559Denominator       = uint64(50)
	EIP1559DenominatorCanyon = uint64(250)
	EIP1559Elasticity        = uint64(10)

	OutFilePerm = os.FileMode(0o440)
)

func main() {

	// Check that `rsync` command is available. We use this to copy the db excluding ancients, which we will copy separately
	if _, err := exec.LookPath("rsync"); err != nil {
		log.Error("Please install `rsync` to use this script")
		return
	}

	app := &cli.App{
		Name:  "migrate",
		Usage: "Migrate Celo block data and state to a CeL2 DB",
		Flags: flags,
		Action: func(ctx *cli.Context) error {
			deployConfig := ctx.Path("deploy-config")
			l1Deployments := ctx.Path("l1-deployments")
			l1RPC := ctx.String("l1-rpc")
			l2AllocsPath := ctx.Path("l2-allocs")
			outfileRollupConfig := ctx.Path("outfile.rollup-config")
			oldDBPath := ctx.String("old-db")
			newDBPath := ctx.String("new-db")
			batchSize := ctx.Uint64("batch-size")
			memoryLimit := ctx.Int64("memory-limit")
			dryRun := ctx.Bool("dry-run")
			clearAll := ctx.Bool("clear-all")
			clearNonAncients := ctx.Bool("clear-nonAncients")

			color := isatty.IsTerminal(os.Stderr.Fd())
			handler := log.NewTerminalHandlerWithLevel(os.Stderr, slog.LevelDebug, color)
			oplog.SetGlobalLogHandler(handler)

			if err := runBlockMigration(oldDBPath, newDBPath, batchSize, memoryLimit, clearAll, clearNonAncients); err != nil {
				return fmt.Errorf("failed to run block migration: %v", err)
			}

			if err := runStateMigration(newDBPath, deployConfig, l1Deployments, l1RPC, l2AllocsPath, outfileRollupConfig, dryRun); err != nil {
				return fmt.Errorf("failed to run state migration: %v", err)
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error in migration", "err", err)
	}
	log.Info("Finished migration successfully!")
}

func runBlockMigration(oldDBPath, newDBPath string, batchSize uint64, memoryLimit int64, clearAll, clearNonAncients bool) error {
	debug.SetMemoryLimit(memoryLimit * 1 << 20) // Set memory limit, converting from MB to bytes

	var err error

	if clearAll {
		if err = os.RemoveAll(newDBPath); err != nil {
			return fmt.Errorf("failed to remove new database: %v", err)
		}
	}
	if clearNonAncients {
		if err = cleanupNonAncientDb(newDBPath); err != nil {
			return fmt.Errorf("failed to cleanup non-ancient database: %v", err)
		}
	}

	if err = createEmptyNewDb(newDBPath); err != nil {
		return fmt.Errorf("failed to create new database: %v", err)
	}

	var numAncientsNew uint64
	if numAncientsNew, err = migrateAncientsDb(oldDBPath, newDBPath, batchSize); err != nil {
		return fmt.Errorf("failed to migrate ancients database: %v", err)
	}

	var numNonAncients uint64
	if numNonAncients, err = migrateNonAncientsDb(oldDBPath, newDBPath, numAncientsNew-1, batchSize); err != nil {
		return fmt.Errorf("failed to migrate non-ancients database: %v", err)
	}

	log.Info("Block Migration Completed", "migratedAncients", numAncientsNew, "migratedNonAncients", numNonAncients)

	return nil
}

func runStateMigration(newDBPath, deployConfig, l1Deployments, l1RPC, l2AllocsPath, outfileRollupConfig string, dryRun bool) error {
	// Read deployment configuration
	log.Info("Deploy config", "path", deployConfig)
	config, err := genesis.NewDeployConfig(deployConfig)
	if err != nil {
		return err
	}

	if config.DeployCeloContracts {
		return errors.New("DeployCeloContracts is not supported in migration")
	}
	if config.FundDevAccounts {
		return errors.New("FundDevAccounts is not supported in migration")
	}

	// Try reading the L1 deployment information
	deployments, err := genesis.NewL1Deployments(l1Deployments)
	if err != nil {
		return fmt.Errorf("cannot read L1 deployments at %s: %w", l1Deployments, err)
	}
	config.SetDeployments(deployments)

	// Get latest block information from L1
	var l1StartBlock *types.Block
	client, err := ethclient.Dial(l1RPC)
	if err != nil {
		return fmt.Errorf("cannot dial %s: %w", l1RPC, err)
	}

	if config.L1StartingBlockTag == nil {
		l1StartBlock, err = client.BlockByNumber(context.Background(), nil)
		if err != nil {
			return fmt.Errorf("cannot fetch latest block: %w", err)
		}
		tag := rpc.BlockNumberOrHashWithHash(l1StartBlock.Hash(), true)
		config.L1StartingBlockTag = (*genesis.MarshalableRPCBlockNumberOrHash)(&tag)
	} else if config.L1StartingBlockTag.BlockHash != nil {
		l1StartBlock, err = client.BlockByHash(context.Background(), *config.L1StartingBlockTag.BlockHash)
		if err != nil {
			return fmt.Errorf("cannot fetch block by hash: %w", err)
		}
	} else if config.L1StartingBlockTag.BlockNumber != nil {
		l1StartBlock, err = client.BlockByNumber(context.Background(), big.NewInt(config.L1StartingBlockTag.BlockNumber.Int64()))
		if err != nil {
			return fmt.Errorf("cannot fetch block by number: %w", err)
		}
	}

	// Ensure that there is a starting L1 block
	if l1StartBlock == nil {
		return fmt.Errorf("no starting L1 block")
	}

	// Sanity check the config. Do this after filling in the L1StartingBlockTag
	// if it is not defined.
	if err := config.Check(); err != nil {
		return err
	}

	log.Info("Using L1 Start Block", "number", l1StartBlock.Number(), "hash", l1StartBlock.Hash().Hex())

	// Build the L2 genesis block
	l2Allocs, err := genesis.LoadForgeAllocs(l2AllocsPath)
	if err != nil {
		return err
	}

	l2Genesis, err := genesis.BuildL2Genesis(config, l2Allocs, l1StartBlock)
	if err != nil {
		return fmt.Errorf("error creating l2 genesis: %w", err)
	}

	// Write changes to state to actual state database
	cel2Header, err := applyStateMigrationChanges(l2Genesis, newDBPath, !dryRun)
	if err != nil {
		return err
	}
	log.Info("Updated Cel2 state")

	rollupConfig, err := config.RollupConfig(l1StartBlock, cel2Header.Hash(), cel2Header.Number.Uint64())
	if err != nil {
		return err
	}
	if err := rollupConfig.Check(); err != nil {
		return fmt.Errorf("generated rollup config does not pass validation: %w", err)
	}

	log.Info("Writing rollup config", "file", outfileRollupConfig)
	if err := jsonutil.WriteJSON(outfileRollupConfig, rollupConfig, OutFilePerm); err != nil {
		return err
	}

	return nil
}
