package txintent

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/lmittmann/w3"
)

var _ Call = (*MultiTrigger)(nil)

// Trigger for using the MultiCall3 to batch Calls
type MultiTrigger struct {
	Emitter common.Address // address of the MultiCall3 contract
	Calls   []Call
}

func (m *MultiTrigger) To() (*common.Address, error) {
	return &m.Emitter, nil
}

func (v *MultiTrigger) EncodeInput() ([]byte, error) {
	type Call3Value struct {
		Target       common.Address
		AllowFailure bool
		CallData     []byte
	}
	var multicall []Call3Value
	for _, call := range v.Calls {
		target, err := call.To()
		if err != nil {
			return nil, fmt.Errorf("failed to aggregate to: %w", err)
		}
		calldata, err := call.EncodeInput()
		if err != nil {
			return nil, fmt.Errorf("failed to aggregate calldata: %w", err)
		}
		multicall = append(multicall, Call3Value{
			Target:       *target,
			AllowFailure: false,
			CallData:     calldata,
		})
	}
	// TODO(15005): Need to do better construct call input than this
	aggregate3 := w3.MustNewFunc("aggregate3((address target, bool allowFailure, bytes callData)[])", "(bool, bytes)[]")
	calldata, err := aggregate3.EncodeArgs(multicall)
	if err != nil {
		return nil, fmt.Errorf("failed to construct calldata: %w", err)
	}
	return calldata, nil
}

func (v *MultiTrigger) AccessList() (types.AccessList, error) {
	var aggAccessList types.AccessList
	for _, call := range v.Calls {
		accessList, err := call.AccessList()
		if err != nil {
			return nil, fmt.Errorf("failed to aggregate access list: %w", err)
		}
		aggAccessList = append(aggAccessList, accessList...)
	}
	return aggAccessList, nil
}
