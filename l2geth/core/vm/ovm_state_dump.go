package vm

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	// AbiBytesTrue represents the ABI encoding of "true" as a byte slice
	AbiBytesTrue = common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000001")

	// AbiBytesFalse represents the ABI encoding of "false" as a byte slice
	AbiBytesFalse = common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000000")

	// UsingOVM is used to enable or disable functionality necessary for the OVM.
	UsingOVM bool
	// EnableArbitraryContractDeployment is used to override the
	// deployer whitelist
	EnableArbitraryContractDeployment *bool

	// These are aliases to the pointer EnableArbitraryContractDeployment
	EnableArbitraryContractDeploymentTrue  bool = true
	EnableArbitraryContractDeploymentFalse bool = false

	WhitelistAddress     = common.HexToAddress("0x4200000000000000000000000000000000000002")
	isDeployerAllowedSig = crypto.Keccak256([]byte("isDeployerAllowed(address)"))[:4]
)

func init() {
	UsingOVM = os.Getenv("USING_OVM") == "true"
	value := os.Getenv("ROLLUP_ENABLE_ARBITRARY_CONTRACT_DEPLOYMENT")
	if value != "" {
		switch value {
		case "true":
			EnableArbitraryContractDeployment = &EnableArbitraryContractDeploymentTrue
		case "false":
			EnableArbitraryContractDeployment = &EnableArbitraryContractDeploymentFalse
		default:
			panic(fmt.Sprintf("Unknown ROLLUP_ENABLE_ARBITRARY_CONTRACT_DEPLOYMENT value: %s", value))
		}
	}
}
