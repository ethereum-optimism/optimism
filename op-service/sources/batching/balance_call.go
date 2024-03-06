package batching

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

type BalanceCall struct {
	addr common.Address
}

var _ Call = (*BalanceCall)(nil)

func NewBalanceCall(addr common.Address) *BalanceCall {
	return &BalanceCall{addr}
}

func (b *BalanceCall) ToBatchElemCreator() (BatchElementCreator, error) {
	return func(block Block) (any, rpc.BatchElem) {
		out := new(hexutil.Big)
		return out, rpc.BatchElem{
			Method: "eth_getBalance",
			Args:   []interface{}{b.addr, block.value},
			Result: &out,
		}
	}, nil
}

func (b *BalanceCall) HandleResult(result interface{}) (*CallResult, error) {
	//balance := (*big.Int)(&result)
	return &CallResult{out: []interface{}{result}}, nil
}
