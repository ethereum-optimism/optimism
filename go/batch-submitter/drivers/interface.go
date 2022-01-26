package drivers

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// L1Client is an abstraction over an L1 Ethereum client functionality required
// by the batch submitter.
type L1Client interface {
	// HeaderByNumber returns a block header from the current canonical chain.
	// If number is nil, the latest known header is returned.
	HeaderByNumber(context.Context, *big.Int) (*types.Header, error)

	// NonceAt returns the account nonce of the given account. The block number
	// can be nil, in which case the nonce is taken from the latest known block.
	NonceAt(context.Context, common.Address, *big.Int) (uint64, error)

	// SendTransaction injects a signed transaction into the pending pool for
	// execution.
	//
	// If the transaction was a contract creation use the TransactionReceipt
	// method to get the contract address after the transaction has been mined.
	SendTransaction(context.Context, *types.Transaction) error

	// SuggestGasTipCap retrieves the currently suggested gas tip cap after 1559
	// to allow a timely execution of a transaction.
	SuggestGasTipCap(context.Context) (*big.Int, error)

	// TransactionReceipt returns the receipt of a transaction by transaction
	// hash. Note that the receipt is not available for pending transactions.
	TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error)
}
