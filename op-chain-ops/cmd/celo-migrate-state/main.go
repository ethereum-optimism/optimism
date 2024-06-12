package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/holiman/uint256"
	"github.com/mattn/go-isatty"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
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
	dbPathFlag = &cli.StringFlag{
		Name:     "db-path",
		Usage:    "Path to the Celo database, not including the `celo/chaindata` part",
		Required: true,
	}
	dryRunFlag = &cli.BoolFlag{
		Name:  "dry-run",
		Usage: "Dry run the upgrade by not committing the database",
	}

	flags = []cli.Flag{
		deployConfigFlag,
		l1DeploymentsFlag,
		l1RPCFlag,
		l2AllocsFlag,
		outfileRollupConfigFlag,
		dbPathFlag,
		dryRunFlag,
	}

	// TODO: read those form the deploy config
	// TODO(pl): select values
	EIP1559Denominator       = uint64(50)
	EIP1559DenominatorCanyon = uint64(250)
	EIP1559Elasticity        = uint64(10)

	OutFilePerm = os.FileMode(0o440)
)

func main() {
	color := isatty.IsTerminal(os.Stderr.Fd())
	handler := log.NewTerminalHandler(os.Stderr, color)
	oplog.SetGlobalLogHandler(handler)

	app := &cli.App{
		Name:  "migrate",
		Usage: "Migrate Celo state to a CeL2 DB",
		Flags: flags,
		Action: func(ctx *cli.Context) error {
			deployConfig := ctx.Path("deploy-config")
			l1Deployments := ctx.Path("l1-deployments")
			l1RPC := ctx.String("l1-rpc")
			l2AllocsPath := ctx.Path("l2-allocs")
			outfileRollupConfig := ctx.Path("outfile.rollup-config")
			dbPath := ctx.String("db-path")
			dryRun := ctx.Bool("dry-run")

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
			cel2Header, err := ApplyMigrationChangesToDB(l2Genesis, dbPath, !dryRun)
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
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error in migration", "err", err)
	}
	log.Info("Finished migration successfully!")
}

func ApplyMigrationChangesToDB(genesis *core.Genesis, dbPath string, commit bool) (*types.Header, error) {
	log.Info("Opening Celo database", "dbPath", dbPath)
	ldb, err := openCeloDb(dbPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open DB: %w", err)
	}
	log.Info("Loaded Celo L1 DB", "db", ldb)

	// Grab the hash of the tip of the legacy chain.
	hash := rawdb.ReadHeadHeaderHash(ldb)
	log.Info("Reading chain tip from database", "hash", hash)

	// Grab the header number.
	num := rawdb.ReadHeaderNumber(ldb, hash)
	if num == nil {
		return nil, fmt.Errorf("cannot find header number for %s", hash)
	}
	log.Info("Reading chain tip num from database", "number", num)

	// Grab the full header.
	header := rawdb.ReadHeader(ldb, hash, *num)
	log.Info("Read header from database", "header", header)

	// We need to update the chain config to set the correct hardforks.
	genesisHash := rawdb.ReadCanonicalHash(ldb, 0)
	cfg := rawdb.ReadChainConfig(ldb, genesisHash)
	if cfg == nil {
		log.Crit("chain config not found")
	}
	log.Info("Read chain config from database", "config", cfg)

	// Set up the backing store.
	// TODO(pl): Do we need the preimages setting here?
	underlyingDB := state.NewDatabaseWithConfig(ldb, &triedb.Config{Preimages: true})

	// Open up the state database.
	db, err := state.New(header.Root, underlyingDB, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot open StateDB: %w", err)
	}

	// So far we applied changes in the memory VM and collected changes in the genesis struct
	// Now we iterate through all accounts that have been written there and set them inside the statedb.
	// This will change the state root
	// Another property is that the total balance changes must be 0
	accountCounter := 0
	overwriteCounter := 0
	for k, v := range genesis.Alloc {
		accountCounter++
		if db.Exist(k) {
			equal := bytes.Equal(db.GetCode(k), v.Code)

			log.Warn("Operating on existing state", "account", k, "equalCode", equal)
			overwriteCounter++
		}
		// TODO(pl): decide what to do with existing accounts.
		db.CreateAccount(k)

		// CreateAccount above copied the balance, check if we change it
		if db.GetBalance(k).Cmp(uint256.MustFromBig(v.Balance)) != 0 {
			// TODO(pl): make this a hard error once the migration has been tested more
			log.Warn("Moving account changed native balance", "address", k, "oldBalance", db.GetBalance(k), "newBalance", v.Balance)
		}

		db.SetNonce(k, v.Nonce)
		db.SetBalance(k, uint256.MustFromBig(v.Balance))
		db.SetCode(k, v.Code)
		db.SetStorage(k, v.Storage)

		log.Info("Moved account", "address", k)
	}
	log.Info("Migrated OP contracts into state DB", "copiedAccounts", accountCounter, "overwrittenAccounts", overwriteCounter)

	migrationBlock := new(big.Int).Add(header.Number, common.Big1)

	// We're done messing around with the database, so we can now commit the changes to the DB.
	// Note that this doesn't actually write the changes to disk.
	log.Info("Committing state DB")
	newRoot, err := db.Commit(migrationBlock.Uint64(), true)
	if err != nil {
		return nil, err
	}

	// Create the header for the Bedrock transition block.
	cel2Header := &types.Header{
		ParentHash:  header.Hash(),
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    predeploys.SequencerFeeVaultAddr,
		Root:        newRoot,
		TxHash:      types.EmptyTxsHash,
		ReceiptHash: types.EmptyReceiptsHash,
		Bloom:       types.Bloom{},
		Difficulty:  new(big.Int).Set(common.Big0),
		Number:      migrationBlock,
		GasLimit:    header.GasLimit,
		GasUsed:     0,
		Time:        uint64(time.Now().Unix()), // TODO(pl): Needed to avoid L1-L2 time mismatches
		Extra:       []byte("CeL2 migration"),
		MixDigest:   common.Hash{},
		Nonce:       types.BlockNonce{},
		BaseFee:     new(big.Int).Set(header.BaseFee),
	}
	log.Info("Build Cel2 migration header", "header", cel2Header)

	// Create the Bedrock transition block from the header. Note that there are no transactions,
	// uncle blocks, or receipts in the Bedrock transition block.
	cel2Block := types.NewBlock(cel2Header, nil, nil, nil, trie.NewStackTrie(nil))

	// We did it!
	log.Info(
		"Built Cel2 migration block",
		"hash", cel2Block.Hash(),
		"root", cel2Block.Root(),
		"number", cel2Block.NumberU64(),
		"gas-used", cel2Block.GasUsed(),
		"gas-limit", cel2Block.GasLimit(),
	)

	// If we're not actually writing this to disk, then we're done.
	if !commit {
		log.Info("Dry run complete")
		return nil, nil
	}

	// Otherwise we need to write the changes to disk. First we commit the state changes.
	log.Info("Committing trie DB")
	if err := db.Database().TrieDB().Commit(newRoot, true); err != nil {
		return nil, err
	}

	// Next we write the Cel2 genesis block to the database.
	rawdb.WriteTd(ldb, cel2Block.Hash(), cel2Block.NumberU64(), cel2Block.Difficulty())
	rawdb.WriteBlock(ldb, cel2Block)
	rawdb.WriteReceipts(ldb, cel2Block.Hash(), cel2Block.NumberU64(), nil)
	rawdb.WriteCanonicalHash(ldb, cel2Block.Hash(), cel2Block.NumberU64())
	rawdb.WriteHeadBlockHash(ldb, cel2Block.Hash())
	rawdb.WriteHeadFastBlockHash(ldb, cel2Block.Hash())
	rawdb.WriteHeadHeaderHash(ldb, cel2Block.Hash())

	// Mark the first CeL2 block as finalized
	rawdb.WriteFinalizedBlockHash(ldb, cel2Block.Hash())

	// Set the standard options.
	cfg.LondonBlock = cel2Block.Number()
	cfg.BerlinBlock = cel2Block.Number()
	cfg.ArrowGlacierBlock = cel2Block.Number()
	cfg.GrayGlacierBlock = cel2Block.Number()
	cfg.MergeNetsplitBlock = cel2Block.Number()
	cfg.TerminalTotalDifficulty = big.NewInt(0)
	cfg.TerminalTotalDifficultyPassed = true
	cfg.ShanghaiTime = &cel2Header.Time
	cfg.CancunTime = &cel2Header.Time

	// Set the Optimism options.
	cfg.BedrockBlock = cel2Block.Number()
	// Enable Regolith from the start of Bedrock
	cfg.RegolithTime = new(uint64) // what are those? do we need those?
	cfg.Optimism = &params.OptimismConfig{
		EIP1559Denominator:       EIP1559Denominator,
		EIP1559DenominatorCanyon: EIP1559DenominatorCanyon,
		EIP1559Elasticity:        EIP1559Elasticity,
	}
	cfg.CanyonTime = &cel2Header.Time
	cfg.EcotoneTime = &cel2Header.Time
	cfg.Cel2Time = &cel2Header.Time

	// Write the chain config to disk.
	// TODO(pl): Why do we need to write this with the genesis hash, not `cel2Block.Hash()`?`
	rawdb.WriteChainConfig(ldb, genesisHash, cfg)
	log.Info("Wrote updated chain config", "config", cfg)

	// We're done!
	log.Info(
		"Wrote CeL2 migration block",
		"height", cel2Header.Number,
		"root", cel2Header.Root.String(),
		"hash", cel2Header.Hash().String(),
		"timestamp", cel2Header.Time,
	)

	// Close the database handle
	if err := ldb.Close(); err != nil {
		return nil, err
	}

	return cel2Header, nil
}

// Opens a Celo database, stored in the `celo` subfolder
func openCeloDb(path string) (ethdb.Database, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	chaindataPath := filepath.Join(path, "celo", "chaindata")
	ancientPath := filepath.Join(chaindataPath, "ancient")
	ldb, err := rawdb.Open(rawdb.OpenOptions{
		Type:              "leveldb",
		Directory:         chaindataPath,
		AncientsDirectory: ancientPath,
		Namespace:         "",
		Cache:             1024,
		Handles:           60,
		ReadOnly:          false,
	})
	if err != nil {
		return nil, err
	}
	return ldb, nil
}
