package indexer

import (
	"fmt"

	l2common "github.com/ethereum-optimism/optimism/l2geth/common"
	"github.com/ethereum/go-ethereum/common"
)

// ParseL1Address parses a L1 ETH addres from a hex string. This method will
// fail if the address is not a valid hexidecimal address.
func ParseL1Address(address string) (common.Address, error) {
	if common.IsHexAddress(address) {
		return common.HexToAddress(address), nil
	}
	return common.Address{}, fmt.Errorf("invalid address: %v", address)
}

// ParseL2Address parses a L2 ETH addres from a hex string. This method will
// fail if the address is not a valid hexidecimal address.
func ParseL2Address(address string) (l2common.Address, error) {
	if l2common.IsHexAddress(address) {
		return l2common.HexToAddress(address), nil
	}
	return l2common.Address{}, fmt.Errorf("invalid address: %v", address)
}
