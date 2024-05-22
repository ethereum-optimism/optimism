package misc

import (
	"github.com/ethereum-optimism/superchain-registry/superchain"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// The original create2deployer contract could not be deployed to Base mainnet at
// the canonical address of 0x13b0D85CcB8bf860b6b79AF3029fCA081AE9beF2 due to
// an accidental nonce increment from a deposit transaction. See
// https://github.com/pcaversaccio/create2deployer/issues/128 for context. This
// file applies the contract code to the canonical address manually in the Canyon
// hardfork.

var create2DeployerAddress = common.HexToAddress("0x13b0D85CcB8bf860b6b79AF3029fCA081AE9beF2")
var create2DeployerCodeHash = common.HexToHash("0xb0550b5b431e30d38000efb7107aaa0ade03d48a7198a140edda9d27134468b2")
var create2DeployerCode []byte

func init() {
	code, err := superchain.LoadContractBytecode(superchain.Hash(create2DeployerCodeHash))
	if err != nil {
		panic(err)
	}
	create2DeployerCode = code
}

func EnsureCreate2Deployer(c *params.ChainConfig, timestamp uint64, db vm.StateDB) {
	if !c.IsOptimism() || c.CanyonTime == nil || *c.CanyonTime != timestamp {
		return
	}
	log.Info("Setting Create2Deployer code", "address", create2DeployerAddress, "codeHash", create2DeployerCodeHash)
	db.SetCode(create2DeployerAddress, create2DeployerCode)
}
