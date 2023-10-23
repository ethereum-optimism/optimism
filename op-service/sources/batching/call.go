package batching

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type BoundContract struct {
	abi  *abi.ABI
	addr common.Address
}

func NewBoundContract(abi *abi.ABI, addr common.Address) *BoundContract {
	return &BoundContract{
		abi:  abi,
		addr: addr,
	}
}

func (b *BoundContract) Call(method string, args ...interface{}) *ContractCall {
	return NewContractCall(b.abi, b.addr, method, args...)
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
		return nil, fmt.Errorf("failed to unpack data: %w", err)
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
