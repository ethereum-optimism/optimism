package client

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/bindings"
	"github.com/ethereum-optimism/optimism/devnet-sdk/interfaces"
	"github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/ethclient"
)

type L2ToL2CrossDomainMessengerBinding struct {
	contractAddress types.Address
	client          *ethclient.Client
	binding         *bindings.L2ToL2CrossDomainMessenger
	abi             *abi.ABI
}

var _ interfaces.L2ToL2CrossDomainMessenger = (*L2ToL2CrossDomainMessengerBinding)(nil)

func (b *L2ToL2CrossDomainMessengerBinding) ABI() *abi.ABI {
	return b.abi
}
