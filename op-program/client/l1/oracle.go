package l1

import (
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Oracle interface {
	// HeaderByHash retrieves the block header with the given hash.
	HeaderByHash(blockHash common.Hash) eth.BlockInfo

	// TransactionsByHash retrieves the transactions from the block with the given hash.
	TransactionsByHash(blockHash common.Hash) (eth.BlockInfo, types.Transactions)

	// ReceiptsByHash retrieves the receipts from the block with the given hash.
	ReceiptsByHash(blockHash common.Hash) (eth.BlockInfo, types.Receipts)
}
