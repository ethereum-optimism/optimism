package node

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/retry"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	// defaultDialTimeout is default duration the processor will wait on
	// startup to make a connection to the backend
	defaultDialTimeout = 5 * time.Second

	// defaultDialAttempts is the default attempts a connection will be made
	// before failing
	defaultDialAttempts = 5

	// defaultRequestTimeout is the default duration the processor will
	// wait for a request to be fulfilled
	defaultRequestTimeout = 10 * time.Second
)

type EthClient interface {
	BlockHeaderByNumber(*big.Int) (*types.Header, error)
	BlockHeaderByHash(common.Hash) (*types.Header, error)
	BlockHeadersByRange(*big.Int, *big.Int) ([]types.Header, error)

	TxByHash(common.Hash) (*types.Transaction, error)

	StorageHash(common.Address, *big.Int) (common.Hash, error)
	FilterLogs(ethereum.FilterQuery) (Logs, error)

	// Close closes the underlying RPC connection.
	// RPC close does not return any errors, but does shut down e.g. a websocket connection.
	Close()
}

type clnt struct {
	rpc RPC
}

func DialEthClient(ctx context.Context, rpcUrl string, metrics Metricer) (EthClient, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()

	bOff := retry.Exponential()
	rpcClient, err := retry.Do(ctx, defaultDialAttempts, bOff, func() (*rpc.Client, error) {
		if !client.IsURLAvailable(rpcUrl) {
			return nil, fmt.Errorf("address unavailable (%s)", rpcUrl)
		}

		client, err := rpc.DialContext(ctx, rpcUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to dial address (%s): %w", rpcUrl, err)
		}

		return client, nil
	})

	if err != nil {
		return nil, err
	}

	return &clnt{rpc: NewRPC(rpcClient, metrics)}, nil
}

// BlockHeaderByHash retrieves the block header attributed to the supplied hash
func (c *clnt) BlockHeaderByHash(hash common.Hash) (*types.Header, error) {
	ctxwt, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	var header *types.Header
	err := c.rpc.CallContext(ctxwt, &header, "eth_getBlockByHash", hash, false)
	if err != nil {
		return nil, err
	} else if header == nil {
		return nil, ethereum.NotFound
	}

	// sanity check on the data returned
	if header.Hash() != hash {
		return nil, errors.New("header mismatch")
	}

	return header, nil
}

// BlockHeaderByNumber retrieves the block header attributed to the supplied height
func (c *clnt) BlockHeaderByNumber(number *big.Int) (*types.Header, error) {
	ctxwt, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	var header *types.Header
	err := c.rpc.CallContext(ctxwt, &header, "eth_getBlockByNumber", toBlockNumArg(number), false)
	if err != nil {
		return nil, err
	} else if header == nil {
		return nil, ethereum.NotFound
	}

	return header, nil
}

// BlockHeadersByRange will retrieve block headers within the specified range -- inclusive. No restrictions
// are placed on the range such as blocks in the "latest", "safe" or "finalized" states. If the specified
// range is too large, `endHeight > latest`, the resulting list is truncated to the available headers
func (c *clnt) BlockHeadersByRange(startHeight, endHeight *big.Int) ([]types.Header, error) {
	// avoid the batch call if there's no range
	if startHeight.Cmp(endHeight) == 0 {
		header, err := c.BlockHeaderByNumber(startHeight)
		if err != nil {
			return nil, err
		}
		return []types.Header{*header}, nil
	}

	count := new(big.Int).Sub(endHeight, startHeight).Uint64() + 1
	headers := make([]types.Header, count)
	batchElems := make([]rpc.BatchElem, count)

	for i := uint64(0); i < count; i++ {
		height := new(big.Int).Add(startHeight, new(big.Int).SetUint64(i))
		batchElems[i] = rpc.BatchElem{Method: "eth_getBlockByNumber", Args: []interface{}{toBlockNumArg(height), false}, Result: &headers[i]}
	}

	ctxwt, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()
	err := c.rpc.BatchCallContext(ctxwt, batchElems)
	if err != nil {
		return nil, err
	}

	// Parse the headers.
	//  - Ensure integrity that they build on top of each other
	//  - Truncate out headers that do not exist (endHeight > "latest")
	size := 0
	for i, batchElem := range batchElems {
		if batchElem.Error != nil {
			if size == 0 {
				return nil, batchElem.Error
			} else {
				break // try return whatever headers are available
			}
		} else if batchElem.Result == nil {
			break
		}

		if i > 0 && headers[i].ParentHash != headers[i-1].Hash() {
			return nil, fmt.Errorf("queried header %s does not follow parent %s", headers[i].Hash(), headers[i-1].Hash())
		}

		size = size + 1
	}

	headers = headers[:size]
	return headers, nil
}

func (c *clnt) TxByHash(hash common.Hash) (*types.Transaction, error) {
	ctxwt, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	var tx *types.Transaction
	err := c.rpc.CallContext(ctxwt, &tx, "eth_getTransactionByHash", hash)
	if err != nil {
		return nil, err
	} else if tx == nil {
		return nil, ethereum.NotFound
	}

	return tx, nil
}

// StorageHash returns the sha3 of the storage root for the specified account
func (c *clnt) StorageHash(address common.Address, blockNumber *big.Int) (common.Hash, error) {
	ctxwt, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	proof := struct{ StorageHash common.Hash }{}
	err := c.rpc.CallContext(ctxwt, &proof, "eth_getProof", address, nil, toBlockNumArg(blockNumber))
	if err != nil {
		return common.Hash{}, err
	}

	return proof.StorageHash, nil
}

func (c *clnt) Close() {
	c.rpc.Close()
}

type Logs struct {
	Logs          []types.Log
	ToBlockHeader *types.Header
}

// FilterLogs returns logs that fit the query parameters. The underlying request is a batch
// request including `eth_getBlockByNumber` to allow the caller to check that connected
// node has the state necessary to fulfill this request
func (c *clnt) FilterLogs(query ethereum.FilterQuery) (Logs, error) {
	arg, err := toFilterArg(query)
	if err != nil {
		return Logs{}, err
	}

	var logs []types.Log
	var header types.Header

	batchElems := make([]rpc.BatchElem, 2)
	batchElems[0] = rpc.BatchElem{Method: "eth_getBlockByNumber", Args: []interface{}{toBlockNumArg(query.ToBlock), false}, Result: &header}
	batchElems[1] = rpc.BatchElem{Method: "eth_getLogs", Args: []interface{}{arg}, Result: &logs}

	ctxwt, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()
	err = c.rpc.BatchCallContext(ctxwt, batchElems)
	if err != nil {
		return Logs{}, err
	}

	if batchElems[0].Error != nil {
		return Logs{}, fmt.Errorf("unable to query for the `FilterQuery#ToBlock` header: %w", batchElems[0].Error)
	}

	if batchElems[1].Error != nil {
		return Logs{}, fmt.Errorf("unable to query logs: %w", batchElems[1].Error)
	}

	return Logs{Logs: logs, ToBlockHeader: &header}, nil
}

// Modeled off op-service/client.go. We can refactor this once the client/metrics portion
// of op-service/client has been generalized

type RPC interface {
	Close()
	CallContext(ctx context.Context, result any, method string, args ...any) error
	BatchCallContext(ctx context.Context, b []rpc.BatchElem) error
}

type rpcClient struct {
	rpc     *rpc.Client
	metrics Metricer
}

func NewRPC(client *rpc.Client, metrics Metricer) RPC {
	return &rpcClient{client, metrics}
}

func (c *rpcClient) Close() {
	c.rpc.Close()
}

func (c *rpcClient) CallContext(ctx context.Context, result any, method string, args ...any) error {
	record := c.metrics.RecordRPCClientRequest(method)
	err := c.rpc.CallContext(ctx, result, method, args...)
	record(err)
	return err
}

func (c *rpcClient) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	record := c.metrics.RecordRPCClientBatchRequest(b)
	err := c.rpc.BatchCallContext(ctx, b)
	record(err)
	return err
}

// Needed private utils from geth

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	if number.Sign() >= 0 {
		return hexutil.EncodeBig(number)
	}
	// It's negative.
	return rpc.BlockNumber(number.Int64()).String()
}

func toFilterArg(q ethereum.FilterQuery) (interface{}, error) {
	arg := map[string]interface{}{"address": q.Addresses, "topics": q.Topics}
	if q.BlockHash != nil {
		arg["blockHash"] = *q.BlockHash
		if q.FromBlock != nil || q.ToBlock != nil {
			return nil, errors.New("cannot specify both BlockHash and FromBlock/ToBlock")
		}
	} else {
		if q.FromBlock == nil {
			arg["fromBlock"] = "0x0"
		} else {
			arg["fromBlock"] = toBlockNumArg(q.FromBlock)
		}
		arg["toBlock"] = toBlockNumArg(q.ToBlock)
	}
	return arg, nil
}
