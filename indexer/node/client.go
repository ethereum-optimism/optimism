package node

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/client"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

// HeadersByRange will retrieve block headers within the specified range -- inclusive. No restrictions
// are placed on the range such as blocks in the "latest", "safe" or "finalized" states. If the specified
// range is too large, `endHeight > latest`, the resulting list is truncated to the available headers
func HeadersByRange(ctx context.Context, c client.Client, startHeight, endHeight *big.Int) ([]types.Header, error) {
	if startHeight.Cmp(endHeight) == 0 {
		header, err := c.HeaderByNumber(ctx, startHeight)
		if err != nil {
			return nil, err
		}
		return []types.Header{*header}, nil
	}

	// Batch the header requests
	rpcElems := makeHeaderRpcElems(startHeight, endHeight)
	if err := c.RPC().BatchCallContext(ctx, rpcElems); err != nil {
		return nil, err
	}

	// Parse the headers.
	//  - Ensure integrity that they build on top of each other
	//  - Truncate out headers that do not exist (endHeight > "latest")
	headers := make([]types.Header, 0, len(rpcElems))
	for i, rpcElem := range rpcElems {
		if rpcElem.Error != nil {
			if len(headers) == 0 {
				return nil, rpcElem.Error // no headers
			} else {
				break // try return whatever headers are available
			}
		} else if rpcElem.Result == nil {
			break
		}

		header := (rpcElem.Result).(*types.Header)
		if i > 0 {
			prevHeader := (rpcElems[i-1].Result).(*types.Header)
			if header.ParentHash != prevHeader.Hash() {
				return nil, fmt.Errorf("queried header %s does not follow parent %s", header.Hash(), prevHeader.Hash())
			}
		}

		headers = append(headers, *header)
	}

	return headers, nil
}

// StorageHash returns the sha3 of the storage root for the specified account
func StorageHash(ctx context.Context, c client.Client, address common.Address, blockNumber *big.Int) (common.Hash, error) {
	proof := struct{ StorageHash common.Hash }{}
	err := c.RPC().CallContext(ctx, &proof, "eth_getProof", address, nil, toBlockNumArg(blockNumber))
	if err != nil {
		return common.Hash{}, err
	}

	return proof.StorageHash, nil
}

type Logs struct {
	Logs          []types.Log
	ToBlockHeader *types.Header
}

// FilterLogsSafe returns logs that fit the query parameters. The underlying request is a batch
// request including `eth_getBlockByNumber` to allow the caller to check that connected
// node has the state necessary to fulfill this request
func FilterLogsSafe(ctx context.Context, c client.Client, query ethereum.FilterQuery) (Logs, error) {
	arg, err := toFilterArg(query)
	if err != nil {
		return Logs{}, err
	}

	var logs []types.Log
	var header types.Header

	batchElems := make([]rpc.BatchElem, 2)
	batchElems[0] = rpc.BatchElem{Method: "eth_getBlockByNumber", Args: []interface{}{toBlockNumArg(query.ToBlock), false}, Result: &header}
	batchElems[1] = rpc.BatchElem{Method: "eth_getLogs", Args: []interface{}{arg}, Result: &logs}

	if err := c.RPC().BatchCallContext(ctx, batchElems); err != nil {
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

func makeHeaderRpcElems(startHeight, endHeight *big.Int) []rpc.BatchElem {
	count := new(big.Int).Sub(endHeight, startHeight).Uint64() + 1
	batchElems := make([]rpc.BatchElem, count)
	for i := uint64(0); i < count; i++ {
		height := new(big.Int).Add(startHeight, new(big.Int).SetUint64(i))
		batchElems[i] = rpc.BatchElem{
			Method: "eth_getBlockByNumber",
			Args:   []interface{}{toBlockNumArg(height), false},
			Result: new(types.Header),
		}
	}
	return batchElems
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
