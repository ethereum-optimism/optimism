package l2

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-program/l2/engineapi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

type Oracle interface {
	NodeByHash(ctx context.Context, nodeHash common.Hash) ([]byte, error)
	BlockByHash(ctx context.Context, blockHash common.Hash) (*types.Block, error)
}

type OracleBackedL2Chain struct {
	ctx       context.Context
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

func NewOracleBackedL2Chain(ctx context.Context, logger log.Logger, oracle Oracle, chainCfg *params.ChainConfig, l2Head common.Hash) (*OracleBackedL2Chain, error) {
	head, err := oracle.BlockByHash(ctx, l2Head)
	if err != nil {
		return nil, fmt.Errorf("loading l2 head: %w", err)
	}
	return &OracleBackedL2Chain{
		ctx:      ctx, // TODO: Probably should be passed to each EngineBackend function?
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

func (o *OracleBackedL2Chain) CurrentBlock() *types.Header {
	return o.head
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
	block, err := o.oracle.BlockByHash(o.ctx, hash)
	if err != nil {
		handleError(err)
	}
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
	// Walk back from current head to the requested block number
	h := o.head
	if h.Number.Uint64() < n {
		return common.Hash{}
	}
	for h.Number.Uint64() > n {
		h = o.GetHeaderByHash(h.ParentHash)
	}
	return h.Hash()
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
	var usedGas = uint64(0)
	gp := new(core.GasPool).AddGas(block.GasLimit())
	parent := o.GetBlockByHash(block.ParentHash())
	statedb, err := o.StateAt(parent.Root())
	if err != nil {
		return err
	}
	var receipts types.Receipts
	for i, tx := range block.Transactions() {
		statedb.SetTxContext(tx.Hash(), i)
		receipt, err := core.ApplyTransaction(o.chainCfg, o, &block.Header().Coinbase, gp, statedb, block.Header(), tx, &usedGas, o.vmCfg)
		if err != nil {
			return err
		}
		receipts = append(receipts, receipt)
	}
	// TODO: Ideally would call engine.Finalize but currently it only applies beacon chain withdrawals which we don't use
	//o.engine.Finalize(o, block.Header(), statedb, block.Transactions(), block.Uncles(), block.Withdrawals())
	err = validateState(o.chainCfg, block, statedb, receipts, usedGas)
	if err != nil {
		return fmt.Errorf("invalid block: %w", err)
	}
	root, err := statedb.Commit(o.chainCfg.IsEIP158(block.Header().Number))
	if err != nil {
		panic(fmt.Sprintf("state write error: %v", err))
	}
	if err := statedb.Database().TrieDB().Commit(root, false); err != nil {
		panic(fmt.Sprintf("trie write error: %v", err))
	}
	o.blocks[block.Hash()] = block
	return nil
}
func validateState(config *params.ChainConfig, block *types.Block, statedb *state.StateDB, receipts types.Receipts, usedGas uint64) error {
	header := block.Header()
	if block.GasUsed() != usedGas {
		return fmt.Errorf("invalid gas used (remote: %d local: %d)", block.GasUsed(), usedGas)
	}
	// Validate the received block's bloom with the one derived from the generated receipts.
	// For valid blocks this should always validate to true.
	rbloom := types.CreateBloom(receipts)
	if rbloom != header.Bloom {
		return fmt.Errorf("invalid bloom (remote: %x  local: %x)", header.Bloom, rbloom)
	}
	// Tre receipt Trie's root (R = (Tr [[H1, R1], ... [Hn, Rn]]))
	receiptSha := types.DeriveSha(receipts, trie.NewStackTrie(nil))
	if receiptSha != header.ReceiptHash {
		return fmt.Errorf("invalid receipt root hash (remote: %x local: %x)", header.ReceiptHash, receiptSha)
	}
	// Validate the state root against the received state root and throw
	// an error if they don't match.
	if root := statedb.IntermediateRoot(config.IsEIP158(header.Number)); header.Root != root {
		return fmt.Errorf("invalid merkle root (remote: %x local: %x)", header.Root, root)
	}
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

func handleError(err error) {
	panic(err)
}
