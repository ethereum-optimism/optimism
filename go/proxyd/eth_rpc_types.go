package proxyd

import (
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

func ptr[T any](n T) *T {
	return &n
}

// TODO: also add in debug methods
var argTypes = map[string][]reflect.Type{
	// PublicEthereumAPI
	"eth_gasPrice": []reflect.Type{
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_maxPriorityFeePerGas": []reflect.Type{
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_feeHistory": []reflect.Type{
		reflect.TypeOf(rpc.DecimalOrHex(0)),
		reflect.TypeOf(rpc.BlockNumber(0)),
		reflect.TypeOf([]float64{}),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_syncing": []reflect.Type{
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_chainId": []reflect.Type{
		reflect.TypeOf(&RequestOptions{}),
	},

	// PublicBlockChainAPI
	"eth_blockNumber": []reflect.Type{
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_getBalance": []reflect.Type{
		reflect.TypeOf(common.Address{}),
		reflect.TypeOf(rpc.BlockNumberOrHash{}),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_getProof": []reflect.Type{
		reflect.TypeOf(common.Address{}),
		reflect.TypeOf([]string{}),
		reflect.TypeOf(rpc.BlockNumberOrHash{}),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_getHeaderByNumber": []reflect.Type{
		reflect.TypeOf(rpc.BlockNumber(0)),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_getHeaderByHash": []reflect.Type{
		reflect.TypeOf(common.Hash{}),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_getBlockByNumber": []reflect.Type{
		reflect.TypeOf(rpc.BlockNumber(0)),
		reflect.TypeOf(true),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_getBlockByHash": []reflect.Type{
		reflect.TypeOf(common.Hash{}),
		reflect.TypeOf(true),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_getUncleByBlockNumberAndIndex": []reflect.Type{
		reflect.TypeOf(rpc.BlockNumber(0)),
		reflect.TypeOf(hexutil.Uint(0)),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_getUncleByBlockHashAndIndex": []reflect.Type{
		reflect.TypeOf(common.Hash{}),
		reflect.TypeOf(hexutil.Uint(0)),
	},
	"eth_getUncleCountByBlockNumber": []reflect.Type{
		reflect.TypeOf(rpc.BlockNumber(0)),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_getUncleCountByBlockHash": []reflect.Type{},
	"eth_getCode": []reflect.Type{
		reflect.TypeOf(common.Address{}),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_getStorageAt": []reflect.Type{
		reflect.TypeOf(common.Address{}),
		reflect.TypeOf(""),
		reflect.TypeOf(rpc.BlockNumberOrHash{}),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_call": []reflect.Type{
		reflect.TypeOf(common.Address{}),
		reflect.TypeOf(TransactionArgs{}),
		reflect.TypeOf(rpc.BlockNumberOrHash{}),
		reflect.TypeOf(&StateOverride{}),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_estimateGas": []reflect.Type{
		reflect.TypeOf(TransactionArgs{}),
		reflect.TypeOf(rpc.BlockNumberOrHash{}),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_createAccessList": []reflect.Type{
		reflect.TypeOf(TransactionArgs{}),
		reflect.TypeOf(rpc.BlockNumberOrHash{}),
		reflect.TypeOf(&RequestOptions{}),
	},

	// PublicTransactionPoolAPI
	"eth_getBlockTransactionCountByNumber": []reflect.Type{
		reflect.TypeOf(rpc.BlockNumber(0)),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_getBlockTransactionCountByHash": []reflect.Type{
		reflect.TypeOf(rpc.BlockNumber(0)),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_getTransactionByBlockNumberAndIndex": []reflect.Type{
		reflect.TypeOf(rpc.BlockNumber(0)),
		reflect.TypeOf(hexutil.Uint(0)),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_getTransactionByBlockHashAndIndex": []reflect.Type{
		reflect.TypeOf(rpc.BlockNumber(0)),
		reflect.TypeOf(hexutil.Uint(0)),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_getRawTransactionByBlockNumberAndIndex": []reflect.Type{
		reflect.TypeOf(rpc.BlockNumber(0)),
		reflect.TypeOf(hexutil.Uint(0)),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_getRawTransactionByBlockHashAndIndex": []reflect.Type{
		reflect.TypeOf(common.Hash{}),
		reflect.TypeOf(hexutil.Uint(0)),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_getTransactionCount": []reflect.Type{
		reflect.TypeOf(common.Address{}),
		reflect.TypeOf(rpc.BlockNumberOrHash{}),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_getTransactionByHash": []reflect.Type{
		reflect.TypeOf(common.Hash{}),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_getRawTransactionByHash": []reflect.Type{
		reflect.TypeOf(common.Hash{}),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_getTransactionReceipt": []reflect.Type{
		reflect.TypeOf(common.Hash{}),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_sendTransaction": []reflect.Type{
		reflect.TypeOf(TransactionArgs{}),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_fillTransaction": []reflect.Type{
		reflect.TypeOf(TransactionArgs{}),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_sendRawTransaction": []reflect.Type{
		reflect.TypeOf(hexutil.Bytes{}),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_sign": []reflect.Type{
		reflect.TypeOf(common.Address{}),
		reflect.TypeOf(hexutil.Bytes{}),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_signTransaction": []reflect.Type{
		reflect.TypeOf(TransactionArgs{}),
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_pendingTransactions": []reflect.Type{
		reflect.TypeOf(&RequestOptions{}),
	},
	"eth_resend": []reflect.Type{
		reflect.TypeOf(TransactionArgs{}),
		reflect.TypeOf(&hexutil.Big{}),
		reflect.TypeOf(ptr(hexutil.Uint64(0))),
		reflect.TypeOf(&RequestOptions{}),
	},

	// net
	"net_version": []reflect.Type{
		reflect.TypeOf(&RequestOptions{}),
	},

	// TODO: fill these out
	// NewPublicTxPoolAPI
	"txpool_content":     []reflect.Type{},
	"txpool_contentFrom": []reflect.Type{},
	"txpool_status":      []reflect.Type{},
	"txpool_inspect":     []reflect.Type{},

	// TODO: debug methods
}

// These types are taken from go-ethereum/internal

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

	// Introduced by AccessListTxType transaction.
	AccessList *types.AccessList `json:"accessList,omitempty"`
	ChainID    *hexutil.Big      `json:"chainId,omitempty"`
}

// OverrideAccount indicates the overriding fields of account during the execution
// of a message call.
// Note, state and stateDiff can't be specified at the same time. If state is
// set, message execution will only use the data in the given state. Otherwise
// if statDiff is set, all diff will be applied first and then execute the call
// message.
type OverrideAccount struct {
	Nonce     *hexutil.Uint64              `json:"nonce"`
	Code      *hexutil.Bytes               `json:"code"`
	Balance   **hexutil.Big                `json:"balance"`
	State     *map[common.Hash]common.Hash `json:"state"`
	StateDiff *map[common.Hash]common.Hash `json:"stateDiff"`
}

// StateOverride is the collection of overridden accounts.
type StateOverride map[common.Address]OverrideAccount
