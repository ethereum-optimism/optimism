package l2

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-program/l2/engineapi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
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
	// TODO implement me
	// TODO: Will require an oracle backed db
	return state.New(root, state.NewDatabase(rawdb.NewDatabase(memorydb.New())), nil)
}

func (o *OracleBackedL2Chain) InsertBlockWithoutSetHead(block *types.Block) error {
	// TODO: Need to actually apply the block here so the state is available
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

func handleError(err error) {
	panic(err)
}
