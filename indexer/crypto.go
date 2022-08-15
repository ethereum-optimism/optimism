package indexer

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// ParseAddress parses a ETH address from a hex string. This method will
// fail if the address is not a valid hexadecimal address.
func ParseAddress(address string) (common.Address, error) {
	if common.IsHexAddress(address) {
		return common.HexToAddress(address), nil
	}
	return common.Address{}, fmt.Errorf("invalid address: %v", address)
}
