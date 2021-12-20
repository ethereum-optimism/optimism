package l1contracts

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/go/l2geth-exporter/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// CTC interacts with the OVM CTC contract
type CTC struct {
	Ctx     context.Context
	Address common.Address
	Client  *ethclient.Client
}

func (ctc *CTC) GetTotalElements() (*big.Int, error) {

	contract, err := bindings.NewCanonicalTransactionChainCaller(ctc.Address, ctc.Client)
	if err != nil {
		return nil, err
	}

	totalElements, err := contract.GetTotalElements(&bind.CallOpts{
		Context: ctc.Ctx,
	})
	if err != nil {
		return nil, err
	}

	return totalElements, nil

}
