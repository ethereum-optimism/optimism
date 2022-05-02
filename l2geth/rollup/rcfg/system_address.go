package rcfg

import (
	"math/big"
	"os"

	"github.com/ethereum-optimism/optimism/l2geth/common"
)

// SystemAddress0 is the first deployable system address.
var SystemAddress0 = common.HexToAddress("0x4200000000000000000000000000000000000042")

// SystemAddress1 is the second deployable system address.
var SystemAddress1 = common.HexToAddress("0x4200000000000000000000000000000000000014")

// ZeroSystemAddress is the emprt system address.
var ZeroSystemAddress common.Address

// SystemAddressDeployer is a tuple containing the deployment
// addresses for SystemAddress0 and SystemAddress1.
type SystemAddressDeployer [2]common.Address

// SystemAddressFor returns the system address for a given deployment
// address. If no system address is configured for this deployer,
// ZeroSystemAddress is returned.
func (s SystemAddressDeployer) SystemAddressFor(addr common.Address) common.Address {
	if s[0] == addr {
		return SystemAddress0
	}

	if s[1] == addr {
		return SystemAddress1
	}

	return ZeroSystemAddress
}

// SystemAddressFor is a convenience method that returns an environment-based
// system address if the passed-in chain ID is not hardcoded.
func SystemAddressFor(chainID *big.Int, addr common.Address) common.Address {
	sysDeployer, hasHardcodedSysDeployer := SystemAddressDeployers[chainID.Uint64()]
	if !hasHardcodedSysDeployer {
		sysDeployer = envSystemAddressDeployer
	}

	return sysDeployer.SystemAddressFor(addr)
}

// SystemAddressDeployers maintains a hardcoded map of chain IDs to
// system addresses.
var SystemAddressDeployers = map[uint64]SystemAddressDeployer{
	// Mainnet
	10: {
		common.HexToAddress("0xcDE47C1a5e2d60b9ff262b0a3b6d486048575Ad9"),
		common.HexToAddress("0x53A6eecC2dD4795Fcc68940ddc6B4d53Bd88Bd9E"),
	},

	// Kovan
	69: {
		common.HexToAddress("0xd23eb5c2dd7035e6eb4a7e129249d9843123079f"),
		common.HexToAddress("0xa81224490b9fa4930a2e920550cd1c9106bb6d9e"),
	},

	// Goerli
	420: {
		common.HexToAddress("0xc30276833798867c1dbc5c468bf51ca900b44e4c"),
		common.HexToAddress("0x5c679a57e018f5f146838138d3e032ef4913d551"),
	},

	// Goerli nightly
	421: {
		common.HexToAddress("0xc30276833798867c1dbc5c468bf51ca900b44e4c"),
		common.HexToAddress("0x5c679a57e018f5f146838138d3e032ef4913d551"),
	},
}

var envSystemAddressDeployer SystemAddressDeployer

func initEnvSystemAddressDeployer() {
	deployer0Env := os.Getenv("SYSTEM_ADDRESS_0_DEPLOYER")
	deployer1Env := os.Getenv("SYSTEM_ADDRESS_1_DEPLOYER")

	if deployer0Env == "" && deployer1Env == "" {
		return
	}
	if !common.IsHexAddress(deployer0Env) {
		panic("SYSTEM_ADDRESS_0_DEPLOYER specified but invalid")
	}
	if !common.IsHexAddress(deployer1Env) {
		panic("SYSTEM_ADDRESS_1_DEPLOYER specified but invalid")
	}
	envSystemAddressDeployer[0] = common.HexToAddress(deployer0Env)
	envSystemAddressDeployer[1] = common.HexToAddress(deployer1Env)
}

func init() {
	initEnvSystemAddressDeployer()
}
