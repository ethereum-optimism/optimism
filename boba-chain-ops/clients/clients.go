package clients

import (
	"context"
	"fmt"
	"math/big"

	ethereum "github.com/ledgerwatch/erigon"
	libcommon "github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/common/hexutil"
	"github.com/ledgerwatch/erigon-lib/common/hexutility"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/rpc"
	"github.com/ledgerwatch/erigon/turbo/adapter/ethapi"
	"github.com/ledgerwatch/log/v3"
	"github.com/urfave/cli/v2"
)

// wrapper for rpc.Client
type RpcClient struct {
	RpcClient   *rpc.Client
	BlockNumber int64
}

func NewRpcClient(url string, blockNumber int64) (*RpcClient, error) {
	rpcClient, err := rpc.DialContext(context.Background(), url, log.New())
	if err != nil {
		return nil, fmt.Errorf("cannot dial rpc client: %w", err)
	}
	return &RpcClient{RpcClient: rpcClient, BlockNumber: blockNumber}, nil
}

func (r *RpcClient) CodeAt(ctx context.Context, contract libcommon.Address, blockNumber *big.Int) ([]byte, error) {
	if blockNumber == nil {
		blockNumber = big.NewInt(r.BlockNumber)
	}
	var result hexutility.Bytes
	if err := r.RpcClient.CallContext(ctx, &result, "eth_getCode", contract, hexutil.EncodeUint64(blockNumber.Uint64())); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *RpcClient) PendingCodeAt(ctx context.Context, contract libcommon.Address) ([]byte, error) {
	return []byte{}, fmt.Errorf("not implemented")
}

func (r *RpcClient) PendingNonceAt(ctx context.Context, account libcommon.Address) (uint64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (r *RpcClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return big.NewInt(0), fmt.Errorf("not implemented")
}

func (r *RpcClient) EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (r *RpcClient) SendTransaction(ctx context.Context, tx types.Transaction) error {
	return fmt.Errorf("not implemented")
}

func (r *RpcClient) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	return []types.Log{}, fmt.Errorf("not implemented")
}

func (r *RpcClient) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *RpcClient) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	if blockNumber == nil {
		blockNumber = big.NewInt(r.BlockNumber)
	}
	data := hexutility.Bytes(call.Data)
	calldata := ethapi.CallArgs{
		From: &call.From,
		To:   call.To,
		Data: &data,
	}
	var result hexutility.Bytes
	if err := r.RpcClient.CallContext(ctx, &result, "eth_call", calldata, hexutil.EncodeUint64(blockNumber.Uint64())); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *RpcClient) StorageAt(ctx context.Context, address libcommon.Address, index libcommon.Hash, blockNumber *big.Int) ([]byte, error) {
	if blockNumber == nil {
		blockNumber = big.NewInt(r.BlockNumber)
	}
	var (
		empty  []byte
		result hexutility.Bytes
	)
	if err := r.RpcClient.CallContext(ctx, &result, "eth_getStorageAt", address, index, hexutil.EncodeUint64(blockNumber.Uint64())); err != nil {
		return common.LeftPadBytes(empty, 32), err
	}
	return common.LeftPadBytes(result, 32), nil
}

// clients represents a set of initialized RPC clients
type Clients struct {
	L1RpcClient *RpcClient
	L2RpcClient *RpcClient
}

// NewClients will create new RPC clients from a CLI context
func NewClients(ctx *cli.Context) (*Clients, error) {
	clients := Clients{}
	blockNumber := ctx.Int64("l2-block-number")

	if l1RpcURL := ctx.String("l1-rpc-url"); l1RpcURL != "" {
		l1RpcClient, err := NewRpcClient(l1RpcURL, blockNumber)
		if err != nil {
			return nil, err
		}
		clients.L1RpcClient = l1RpcClient
	}

	if l2RpcURL := ctx.String("l2-rpc-url"); l2RpcURL != "" {
		l2RpcClient, err := NewRpcClient(l2RpcURL, blockNumber)
		if err != nil {
			return nil, err
		}
		clients.L2RpcClient = l2RpcClient
	}

	return &clients, nil
}
