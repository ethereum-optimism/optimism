package batching

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrUnknownMethod = errors.New("unknown method")
	ErrInvalidCall   = errors.New("invalid call")
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

func (b *BoundContract) Addr() common.Address {
	return b.addr
}

func (b *BoundContract) Call(method string, args ...interface{}) *ContractCall {
	return NewContractCall(b.abi, b.addr, method, args...)
}

func (b *BoundContract) DecodeCall(data []byte) (string, *CallResult, error) {
	if len(data) < 4 {
		return "", nil, ErrUnknownMethod
	}
	method, err := b.abi.MethodById(data[:4])
	if err != nil {
		// ABI doesn't return a nicely typed error so treat any failure to find the method as unknown
		return "", nil, fmt.Errorf("%w: %v", ErrUnknownMethod, err.Error())
	}
	args, err := method.Inputs.Unpack(data[4:])
	if err != nil {
		return "", nil, fmt.Errorf("%w: %v", ErrInvalidCall, err.Error())
	}
	return method.Name, &CallResult{args}, nil
}
