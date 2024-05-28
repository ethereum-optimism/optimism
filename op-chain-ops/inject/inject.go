package inject

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

func InjectState(transitionState *core.Genesis, db ethdb.Database, deployConfig *genesis.DeployConfig, hardforkBlock int) error {
	hash := rawdb.ReadHeadHeaderHash(db)
	if hardforkBlock != 0 {
		hash = rawdb.ReadCanonicalHash(db, uint64(hardforkBlock))
	}
	num := rawdb.ReadHeaderNumber(db, hash)
	if *num != deployConfig.L2OutputOracleStartingBlockNumber-1 {
		return fmt.Errorf("hardfork block must be the block before the L2OutputOracleStartingBlockNumber")
	}
	header := rawdb.ReadHeader(db, hash, *num)

	statedb, err := state.New(header.Root, state.NewDatabaseWithConfig(db, &triedb.Config{Preimages: true}), nil)
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
		statedb.SetNonce(common.HexToAddress(address), 0)
		statedb.SetBalance(common.HexToAddress(address), uint256.NewInt(0))
		statedb.SetCode(common.HexToAddress(address), nil)
		for k := range state.Accounts[address].Storage {
			statedb.SetState(common.HexToAddress(address), k, common.Hash{})
		}
	}
	// Add new accounts
	for addr, account := range transitionState.Alloc {
		statedb.SetNonce(addr, account.Nonce)
		statedb.SetBalance(addr, uint256.MustFromBig(account.Balance))
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
	td := rawdb.ReadTd(db, transitionBlock.ParentHash(), transitionBlock.NumberU64()-1)
	rawdb.WriteTd(db, transitionBlock.Hash(), transitionBlock.NumberU64(), td)
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
		return fmt.Errorf("chain config not found")
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
	cfg.CancunTime = transitionState.Config.CancunTime
	cfg.EcotoneTime = transitionState.Config.EcotoneTime

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

	return nil
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
