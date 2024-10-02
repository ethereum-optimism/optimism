package opcm

import (
	"embed"
	"fmt"

	"github.com/ethereum-optimism/superchain-registry/superchain"
	"github.com/ethereum/go-ethereum/common"
)

//go:embed standard-versions-mainnet.toml
var StandardVersionsMainnetData string

//go:embed standard-versions-sepolia.toml
var StandardVersionsSepoliaData string

var _ embed.FS

func StandardVersionsFor(chainID uint64) (string, error) {
	switch chainID {
	case 1:
		return StandardVersionsMainnetData, nil
	case 11155111:
		return StandardVersionsSepoliaData, nil
	default:
		return "", fmt.Errorf("unsupported chain ID: %d", chainID)
	}
}

func SuperchainFor(chainID uint64) (*superchain.Superchain, error) {
	switch chainID {
	case 1:
		return superchain.Superchains["mainnet"], nil
	case 11155111:
		return superchain.Superchains["sepolia"], nil
	default:
		return nil, fmt.Errorf("unsupported chain ID: %d", chainID)
	}
}

func ManagerImplementationAddrFor(chainID uint64) (common.Address, error) {
	switch chainID {
	case 11155111:
		// Generated using the bootstrap command on 09/26/2024.
		return common.HexToAddress("0x0dc727671d5c08e4e41e8909983ebfa6f57aa0bf"), nil
	default:
		return common.Address{}, fmt.Errorf("unsupported chain ID: %d", chainID)
	}
}

func ManagerOwnerAddrFor(chainID uint64) (common.Address, error) {
	switch chainID {
	case 1:
		// Set to superchain proxy admin
		return common.HexToAddress("0x543bA4AADBAb8f9025686Bd03993043599c6fB04"), nil
	case 11155111:
		// Set to development multisig
		return common.HexToAddress("0xDEe57160aAfCF04c34C887B5962D0a69676d3C8B"), nil
	default:
		return common.Address{}, fmt.Errorf("unsupported chain ID: %d", chainID)
	}
}
