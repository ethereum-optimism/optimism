package l1

import (
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Oracle interface {
	// HeaderByBlockHash retrieves the block header with the given hash.
	HeaderByBlockHash(blockHash common.Hash) eth.BlockInfo

	// TransactionsByBlockHash retrieves the transactions from the block with the given hash.
	TransactionsByBlockHash(blockHash common.Hash) (eth.BlockInfo, types.Transactions)

	// ReceiptsByBlockHash retrieves the receipts from the block with the given hash.
	ReceiptsByBlockHash(blockHash common.Hash) (eth.BlockInfo, types.Receipts)
}
