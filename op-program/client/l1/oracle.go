package l1

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-program/client/mpt"
	"github.com/ethereum-optimism/optimism/op-program/preimage"
)

type Oracle interface {
	// HeaderByBlockHash retrieves the block header with the given hash.
	HeaderByBlockHash(blockHash common.Hash) eth.BlockInfo

	// TransactionsByBlockHash retrieves the transactions from the block with the given hash.
	TransactionsByBlockHash(blockHash common.Hash) (eth.BlockInfo, types.Transactions)

	// ReceiptsByBlockHash retrieves the receipts from the block with the given hash.
	ReceiptsByBlockHash(blockHash common.Hash) (eth.BlockInfo, types.Receipts)
}

// PreimageOracle implements Oracle using by interfacing with the pure preimage.Oracle
// to fetch pre-images to decode into the requested data.
type PreimageOracle struct {
	oracle preimage.Oracle
	hint   preimage.Hinter
}

var _ Oracle = (*PreimageOracle)(nil)

func NewPreimageOracle(raw preimage.Oracle, hint preimage.Hinter) *PreimageOracle {
	return &PreimageOracle{
		oracle: raw,
		hint:   hint,
	}
}

func (p *PreimageOracle) headerByBlockHash(blockHash common.Hash) *types.Header {
	p.hint.Hint(BlockHeaderHint(blockHash))
	headerRlp := p.oracle.Get(preimage.Keccak256Key(blockHash))
	var header types.Header
	if err := rlp.DecodeBytes(headerRlp, &header); err != nil {
		panic(fmt.Errorf("invalid block header %s: %w", blockHash, err))
	}
	return &header
}

func (p *PreimageOracle) HeaderByBlockHash(blockHash common.Hash) eth.BlockInfo {
	return eth.HeaderBlockInfo(p.headerByBlockHash(blockHash))
}

func (p *PreimageOracle) TransactionsByBlockHash(blockHash common.Hash) (eth.BlockInfo, types.Transactions) {
	header := p.headerByBlockHash(blockHash)
	p.hint.Hint(TransactionsHint(blockHash))

	opaqueTxs := mpt.ReadTrie(header.TxHash, func(key common.Hash) []byte {
		return p.oracle.Get(preimage.Keccak256Key(key))
	})

	txs, err := eth.DecodeTransactions(opaqueTxs)
	if err != nil {
		panic(fmt.Errorf("failed to decode list of txs: %w", err))
	}

	return eth.HeaderBlockInfo(header), txs
}

func (p *PreimageOracle) ReceiptsByBlockHash(blockHash common.Hash) (eth.BlockInfo, types.Receipts) {
	info, txs := p.TransactionsByBlockHash(blockHash)

	p.hint.Hint(ReceiptsHint(blockHash))

	opaqueReceipts := mpt.ReadTrie(info.ReceiptHash(), func(key common.Hash) []byte {
		return p.oracle.Get(preimage.Keccak256Key(key))
	})

	txHashes := eth.TransactionsToHashes(txs)
	receipts, err := eth.DecodeRawReceipts(eth.ToBlockID(info), opaqueReceipts, txHashes)
	if err != nil {
		panic(fmt.Errorf("bad receipts data for block %s: %w", blockHash, err))
	}

	return info, receipts
}
