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
		// Bootstrapped on 10/02/2025 using OP Deployer.
		standard.ContractsV410Tag: common.HexToAddress("0x845FEF377Fa9C678B3eBe33B024678538f1215dD"),
		// Bootstrapped on 10/27/2025 using OP Deployer.
		standard.ContractsV500Tag: common.HexToAddress("0xDCE1A51A25dD5BF02ccB4264D039EDdF11A95b43"),
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
		// Bootstrapped on 10/02/2025 using OP Deployer.
		standard.ContractsV410Tag: common.HexToAddress("0x7B4d2a02d5fa6C7C98D835d819956EBB876Ff439"),
		// Bootstrapped on 10/27/2025 using OP Deployer.
		standard.ContractsV500Tag: common.HexToAddress("0x757bFA3AAABcE60112Cee3239DCD05b5F6EFaE3A"),
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
