package l1contracts

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/optimisticben/optimism/go/l2geth-exporter/bindings"
)

// OVMCTC interacts with the OVM CTC contract
type OVMCTC struct {
	Ctx     context.Context
	Address common.Address
	Client  *ethclient.Client
}

func (ovmctc *OVMCTC) GetTotalElements() (*big.Int, error) {

	contract, err := bindings.NewOVMCanonicalTransactionChainCaller(ovmctc.Address, ovmctc.Client)
	if err != nil {
		return nil, err
	}

	totalElements, err := contract.GetTotalElements(&bind.CallOpts{
		Context: ovmctc.Ctx,
	})
	if err != nil {
		return nil, err
	}

	return totalElements, nil

}
