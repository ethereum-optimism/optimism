package batching

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	ErrUnknownMethod = errors.New("unknown method")
	ErrInvalidCall   = errors.New("invalid call")
	ErrUnknownEvent  = errors.New("unknown event")
	ErrInvalidEvent  = errors.New("invalid event")
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
		return "", nil, fmt.Errorf("%w: %w", ErrUnknownMethod, err)
	}
	args, err := method.Inputs.Unpack(data[4:])
	if err != nil {
		return "", nil, fmt.Errorf("%w: %w", ErrInvalidCall, err)
	}
	return method.Name, &CallResult{args}, nil
}

func (b *BoundContract) DecodeEvent(log *types.Log) (string, *CallResult, error) {
	if len(log.Topics) == 0 {
		return "", nil, ErrUnknownEvent
	}
	event, err := b.abi.EventByID(log.Topics[0])
	if err != nil {
		return "", nil, fmt.Errorf("%w: %w", ErrUnknownEvent, err)
	}

	argsMap := make(map[string]interface{})
	var indexed abi.Arguments
	for _, arg := range event.Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if err := abi.ParseTopicsIntoMap(argsMap, indexed, log.Topics[1:]); err != nil {
		return "", nil, fmt.Errorf("%w indexed topics: %w", ErrInvalidEvent, err)
	}

	nonIndexed := event.Inputs.NonIndexed()
	if len(nonIndexed) > 0 {
		if err := nonIndexed.UnpackIntoMap(argsMap, log.Data); err != nil {
			return "", nil, fmt.Errorf("%w non-indexed topics: %w", ErrInvalidEvent, err)
		}
	}
	args := make([]interface{}, 0, len(event.Inputs))
	for _, input := range event.Inputs {
		val, ok := argsMap[input.Name]
		if !ok {
			return "", nil, fmt.Errorf("%w missing argument: %v", ErrUnknownEvent, input.Name)
		}
		args = append(args, val)
	}
	return event.Name, &CallResult{args}, nil
}
