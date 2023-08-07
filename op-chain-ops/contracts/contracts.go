package contracts

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

// Addresses represents the address values of various contracts. The values can
// be easily populated via a [cli.Context].
type Addresses struct {
	AddressManager            common.Address
	OptimismPortal            common.Address
	L1StandardBridge          common.Address
	L1CrossDomainMessenger    common.Address
	CanonicalTransactionChain common.Address
	StateCommitmentChain      common.Address
}

// NewAddresses populates an Addresses struct given a [cli.Context].
// This is useful for writing scripts that interact with smart contracts.
func NewAddresses(ctx *cli.Context) (*Addresses, error) {
	var addresses Addresses
	var err error

	addresses.AddressManager, err = parseAddress(ctx, "address-manager-address")
	if err != nil {
		return nil, err
	}
	addresses.OptimismPortal, err = parseAddress(ctx, "optimism-portal-address")
	if err != nil {
		return nil, err
	}
	addresses.L1StandardBridge, err = parseAddress(ctx, "l1-standard-bridge-address")
	if err != nil {
		return nil, err
	}
	addresses.L1CrossDomainMessenger, err = parseAddress(ctx, "l1-crossdomain-messenger-address")
	if err != nil {
		return nil, err
	}
	addresses.CanonicalTransactionChain, err = parseAddress(ctx, "canonical-transaction-chain-address")
	if err != nil {
		return nil, err
	}
	addresses.StateCommitmentChain, err = parseAddress(ctx, "state-commitment-chain-address")
	if err != nil {
		return nil, err
	}
	return &addresses, nil
}
