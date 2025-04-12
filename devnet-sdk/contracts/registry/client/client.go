package client

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/bindings"
	"github.com/ethereum-optimism/optimism/devnet-sdk/interfaces"
	"github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ClientRegistry is a Registry implementation that uses an ethclient.Client
type ClientRegistry struct {
	Client *ethclient.Client
}

var _ interfaces.ContractsRegistry = (*ClientRegistry)(nil)

func (r *ClientRegistry) SuperchainWETH(address types.Address) (interfaces.SuperchainWETH, error) {
	binding, err := bindings.NewSuperchainWETH(address, r.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to create SuperchainWETH binding: %w", err)
	}
	return &superchainWETHBinding{
		contractAddress: address,
		client:          r.Client,
		binding:         binding,
	}, nil
}

func (r *ClientRegistry) L2ToL2CrossDomainMessenger(address types.Address) (interfaces.L2ToL2CrossDomainMessenger, error) {
	binding, err := bindings.NewL2ToL2CrossDomainMessenger(address, r.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to create L2ToL2CrossDomainMessenger binding: %w", err)
	}
	abi, err := abi.JSON(strings.NewReader(bindings.L2ToL2CrossDomainMessengerMetaData.ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to create L2ToL2CrossDomainMessenger binding ABI: %w", err)
	}
	return &L2ToL2CrossDomainMessengerBinding{
		contractAddress: address,
		client:          r.Client,
		binding:         binding,
		abi:             &abi,
	}, nil
}
