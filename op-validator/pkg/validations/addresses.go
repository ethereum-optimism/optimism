package validations

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"

	"github.com/ethereum/go-ethereum/common"
)

var addresses = map[uint64]map[string]common.Address{
	1: {
		// Bootstrapped on 03/07/2025 using OP Deployer.
		standard.ContractsV180Tag: common.HexToAddress("0x37fb5b21750d0e08a992350574bd1c24f4bcedf9"),
		// Bootstrapped on 03/07/2025 using OP Deployer.
		standard.ContractsV200Tag: common.HexToAddress("0x12a9e38628e5a5b24d18b1956ed68a24fe4e3dc0"),
		// Bootstrapped on 04/16/2025 using OP Deployer.
		standard.ContractsV300Tag: common.HexToAddress("0xf989Df70FB46c581ba6157Ab335c0833bA60e1f0"),
		// Bootstrapped on 06/03/2025 using OP Deployer.
		standard.ContractsV400Tag: common.HexToAddress("0x3dfc5e44043DC5998928E0b8280136b7352d3F70"),
	},
	11155111: {
		// Bootstrapped on 03/02/2025 using OP Deployer.
		standard.ContractsV180Tag: common.HexToAddress("0x0a5bf8ebb4b177b2dcc6eba933db726a2e2e2b4d"),
		// Bootstrapped on 03/02/2025 using OP Deployer.
		standard.ContractsV200Tag: common.HexToAddress("0x37739a6b0a3f1e7429499a4ec4a0685439daff5c"),
		// Bootstrapped on 04/03/2025 using OP Deployer.
		standard.ContractsV300Tag: common.HexToAddress("0x2d56022cb84ce6b961c3b4288ca36386bcd9024c"),
		// Bootstrapped on 06/03/2025 using OP Deployer.
		standard.ContractsV400Tag: common.HexToAddress("0xA8a1529547306FEC7A32a001705160f2110451aE"),
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
