package contracts

import (
	"context"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

type EthRpc interface {
	CallContext(ctx context.Context, out interface{}, method string, args ...interface{}) error
	BatchCallContext(ctx context.Context, b []rpc.BatchElem) error
}

type ContractCall struct {
	Abi    *abi.ABI
	Addr   common.Address
	Method string
	Args   []interface{}
}

func NewContractCall(abi *abi.ABI, addr common.Address, method string, args ...interface{}) *ContractCall {
	return &ContractCall{
		Abi:    abi,
		Addr:   addr,
		Method: method,
		Args:   args,
	}
}

func (c *ContractCall) ToCallArgs() (interface{}, error) {
	data, err := c.Abi.Pack(c.Method, c.Args...)
	if err != nil {
		return nil, fmt.Errorf("failed to pack arguments: %w", err)
	}
	msg := ethereum.CallMsg{
		To:   &c.Addr,
		Data: data,
	}
	return toCallArg(msg), nil
}

func (c *ContractCall) Unpack(hex hexutil.Bytes) ([]interface{}, error) {
	out, err := c.Abi.Unpack(c.Method, hex)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack claim data: %w", err)
	}
	return out, nil
}

type CallResult struct {
	out []interface{}
}

func (c *CallResult) GetUint8(i int) uint8 {
	return *abi.ConvertType(c.out[i], new(uint8)).(*uint8)
}

func (c *CallResult) GetUint32(i int) uint32 {
	return *abi.ConvertType(c.out[i], new(uint32)).(*uint32)
}

func (c *CallResult) GetBool(i int) bool {
	return *abi.ConvertType(c.out[i], new(bool)).(*bool)
}

func (c *CallResult) GetHash(i int) common.Hash {
	return *abi.ConvertType(c.out[i], new([32]byte)).(*[32]byte)
}

func (c *CallResult) GetBigInt(i int) *big.Int {
	return *abi.ConvertType(c.out[i], new(*big.Int)).(**big.Int)
}

type MultiCaller struct {
	rpc       EthRpc
	batchSize int
}

func NewMultiCaller(rpc EthRpc, batchSize int) *MultiCaller {
	return &MultiCaller{
		rpc:       rpc,
		batchSize: batchSize,
	}
}

func (m *MultiCaller) SingleCallLatest(ctx context.Context, call *ContractCall) (*CallResult, error) {
	results, err := m.CallLatest(ctx, call)
	if err != nil {
		return nil, err
	}
	return results[0], nil
}

func (m *MultiCaller) CallLatest(ctx context.Context, calls ...*ContractCall) ([]*CallResult, error) {
	keys := make([]interface{}, len(calls))
	for i := 0; i < len(calls); i++ {
		args, err := calls[i].ToCallArgs()
		if err != nil {
			return nil, err
		}
		keys[i] = args
	}
	fetcher := sources.NewIterativeBatchCall[interface{}, *hexutil.Bytes](
		keys,
		func(args interface{}) (*hexutil.Bytes, rpc.BatchElem) {
			out := new(hexutil.Bytes)
			return out, rpc.BatchElem{
				Method: "eth_call",
				Args:   []interface{}{args, "latest"},
				Result: &out,
			}
		},
		m.rpc.BatchCallContext,
		m.rpc.CallContext,
		m.batchSize)
	for {
		if err := fetcher.Fetch(ctx); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to fetch claims: %w", err)
		}
	}
	results, err := fetcher.Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get batch call results: %w", err)
	}

	callResults := make([]*CallResult, len(results))
	for i, result := range results {
		call := calls[i]
		out, err := call.Unpack(*result)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack result: %w", err)
		}
		callResults[i] = &CallResult{
			out: out,
		}
	}
	return callResults, nil
}

func toCallArg(msg ethereum.CallMsg) interface{} {
	arg := map[string]interface{}{
		"from": msg.From,
		"to":   msg.To,
	}
	if len(msg.Data) > 0 {
		arg["data"] = hexutil.Bytes(msg.Data)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	return arg
}
