package l2

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-program/client/l2/engineapi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

type OracleBackedL2Chain struct {
	log       log.Logger
	oracle    Oracle
	chainCfg  *params.ChainConfig
	engine    consensus.Engine
	head      *types.Header
	safe      *types.Header
	finalized *types.Header
	vmCfg     vm.Config

	// Inserted blocks
	blocks map[common.Hash]*types.Block
	db     ethdb.KeyValueStore
}

var _ engineapi.EngineBackend = (*OracleBackedL2Chain)(nil)

func NewOracleBackedL2Chain(logger log.Logger, oracle Oracle, chainCfg *params.ChainConfig, l2Head common.Hash) (*OracleBackedL2Chain, error) {
	head := oracle.BlockByHash(l2Head)
	logger.Info("Loaded L2 head", "hash", head.Hash(), "number", head.Number())
	return &OracleBackedL2Chain{
		log:      logger,
		oracle:   oracle,
		chainCfg: chainCfg,
		engine:   beacon.New(nil),

		// Treat the agreed starting head as finalized - nothing before it can be disputed
		head:      head.Header(),
		safe:      head.Header(),
		finalized: head.Header(),
		blocks:    make(map[common.Hash]*types.Block),
		db:        NewOracleBackedDB(oracle),
	}, nil
}

func (o *OracleBackedL2Chain) CurrentHeader() *types.Header {
	return o.head
}

func (o *OracleBackedL2Chain) GetHeaderByNumber(n uint64) *types.Header {
	// Walk back from current head to the requested block number
	h := o.head
	if h.Number.Uint64() < n {
		return nil
	}
	for h.Number.Uint64() > n {
		h = o.GetHeaderByHash(h.ParentHash)
	}
	return h
}

func (o *OracleBackedL2Chain) GetTd(hash common.Hash, number uint64) *big.Int {
	// Difficulty is always 0 post-merge and bedrock starts post-merge so total difficulty also always 0
	return common.Big0
}

func (o *OracleBackedL2Chain) CurrentSafeBlock() *types.Header {
	return o.safe
}

func (o *OracleBackedL2Chain) CurrentFinalBlock() *types.Header {
	return o.finalized
}

func (o *OracleBackedL2Chain) GetHeaderByHash(hash common.Hash) *types.Header {
	block := o.GetBlockByHash(hash)
	if block == nil {
		return nil
	}
	return block.Header()
}

func (o *OracleBackedL2Chain) GetBlockByHash(hash common.Hash) *types.Block {
	// Check inserted blocks
	block, ok := o.blocks[hash]
	if ok {
		return block
	}
	// Retrieve from the oracle
	block = o.oracle.BlockByHash(hash)
	if block == nil {
		return nil
	}
	return block
}

func (o *OracleBackedL2Chain) GetBlock(hash common.Hash, number uint64) *types.Block {
	block := o.GetBlockByHash(hash)
	if block == nil {
		return nil
	}
	if block.NumberU64() != number {
		return nil
	}
	return block
}

func (o *OracleBackedL2Chain) GetHeader(hash common.Hash, u uint64) *types.Header {
	block := o.GetBlock(hash, u)
	if block == nil {
		return nil
	}
	return block.Header()
}

func (o *OracleBackedL2Chain) HasBlockAndState(hash common.Hash, number uint64) bool {
	block := o.GetBlock(hash, number)
	return block != nil
}

func (o *OracleBackedL2Chain) GetCanonicalHash(n uint64) common.Hash {
	header := o.GetHeaderByNumber(n)
	if header == nil {
		return common.Hash{}
	}
	return header.Hash()
}

func (o *OracleBackedL2Chain) GetVMConfig() *vm.Config {
	return &o.vmCfg
}

func (o *OracleBackedL2Chain) Config() *params.ChainConfig {
	return o.chainCfg
}

func (o *OracleBackedL2Chain) Engine() consensus.Engine {
	return o.engine
}

func (o *OracleBackedL2Chain) StateAt(root common.Hash) (*state.StateDB, error) {
	return state.New(root, state.NewDatabase(rawdb.NewDatabase(o.db)), nil)
}

func (o *OracleBackedL2Chain) InsertBlockWithoutSetHead(block *types.Block) error {
	processor, err := engineapi.NewBlockProcessorFromHeader(o, block.Header())
	if err != nil {
		return err
	}
	for i, tx := range block.Transactions() {
		err = processor.AddTx(tx)
		if err != nil {
			return fmt.Errorf("invalid transaction (%d): %w", i, err)
		}
	}
	expected, err := processor.Assemble()
	if err != nil {
		return fmt.Errorf("invalid block: %w", err)
	}
	if expected.Hash() != block.Hash() {
		return fmt.Errorf("block root mismatch, expected: %v, actual: %v", expected.Hash(), block.Hash())
	}
	err = processor.Commit()
	if err != nil {
		return fmt.Errorf("commit block: %w", err)
	}
	o.blocks[block.Hash()] = block
	return nil
}

func (o *OracleBackedL2Chain) SetCanonical(head *types.Block) (common.Hash, error) {
	o.head = head.Header()
	return head.Hash(), nil
}

func (o *OracleBackedL2Chain) SetFinalized(header *types.Header) {
	o.finalized = header
}

func (o *OracleBackedL2Chain) SetSafe(header *types.Header) {
	o.safe = header
}
