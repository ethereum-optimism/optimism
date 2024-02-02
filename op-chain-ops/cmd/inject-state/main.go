package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"

	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	app := &cli.App{
		Name:  "inject-state",
		Usage: "Inject state into the geth db",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "db-path",
				Required: true,
				Usage:    "Path to the geth db",
				EnvVars:  []string{"DB_PATH"},
			},
			&cli.StringFlag{
				Name:     "transition-file-path",
				Required: true,
				Usage:    "Path to the transition file",
				EnvVars:  []string{"TRANSITION_FILE_PATH"},
			},
			&cli.StringFlag{
				Name:     "deploy-config-path",
				Usage:    "Path to the deploy config file",
				Required: true,
				EnvVars:  []string{"DEPLOY_CONFIG_PATH"},
			},
			&cli.IntFlag{
				Name:  "hardfork-block",
				Usage: "Block number to hardfork at",
			},
			&cli.IntFlag{
				Name:  "db-cache",
				Usage: "LevelDB cache size in mb",
				Value: 1024,
			},
			&cli.IntFlag{
				Name:  "db-handles",
				Usage: "LevelDB number of handles",
				Value: 60,
			},
		},
		Action: entrypoint,
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error inject-state", "err", err)
	}
}

func entrypoint(ctx *cli.Context) error {
	transitionFilePath := ctx.String("transition-file-path")

	transitionFile, err := os.Open(transitionFilePath)
	if err != nil {
		log.Error("error opening state file", "err", err)
		return err
	}
	defer transitionFile.Close()

	transitionState := new(core.Genesis)
	if err := json.NewDecoder(transitionFile).Decode(transitionState); err != nil {
		log.Error("failed to decode transition config file", "err", err)
		return err
	}

	deployConfigPath := ctx.String("deploy-config-path")
	deployConfig, err := genesis.NewDeployConfig(deployConfigPath)
	if err != nil {
		return fmt.Errorf("error loading deploy config: %w", err)
	}

	dbCache := ctx.Int("db-cache")
	dbHandles := ctx.Int("db-handles")
	dbPath := ctx.String("db-path")
	db, err := Open(dbPath, dbCache, dbHandles)
	if err != nil {
		return fmt.Errorf("cannot open DB: %w", err)
	}

	hardforkBlock := ctx.Int("hardfork-block")
	hash := rawdb.ReadHeadHeaderHash(db)
	if hardforkBlock != 0 {
		hash = rawdb.ReadCanonicalHash(db, uint64(hardforkBlock))
	}
	num := rawdb.ReadHeaderNumber(db, hash)
	if *num != deployConfig.L2OutputOracleStartingBlockNumber-1 {
		return fmt.Errorf("hardfork block must be the block before the L2OutputOracleStartingBlockNumber")
	}

	log.Info("loading state", "hash", hash, "number", num)
	header := rawdb.ReadHeader(db, hash, *num)

	statedb, err := state.New(header.Root, state.NewDatabaseWithConfig(db, &trie.Config{Preimages: true}), nil)
	if err != nil {
		return err
	}

	config := &state.DumpConfig{
		SkipCode:          false,
		SkipStorage:       false,
		OnlyWithAddresses: false,
		Start:             common.Hash{}.Bytes(),
		Max:               uint64(0),
	}
	state := statedb.RawDump(config)

	// Reset all acounts
	for address := range state.Accounts {
		statedb.SetNonce(address, 0)
		statedb.SetBalance(address, common.Big0)
		statedb.SetCode(address, nil)
		for k := range state.Accounts[address].Storage {
			statedb.SetState(address, k, common.Hash{})
		}
	}
	// Add new accounts
	for addr, account := range transitionState.Alloc {
		statedb.SetNonce(addr, account.Nonce)
		statedb.SetBalance(addr, account.Balance)
		statedb.SetCode(addr, account.Code)
		for k, v := range account.Storage {
			statedb.SetState(addr, k, v)
		}
	}

	// We're done messing around with the database, so we can now commit the changes to the DB.
	// Note that this doesn't actually write the changes to disk.
	log.Info("Committing state DB")
	newRoot, err := statedb.Commit(uint64(hardforkBlock+1), true)
	if err != nil {
		return err
	}

	transitionHeader := CreateHeader(transitionState, header, deployConfig, newRoot)
	// Create the Bedrock transition block from the header. Note that there are no transactions,
	// uncle blocks, or receipts in the Bedrock transition block.
	transitionBlock := types.NewBlock(transitionHeader, nil, nil, nil, trie.NewStackTrie(nil))

	// We did it!
	log.Info(
		"Built Bedrock transition",
		"hash", transitionBlock.Hash(),
		"root", transitionBlock.Root(),
		"number", transitionBlock.NumberU64(),
		"gas-used", transitionBlock.GasUsed(),
		"gas-limit", transitionBlock.GasLimit(),
	)

	// Otherwise we need to write the changes to disk. First we commit the state changes.
	log.Info("Committing trie DB")
	if err := statedb.Database().TrieDB().Commit(newRoot, true); err != nil {
		return err
	}

	// Next we write the Bedrock transition block to the database.
	tD := rawdb.ReadTd(db, transitionBlock.ParentHash(), transitionBlock.NumberU64()-1)
	if transitionState.Config.ChainID.Cmp(big.NewInt(28882)) == 0 {
		tD = big.NewInt(3)
	}
	rawdb.WriteTd(db, transitionBlock.Hash(), transitionBlock.NumberU64(), tD)
	rawdb.WriteBlock(db, transitionBlock)
	rawdb.WriteReceipts(db, transitionBlock.Hash(), transitionBlock.NumberU64(), nil)
	rawdb.WriteCanonicalHash(db, transitionBlock.Hash(), transitionBlock.NumberU64())
	rawdb.WriteHeadBlockHash(db, transitionBlock.Hash())
	rawdb.WriteHeadFastBlockHash(db, transitionBlock.Hash())
	rawdb.WriteHeadHeaderHash(db, transitionBlock.Hash())

	// Make the first Bedrock block a finalized block.
	rawdb.WriteFinalizedBlockHash(db, transitionBlock.Hash())

	// We need to update the chain config to set the correct hardforks.
	genesisHash := rawdb.ReadCanonicalHash(db, 0)
	cfg := rawdb.ReadChainConfig(db, genesisHash)
	if cfg == nil {
		log.Crit("chain config not found")
	}

	// Set the standard options.
	cfg.LondonBlock = transitionBlock.Number()
	cfg.ArrowGlacierBlock = transitionBlock.Number()
	cfg.GrayGlacierBlock = transitionBlock.Number()
	cfg.MergeNetsplitBlock = transitionBlock.Number()
	cfg.TerminalTotalDifficulty = big.NewInt(0)
	cfg.TerminalTotalDifficultyPassed = true

	// Set the Optimism options.
	cfg.BedrockBlock = transitionBlock.Number()
	cfg.RegolithTime = transitionState.Config.RegolithTime
	cfg.CanyonTime = transitionState.Config.CanyonTime
	cfg.ShanghaiTime = transitionState.Config.ShanghaiTime

	cfg.Optimism = &params.OptimismConfig{
		EIP1559Denominator:       transitionState.Config.Optimism.EIP1559Denominator,
		EIP1559Elasticity:        transitionState.Config.Optimism.EIP1559Elasticity,
		EIP1559DenominatorCanyon: transitionState.Config.Optimism.EIP1559DenominatorCanyon,
	}

	// Write the chain config to disk.
	rawdb.WriteChainConfig(db, genesisHash, cfg)

	// Yay!
	log.Info(
		"wrote chain config",
		"1559-denominator", cfg.Optimism.EIP1559Denominator,
		"1559-elasticity", cfg.Optimism.EIP1559Elasticity,
		"1559-denominator-canyon", cfg.Optimism.EIP1559DenominatorCanyon,
	)

	// We're done!
	log.Info(
		"wrote Bedrock transition block",
		"height", transitionBlock.Number,
		"root", transitionBlock.Root().String(),
		"hash", transitionBlock.Hash().String(),
		"timestamp", transitionBlock.Time,
	)

	// Close the database handle
	if err := db.Close(); err != nil {
		return err
	}

	return nil
}

func Open(path string, cache int, handles int) (ethdb.Database, error) {
	chaindataPath := filepath.Join(path, "geth", "chaindata")
	ancientPath := filepath.Join(chaindataPath, "ancient")
	ldb, err := rawdb.Open(rawdb.OpenOptions{
		Type:              "leveldb",
		Directory:         chaindataPath,
		AncientsDirectory: ancientPath,
		Namespace:         "",
		Cache:             cache,
		Handles:           handles,
		ReadOnly:          false,
	})
	if err != nil {
		return nil, err
	}
	return ldb, nil
}

func CreateHeader(g *core.Genesis, parentHeader *types.Header, config *genesis.DeployConfig, root common.Hash) *types.Header {
	// Create the header for the Bedrock transition block.
	head := &types.Header{
		Number:        big.NewInt(int64(config.L2OutputOracleStartingBlockNumber)),
		Nonce:         types.EncodeNonce(g.Nonce),
		Time:          g.Timestamp,
		ParentHash:    g.ParentHash,
		Extra:         g.ExtraData,
		GasLimit:      g.GasLimit,
		GasUsed:       g.GasUsed,
		Difficulty:    g.Difficulty,
		MixDigest:     g.Mixhash,
		Coinbase:      g.Coinbase,
		BaseFee:       g.BaseFee,
		ExcessBlobGas: g.ExcessBlobGas,
		Root:          root,
	}

	head.Extra = []byte{}
	head.Time = uint64(config.L2OutputOracleStartingTimestamp)
	head.Difficulty = big.NewInt(0)
	head.BaseFee = common.Big0
	head.ParentHash = parentHeader.Hash()

	return head
}
