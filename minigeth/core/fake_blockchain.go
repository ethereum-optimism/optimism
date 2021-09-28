package core

import (
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/oracle"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

type BlockChain struct {
	// TODO: write stub BlockChain
	chainConfig *params.ChainConfig // Chain & network configuration
	engine      consensus.Engine
	lastBlock   *types.Header
}

func NewBlockChain(parent *types.Header) *BlockChain {
	return &BlockChain{
		chainConfig: params.MainnetChainConfig,
		engine:      &ethash.Ethash{},
		lastBlock:   parent,
	}
}

// Config retrieves the chain's fork configuration.
func (bc *BlockChain) Config() *params.ChainConfig { return bc.chainConfig }

// Engine retrieves the blockchain's consensus engine.
func (bc *BlockChain) Engine() consensus.Engine { return bc.engine }

// GetHeader retrieves a block header from the database by hash and number,
// caching it if found.
func (bc *BlockChain) GetHeader(hash common.Hash, number uint64) *types.Header {
	if hash == bc.lastBlock.Hash() {
		return bc.lastBlock
	}
	oracle.PrefetchBlock(big.NewInt(int64(number)), true, nil)

	var ret types.Header
	err := rlp.DecodeBytes(oracle.Preimage(hash), &ret)
	if err != nil {
		log.Fatal(err)
	}

	return &ret
}

func (bc *BlockChain) CurrentHeader() *types.Header {
	log.Fatal("CurrentHeader")
	// this right?
	return bc.lastBlock
}

// GetHeaderByHash retrieves a block header from the database by hash, caching it if
// found.
func (bc *BlockChain) GetHeaderByHash(hash common.Hash) *types.Header {
	log.Fatal("GetHeaderByHash", hash)
	return nil
}

// GetHeaderByNumber retrieves a block header from the database by number,
// caching it (associated with its hash) if found.
func (bc *BlockChain) GetHeaderByNumber(number uint64) *types.Header {
	log.Fatal("GetHeaderByNumber", number)
	return nil
}
