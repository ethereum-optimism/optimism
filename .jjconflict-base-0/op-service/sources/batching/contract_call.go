package batching

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

type ContractCall struct {
	Abi    *abi.ABI
	Addr   common.Address
	Method string
	Args   []interface{}
	From   common.Address
}

func NewContractCall(abi *abi.ABI, addr common.Address, method string, args ...interface{}) *ContractCall {
	return &ContractCall{
		Abi:    abi,
		Addr:   addr,
		Method: method,
		Args:   args,
	}
}

func (c *ContractCall) Pack() ([]byte, error) {
	return c.Abi.Pack(c.Method, c.Args...)
}

func (c *ContractCall) CallMethod() string {
	return "eth_call"
}

func (c *ContractCall) ToBatchElemCreator() (BatchElementCreator, error) {
	args, err := c.ToCallArgs()
	if err != nil {
		return nil, err
	}
	f := func(block rpcblock.Block) (any, rpc.BatchElem) {
		out := new(hexutil.Bytes)
		return out, rpc.BatchElem{
			Method: "eth_call",
			Args:   []interface{}{args, block.ArgValue()},
			Result: &out,
		}
	}
	return f, nil
}

func (c *ContractCall) ToCallArgs() (interface{}, error) {
	data, err := c.Pack()
	if err != nil {
		return nil, fmt.Errorf("failed to pack arguments: %w", err)
	}

	arg := map[string]interface{}{
		"from":  c.From,
		"to":    &c.Addr,
		"input": hexutil.Bytes(data),
	}
	return arg, nil
}

func (c *ContractCall) CreateResult() interface{} {
	return new(hexutil.Bytes)
}

func (c *ContractCall) HandleResult(result interface{}) (*CallResult, error) {
	out, err := c.Unpack(*result.(*hexutil.Bytes))
	return out, err
}

func (c *ContractCall) Unpack(hex hexutil.Bytes) (*CallResult, error) {
	out, err := c.Abi.Unpack(c.Method, hex)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack data: %w", err)
	}
	return &CallResult{out: out}, nil
}

func (c *ContractCall) ToTxCandidate() (txmgr.TxCandidate, error) {
	data, err := c.Pack()
	if err != nil {
		return txmgr.TxCandidate{}, fmt.Errorf("failed to pack arguments: %w", err)
	}
	return txmgr.TxCandidate{
		TxData: data,
		To:     &c.Addr,
	}, nil
}
