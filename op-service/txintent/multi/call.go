package txintent

import (
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var _ txintent.Call = (*MultiTrigger)(nil)

type MultiTrigger struct {
	Calls []txintent.Call
}

func (v *MultiTrigger) To() (*common.Address, error) {
	// TODO format multi-call
	return nil, nil
}

func (v *MultiTrigger) Data() ([]byte, error) {
	// TODO format multi-call
	return nil, nil
}

func (v *MultiTrigger) AccessList() (types.AccessList, error) {
	// TODO format multi-call
	return nil, nil
}
