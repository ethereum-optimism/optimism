package signer

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// TransactionArgs represents the arguments to construct a new transaction
// or a message call.
type TransactionArgs struct {
	From                 *common.Address `json:"from"`
	To                   *common.Address `json:"to"`
	Gas                  *hexutil.Uint64 `json:"gas"`
	GasPrice             *hexutil.Big    `json:"gasPrice"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas"`
	Value                *hexutil.Big    `json:"value"`
	Nonce                *hexutil.Uint64 `json:"nonce"`

	// We accept "data" and "input" for backwards-compatibility reasons.
	// "input" is the newer name and should be preferred by clients.
	// Issue detail: https://github.com/ethereum/go-ethereum/issues/15628
	Data  *hexutil.Bytes `json:"data"`
	Input *hexutil.Bytes `json:"input"`

	AccessList *types.AccessList `json:"accessList,omitempty"`
	ChainID    *hexutil.Big      `json:"chainId,omitempty"`
}

// NewTransactionArgsFromTransaction creates a TransactionArgs struct from an EIP-1559 transaction
func NewTransactionArgsFromTransaction(chainId *big.Int, from common.Address, tx *types.Transaction) *TransactionArgs {
	data := hexutil.Bytes(tx.Data())
	nonce := hexutil.Uint64(tx.Nonce())
	gas := hexutil.Uint64(tx.Gas())
	accesses := tx.AccessList()
	args := &TransactionArgs{
		From:                 &from,
		Input:                &data,
		Nonce:                &nonce,
		Value:                (*hexutil.Big)(tx.Value()),
		Gas:                  &gas,
		To:                   tx.To(),
		ChainID:              (*hexutil.Big)(chainId),
		MaxFeePerGas:         (*hexutil.Big)(tx.GasFeeCap()),
		MaxPriorityFeePerGas: (*hexutil.Big)(tx.GasTipCap()),
		AccessList:           &accesses,
	}
	return args
}

// data retrieves the transaction calldata. Input field is preferred.
func (args *TransactionArgs) data() []byte {
	if args.Input != nil {
		return *args.Input
	}
	if args.Data != nil {
		return *args.Data
	}
	return nil
}

// ToTransaction converts the arguments to a transaction.
func (args *TransactionArgs) ToTransaction() *types.Transaction {
	var data types.TxData
	al := types.AccessList{}
	if args.AccessList != nil {
		al = *args.AccessList
	}
	data = &types.DynamicFeeTx{
		To:         args.To,
		ChainID:    (*big.Int)(args.ChainID),
		Nonce:      uint64(*args.Nonce),
		Gas:        uint64(*args.Gas),
		GasFeeCap:  (*big.Int)(args.MaxFeePerGas),
		GasTipCap:  (*big.Int)(args.MaxPriorityFeePerGas),
		Value:      (*big.Int)(args.Value),
		Data:       args.data(),
		AccessList: al,
	}
	return types.NewTx(data)
}
