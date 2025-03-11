package client

import (
	"context"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/bindings"
	"github.com/ethereum-optimism/optimism/devnet-sdk/interfaces"
	"github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

type L2ToL2CrossDomainMessengerBinding struct {
	contractAddress types.Address
	client          *ethclient.Client
	binding         *bindings.L2ToL2CrossDomainMessenger
}

var _ interfaces.L2ToL2CrossDomainMessenger = (*L2ToL2CrossDomainMessengerBinding)(nil)

func (b *L2ToL2CrossDomainMessengerBinding) SendMessage(chainID types.ChainID, target common.Address, message []byte) types.ReadInvocation[*ethtypes.Transaction] {
	return &L2ToL2CrossDomainMessengerBindingSendMessageImpl{
		contract: b,
		chainID:  chainID,
		target:   target,
		message:  message,
	}
}

type L2ToL2CrossDomainMessengerBindingSendMessageImpl struct {
	contract *L2ToL2CrossDomainMessengerBinding
	chainID  types.ChainID
	target   common.Address
	message  []byte
}

func (i *L2ToL2CrossDomainMessengerBindingSendMessageImpl) Call(ctx context.Context) (*ethtypes.Transaction, error) {
	tx, err := i.contract.binding.SendMessage(nil, i.chainID, i.target, i.message)
	if err != nil {
		return &ethtypes.Transaction{}, err
	}
	return tx, nil
}
