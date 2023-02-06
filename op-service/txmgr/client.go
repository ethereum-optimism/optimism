package txmgr

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/l2geth/common/hexutil"
	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

// FallbackGasTipCap is the default fallback gasTipCap used when we are
// unable to query an L1 backend for a suggested gasTipCap.
var FallbackGasTipCap = big.NewInt(1_500_000_000)

var _ ETHBackend = (&MultiBackendEthClient{})

// MultiBackendEthClient implements the ETHBackend interface over
// multiple L1 RPC providers. Specifically in needs to implement
// SuggestGasTipCap on RPC providers that do not provide
// eth_maxPriorityFee (geth specific) and instead fall-back to
// a different API or a hard coded value.
type MultiBackendEthClient struct {
	c client.RPC
	// TODO: record values of what works & what doesn't
	// TODO: Optional default value or opts value?
	// TODO: Should we directly embed a gas price oracle (from geth) here?
}

// From geth
// // OracleBackend includes all necessary background APIs for oracle.
// type OracleBackend interface {
// 	HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error)
// 	BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error)
// 	GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error)
// 	PendingBlockAndReceipts() (*types.Block, types.Receipts)
// 	ChainConfig() *params.ChainConfig
// 	SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription
// }

// NewMultiBackendEthClient wraps an client.RPC that may have a differnt
// collection of fee estimation APIs.
// TODO: Add more options here
func NewMultiBackendEthClient(client client.RPC) *MultiBackendEthClient {
	return &MultiBackendEthClient{c: client}
}

// SuggestGasTipCap retrieves the currently suggested gas tip cap after 1559 to
// allow a timely execution of a transaction.
// It can handle the case that the underlying RPC connection does not provide the
// eth_maxPriorityFee method.
func (ec *MultiBackendEthClient) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	if ret, err := ec.suggestGasTipCap(ctx); err == nil {
		return ret, nil
	} else if rpcErr, ok := err.(rpc.Error); ok {
		if rpcErr.ErrorCode() != -32601 {
			return nil, err
		}
	}
	// Error was an RPC - method not found (error code -32601)
	// Go ahead & attempt a different approach to gas fee selection.
	return FallbackGasTipCap, nil
}

// suggestGasTipCap retrieves the currently suggested gas tip cap after 1559 to
// allow a timely execution of a transaction.
func (ec *MultiBackendEthClient) suggestGasTipCap(ctx context.Context) (*big.Int, error) {
	var hex hexutil.Big
	if err := ec.c.CallContext(ctx, &hex, "eth_maxPriorityFeePerGas"); err != nil {
		return nil, err
	}
	return (*big.Int)(&hex), nil
}

// BlockNumber returns the most recent block number
func (ec *MultiBackendEthClient) BlockNumber(ctx context.Context) (uint64, error) {
	var result hexutil.Uint64
	err := ec.c.CallContext(ctx, &result, "eth_blockNumber")
	return uint64(result), err
}

// HeaderByNumber returns a block header from the current canonical chain. If number is
// nil, the latest known header is returned.
func (ec *MultiBackendEthClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	var head *types.Header
	err := ec.c.CallContext(ctx, &head, "eth_getBlockByNumber", toBlockNumArg(number), false)
	if err == nil && head == nil {
		err = ethereum.NotFound
	}
	return head, err
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	pending := big.NewInt(-1)
	if number.Cmp(pending) == 0 {
		return "pending"
	}
	finalized := big.NewInt(int64(rpc.FinalizedBlockNumber))
	if number.Cmp(finalized) == 0 {
		return "finalized"
	}
	safe := big.NewInt(int64(rpc.SafeBlockNumber))
	if number.Cmp(safe) == 0 {
		return "safe"
	}
	return hexutil.EncodeBig(number)
}

// SendTransaction injects a signed transaction into the pending pool for execution.
//
// If the transaction was a contract creation use the TransactionReceipt method to get the
// contract address after the transaction has been mined.
func (ec *MultiBackendEthClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	data, err := tx.MarshalBinary()
	if err != nil {
		return err
	}
	return ec.c.CallContext(ctx, nil, "eth_sendRawTransaction", hexutil.Encode(data))
}

// TransactionReceipt returns the receipt of a transaction by transaction hash.
// Note that the receipt is not available for pending transactions.
func (ec *MultiBackendEthClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	var r *types.Receipt
	err := ec.c.CallContext(ctx, &r, "eth_getTransactionReceipt", txHash)
	if err == nil {
		if r == nil {
			return nil, ethereum.NotFound
		}
	}
	return r, err
}

func (ec *MultiBackendEthClient) Close() {
	ec.c.Close()
}
