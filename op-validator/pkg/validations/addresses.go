package validations

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

const (
	VersionV180 = "v1.8.0"
	VersionV200 = "v2.0.0"
	VersionV300 = "v3.0.0"
)

var addresses = map[uint64]map[string]common.Address{
	1: {
		// Bootstrapped on 03/07/2025 using OP Deployer.
		VersionV180: common.HexToAddress("0x37fb5b21750d0e08a992350574bd1c24f4bcedf9"),
		// Bootstrapped on 03/07/2025 using OP Deployer.
		VersionV200: common.HexToAddress("0x12a9e38628e5a5b24d18b1956ed68a24fe4e3dc0"),
		// Bootstrapped on 03/10/2025 using OP Deployer.
		VersionV300: common.HexToAddress("0x1b8f0e76bb49b60dffbe3b847e25d6429ebff3e6"),
	},
	11155111: {
		// Bootstrapped on 03/02/2025 using OP Deployer.
		VersionV180: common.HexToAddress("0x0a5bf8ebb4b177b2dcc6eba933db726a2e2e2b4d"),
		// Bootstrapped on 03/02/2025 using OP Deployer.
		VersionV200: common.HexToAddress("0x37739a6b0a3f1e7429499a4ec4a0685439daff5c"),
		// Bootstrapped on 03/10/2025 using OP Deployer.
		VersionV300: common.HexToAddress("0x5fd6dc82a0cbc0db40d18963d812487772113216"),
	},
}

func ValidatorAddress(chainID uint64, version string) (common.Address, error) {
	chainAddresses, ok := addresses[chainID]
	if !ok {
		return common.Address{}, fmt.Errorf("unsupported chain ID: %d", chainID)
	}

	address, ok := chainAddresses[version]
	if !ok {
		return common.Address{}, fmt.Errorf("unsupported version: %s", version)
	}
	return address, nil
}
