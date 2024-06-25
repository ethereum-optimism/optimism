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
		Usage:    "Path to the JSON file that was used for the l1 contracts deployment. A test example can be found here 'op-chain-ops/genesis/testdata/test-deploy-config-full.json' and documentation for the fields is at https://docs.optimism.io/builders/chain-operators/management/configuration",
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
		Usage:    "Path to L2 genesis allocs file. You can find instructions on how to generate this file in the README",
		Required: true,
	}
	outfileRollupConfigFlag = &cli.PathFlag{
		Name:     "outfile.rollup-config",
		Usage:    "Path to write the rollup config JSON file, to be provided to op-node with the 'rollup.config' flag",
		Required: true,
	}
	oldDBPathFlag = &cli.PathFlag{
		Name:     "old-db",
		Usage:    "Path to the old Celo chaindata dir, can be found at '<datadir>/celo/chaindata'",
		Required: true,
	}
	newDBPathFlag = &cli.PathFlag{
		Name:     "new-db",
		Usage:    "Path to write migrated Celo chaindata, note the new node implementation expects to find this chaindata at the following path '<datadir>/geth/chaindata",
		Required: true,
	}
	batchSizeFlag = &cli.Uint64Flag{
		Name:  "batch-size",
		Usage: "Batch size to use for block migration, larger batch sizes can speed up migration but require more memory. If increasing the batch size consider also increasing the memory-limit",
		Value: 50000, // TODO(Alec) optimize default parameters
	}
	bufferSizeFlag = &cli.Uint64Flag{
		Name:  "buffer-size",
		Usage: "Buffer size to use for ancient block migration channels. Defaults to 0. Included to facilitate testing for performance improvements.",
		Value: 0,
	}
	memoryLimitFlag = &cli.Int64Flag{
		Name:  "memory-limit",
		Usage: "Memory limit in MiB, should be set lower than the available amount of memory in your system to prevent out of memory errors",
		Value: 7500,
	}
	clearAllFlag = &cli.BoolFlag{
		Name:  "clear-all",
		Usage: "Use this to start with a fresh new db, deleting all data including ancients. CAUTION: Re-migrating ancients takes time.",
	}
	keepNonAncientsFlag = &cli.BoolFlag{
		Name:  "keep-non-ancients",
		Usage: "CAUTION: Not recommended for production. Use to keep all data in the new db as is, including any partially migrated non-ancient blocks and state data. If non-ancient blocks are partially migrated, the script will attempt to resume the migration.",
	}
	onlyAncientsFlag = &cli.BoolFlag{
		Name:  "only-ancients",
		Usage: "Use to only migrate ancient blocks. Ignored when running full migration",
	}

	blockMigrationFlags = []cli.Flag{
		onlyAncientsFlag,
		oldDBPathFlag,
		newDBPathFlag,
		batchSizeFlag,
		bufferSizeFlag,
		memoryLimitFlag,
		clearAllFlag,
		keepNonAncientsFlag,
	}
	stateMigrationFlags = []cli.Flag{
		newDBPathFlag,
		deployConfigFlag,
		l1DeploymentsFlag,
		l1RPCFlag,
		l2AllocsFlag,
		outfileRollupConfigFlag,
	}
	// Ignore onlyAncients flag and duplicate newDBPathFlag for full migration
	fullMigrationFlags = append(blockMigrationFlags[1:], stateMigrationFlags[1:]...)
)

type blockMigrationOptions struct {
	oldDBPath       string
	newDBPath       string
	batchSize       uint64
	bufferSize      uint64
	memoryLimit     int64
	clearAll        bool
	keepNonAncients bool
	onlyAncients    bool
}

type stateMigrationOptions struct {
	deployConfig        string
	l1Deployments       string
	l1RPC               string
	l2AllocsPath        string
	outfileRollupConfig string
	newDBPath           string
}

func parseBlockMigrationOptions(ctx *cli.Context) blockMigrationOptions {
	return blockMigrationOptions{
		oldDBPath:       ctx.String(oldDBPathFlag.Name),
		newDBPath:       ctx.String(newDBPathFlag.Name),
		batchSize:       ctx.Uint64(batchSizeFlag.Name),
		bufferSize:      ctx.Uint64(bufferSizeFlag.Name),
		memoryLimit:     ctx.Int64(memoryLimitFlag.Name),
		clearAll:        ctx.Bool(clearAllFlag.Name),
		keepNonAncients: ctx.Bool(keepNonAncientsFlag.Name),
		onlyAncients:    ctx.Bool(onlyAncientsFlag.Name),
	}
}

func parseStateMigrationOptions(ctx *cli.Context) stateMigrationOptions {
	return stateMigrationOptions{
		newDBPath:           ctx.String(newDBPathFlag.Name),
		deployConfig:        ctx.Path(deployConfigFlag.Name),
		l1Deployments:       ctx.Path(l1DeploymentsFlag.Name),
		l1RPC:               ctx.String(l1RPCFlag.Name),
		l2AllocsPath:        ctx.Path(l2AllocsFlag.Name),
		outfileRollupConfig: ctx.Path(outfileRollupConfigFlag.Name),
	}
}

func main() {

	color := isatty.IsTerminal(os.Stderr.Fd())
	handler := log.NewTerminalHandlerWithLevel(os.Stderr, slog.LevelDebug, color)
	oplog.SetGlobalLogHandler(handler)

	log.Info("Beginning Cel2 Migration")

	app := &cli.App{
		Name:  "celo-migrate",
		Usage: "Migrate Celo block and state data to a CeL2 DB",
		Commands: []*cli.Command{
			{
				Name:    "blocks",
				Aliases: []string{"b"},
				Usage:   "Migrate Celo block data to a CeL2 DB",
				Flags:   blockMigrationFlags,
				Action: func(ctx *cli.Context) error {
					return runBlockMigration(parseBlockMigrationOptions(ctx))
				},
			},
			{
				Name:    "state",
				Aliases: []string{"s"},
				Usage:   "Migrate Celo state data to a CeL2 DB. Makes necessary state changes and generates a rollup config file.",
				Flags:   stateMigrationFlags,
				Action: func(ctx *cli.Context) error {
					return runStateMigration(parseStateMigrationOptions(ctx))
				},
			},
			{
				Name:    "full",
				Aliases: []string{"f", "all", "a"},
				Usage:   "Perform a full migration of both block and state data to a CeL2 DB",
				Flags:   fullMigrationFlags,
				Action: func(ctx *cli.Context) error {
					if err := runBlockMigration(parseBlockMigrationOptions(ctx)); err != nil {
						return fmt.Errorf("failed to run block migration: %w", err)
					}

					if err := runStateMigration(parseStateMigrationOptions(ctx)); err != nil {
						return fmt.Errorf("failed to run state migration: %w", err)
					}

					return nil
				},
			},
		},
		OnUsageError: func(ctx *cli.Context, err error, isSubcommand bool) error {
			if isSubcommand {
				return err
			}
			_ = cli.ShowAppHelp(ctx)
			return fmt.Errorf("please provide a valid command")
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error in migration", "err", err)
	}
	log.Info("Finished migration successfully!")
}

func runBlockMigration(opts blockMigrationOptions) error {

	// Check that `rsync` command is available. We use this to copy the db excluding ancients, which we will copy separately
	if _, err := exec.LookPath("rsync"); err != nil {
		return fmt.Errorf("please install `rsync` to run block migration")
	}

	debug.SetMemoryLimit(opts.memoryLimit * 1 << 20) // Set memory limit, converting from MiB to bytes

	log.Info("Block Migration Started", "oldDBPath", opts.oldDBPath, "newDBPath", opts.newDBPath, "batchSize", opts.batchSize, "memoryLimit", opts.memoryLimit, "clearAll", opts.clearAll, "keepNonAncients", opts.keepNonAncients, "onlyAncients", opts.onlyAncients)

	var err error

	if err = createNewDbIfNotExists(opts.newDBPath); err != nil {
		return fmt.Errorf("failed to create new database: %w", err)
	}

	if opts.clearAll {
		if err = os.RemoveAll(opts.newDBPath); err != nil {
			return fmt.Errorf("failed to remove new database: %w", err)
		}
	} else if !opts.keepNonAncients {
		if err = cleanupNonAncientDb(opts.newDBPath); err != nil {
			return fmt.Errorf("failed to reset non-ancient database: %w", err)
		}
	}

	var numAncientsNewBefore uint64
	var numAncientsNewAfter uint64
	if numAncientsNewBefore, numAncientsNewAfter, err = migrateAncientsDb(opts.oldDBPath, opts.newDBPath, opts.batchSize, opts.bufferSize); err != nil {
		return fmt.Errorf("failed to migrate ancients database: %w", err)
	}

	var numNonAncients uint64
	if !opts.onlyAncients {
		if numNonAncients, err = migrateNonAncientsDb(opts.oldDBPath, opts.newDBPath, numAncientsNewAfter-1, opts.batchSize); err != nil {
			return fmt.Errorf("failed to migrate non-ancients database: %w", err)
		}
	} else {
		log.Info("Skipping non-ancients migration")
	}

	log.Info("Block Migration Completed", "migratedAncients", numAncientsNewAfter-numAncientsNewBefore, "migratedNonAncients", numNonAncients)

	return nil
}

func runStateMigration(opts stateMigrationOptions) error {
	log.Info("State Migration Started", "newDBPath", opts.newDBPath, "deployConfig", opts.deployConfig, "l1Deployments", opts.l1Deployments, "l1RPC", opts.l1RPC, "l2AllocsPath", opts.l2AllocsPath, "outfileRollupConfig", opts.outfileRollupConfig)

	// Read deployment configuration
	config, err := genesis.NewDeployConfig(opts.deployConfig)
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
	deployments, err := genesis.NewL1Deployments(opts.l1Deployments)
	if err != nil {
		return fmt.Errorf("cannot read L1 deployments at %s: %w", opts.l1Deployments, err)
	}
	config.SetDeployments(deployments)

	// Get latest block information from L1
	var l1StartBlock *types.Block
	client, err := ethclient.Dial(opts.l1RPC)
	if err != nil {
		return fmt.Errorf("cannot dial %s: %w", opts.l1RPC, err)
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
	l2Allocs, err := genesis.LoadForgeAllocs(opts.l2AllocsPath)
	if err != nil {
		return err
	}

	l2Genesis, err := genesis.BuildL2Genesis(config, l2Allocs, l1StartBlock)
	if err != nil {
		return fmt.Errorf("error creating l2 genesis: %w", err)
	}

	// Write changes to state to actual state database
	cel2Header, err := applyStateMigrationChanges(config, l2Genesis, opts.newDBPath)
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

	log.Info("Writing rollup config", "file", opts.outfileRollupConfig)
	if err := jsonutil.WriteJSON(opts.outfileRollupConfig, rollupConfig, OutFilePerm); err != nil {
		return err
	}

	log.Info("State Migration Completed")

	return nil
}
