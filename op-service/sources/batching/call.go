package batching

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

type BatchElementCreator func(block rpcblock.Block) (any, rpc.BatchElem)

type Call interface {
	ToBatchElemCreator() (BatchElementCreator, error)
	HandleResult(interface{}) (*CallResult, error)
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

func (c *CallResult) GetUint64(i int) uint64 {
	return *abi.ConvertType(c.out[i], new(uint64)).(*uint64)
}

func (c *CallResult) GetBool(i int) bool {
	return *abi.ConvertType(c.out[i], new(bool)).(*bool)
}

func (c *CallResult) GetHash(i int) common.Hash {
	return *abi.ConvertType(c.out[i], new([32]byte)).(*[32]byte)
}

func (c *CallResult) GetAddress(i int) common.Address {
	return *abi.ConvertType(c.out[i], new([20]byte)).(*[20]byte)
}

func (c *CallResult) GetBigInt(i int) *big.Int {
	return *abi.ConvertType(c.out[i], new(*big.Int)).(**big.Int)
}

func (c *CallResult) GetStruct(i int, target interface{}) {
	abi.ConvertType(c.out[i], target)
}

func (c *CallResult) GetBytes(i int) []byte {
	return *abi.ConvertType(c.out[i], new([]byte)).(*[]byte)
}

func (c *CallResult) GetBytes32(i int) [32]byte {
	return *abi.ConvertType(c.out[i], new([32]byte)).(*[32]byte)
}

func (c *CallResult) GetBytes32Slice(i int) [][32]byte {
	return *abi.ConvertType(c.out[i], new([][32]byte)).(*[][32]byte)
}

func (c *CallResult) GetString(i int) string {
	return *abi.ConvertType(c.out[i], new(string)).(*string)
}
