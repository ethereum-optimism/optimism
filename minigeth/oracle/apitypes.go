//go:build !mips
// +build !mips

package oracle

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// SendTxArgs represents the arguments to submit a transaction
// This struct is identical to ethapi.TransactionArgs, except for the usage of
// common.MixedcaseAddress in From and To
type SendTxArgs struct {
	From                 common.MixedcaseAddress  `json:"from"`
	To                   *common.MixedcaseAddress `json:"to"`
	Gas                  hexutil.Uint64           `json:"gas"`
	GasPrice             *hexutil.Big             `json:"gasPrice"`
	MaxFeePerGas         *hexutil.Big             `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big             `json:"maxPriorityFeePerGas"`
	Value                hexutil.Big              `json:"value"`
	Nonce                hexutil.Uint64           `json:"nonce"`

	// We accept "data" and "input" for backwards-compatibility reasons.
	// "input" is the newer name and should be preferred by clients.
	// Issue detail: https://github.com/ethereum/go-ethereum/issues/15628
	Data  *hexutil.Bytes `json:"data"`
	Input *hexutil.Bytes `json:"input,omitempty"`

	// For non-legacy transactions
	AccessList *types.AccessList `json:"accessList,omitempty"`
	ChainID    *hexutil.Big      `json:"chainId,omitempty"`

	// Signature values
	V *hexutil.Big `json:"v" gencodec:"required"`
	R *hexutil.Big `json:"r" gencodec:"required"`
	S *hexutil.Big `json:"s" gencodec:"required"`
}

type Header struct {
	ParentHash  *common.Hash      `json:"parentHash"       gencodec:"required"`
	UncleHash   *common.Hash      `json:"sha3Uncles"       gencodec:"required"`
	Coinbase    *common.Address   `json:"miner"            gencodec:"required"`
	Root        *common.Hash      `json:"stateRoot"        gencodec:"required"`
	TxHash      *common.Hash      `json:"transactionsRoot" gencodec:"required"`
	ReceiptHash *common.Hash      `json:"receiptsRoot"     gencodec:"required"`
	Bloom       *types.Bloom      `json:"logsBloom"        gencodec:"required"`
	Difficulty  *hexutil.Big      `json:"difficulty"       gencodec:"required"`
	Number      *hexutil.Big      `json:"number"           gencodec:"required"`
	GasLimit    *hexutil.Uint64   `json:"gasLimit"         gencodec:"required"`
	GasUsed     *hexutil.Uint64   `json:"gasUsed"          gencodec:"required"`
	Time        *hexutil.Uint64   `json:"timestamp"        gencodec:"required"`
	Extra       *hexutil.Bytes    `json:"extraData"        gencodec:"required"`
	MixDigest   *common.Hash      `json:"mixHash"`
	Nonce       *types.BlockNonce `json:"nonce"`
	BaseFee     *hexutil.Big      `json:"baseFeePerGas" rlp:"optional"`
	// transactions
	Transactions []SendTxArgs `json:"transactions"`
}

func (dec *Header) ToHeader() types.Header {
	var h types.Header
	h.ParentHash = *dec.ParentHash
	h.UncleHash = *dec.UncleHash
	h.Coinbase = *dec.Coinbase
	h.Root = *dec.Root
	h.TxHash = *dec.TxHash
	h.ReceiptHash = *dec.ReceiptHash
	h.Bloom = *dec.Bloom
	h.Difficulty = (*big.Int)(dec.Difficulty)
	h.Number = (*big.Int)(dec.Number)
	h.GasLimit = uint64(*dec.GasLimit)
	h.GasUsed = uint64(*dec.GasUsed)
	h.Time = uint64(*dec.Time)
	h.Extra = *dec.Extra
	if dec.MixDigest != nil {
		h.MixDigest = *dec.MixDigest
	}
	if dec.Nonce != nil {
		h.Nonce = *dec.Nonce
	}
	if dec.BaseFee != nil {
		h.BaseFee = (*big.Int)(dec.BaseFee)
	}
	return h
}

// ToTransaction converts the arguments to a transaction.
func (args *SendTxArgs) ToTransaction() *types.Transaction {
	// Add the To-field, if specified
	var to *common.Address
	if args.To != nil {
		dstAddr := args.To.Address()
		to = &dstAddr
	}

	var input []byte
	if args.Input != nil {
		input = *args.Input
	} else if args.Data != nil {
		input = *args.Data
	}

	var data types.TxData
	switch {
	case args.MaxFeePerGas != nil:
		al := types.AccessList{}
		if args.AccessList != nil {
			al = *args.AccessList
		}
		data = &types.DynamicFeeTx{
			To:         to,
			ChainID:    (*big.Int)(args.ChainID),
			Nonce:      uint64(args.Nonce),
			Gas:        uint64(args.Gas),
			GasFeeCap:  (*big.Int)(args.MaxFeePerGas),
			GasTipCap:  (*big.Int)(args.MaxPriorityFeePerGas),
			Value:      (*big.Int)(&args.Value),
			Data:       input,
			AccessList: al,
			V:          (*big.Int)(args.V),
			R:          (*big.Int)(args.R),
			S:          (*big.Int)(args.S),
		}
	case args.AccessList != nil:
		data = &types.AccessListTx{
			To:         to,
			ChainID:    (*big.Int)(args.ChainID),
			Nonce:      uint64(args.Nonce),
			Gas:        uint64(args.Gas),
			GasPrice:   (*big.Int)(args.GasPrice),
			Value:      (*big.Int)(&args.Value),
			Data:       input,
			AccessList: *args.AccessList,
			V:          (*big.Int)(args.V),
			R:          (*big.Int)(args.R),
			S:          (*big.Int)(args.S),
		}
	default:
		data = &types.LegacyTx{
			To:       to,
			Nonce:    uint64(args.Nonce),
			Gas:      uint64(args.Gas),
			GasPrice: (*big.Int)(args.GasPrice),
			Value:    (*big.Int)(&args.Value),
			Data:     input,
			V:        (*big.Int)(args.V),
			R:        (*big.Int)(args.R),
			S:        (*big.Int)(args.S),
		}
	}
	return types.NewTx(data)
}
