package opcm

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
)

var l2GenesisSpec = ScriptSpec{
	ScriptFile:      "L2Genesis.s.sol",
	ContractName:    "L2Genesis",
	ForgeScriptPath: "scripts/L2Genesis.s.sol:L2Genesis",
}

type L2GenesisInput = ScriptInput

type L2GenesisScript = ScriptWithoutOutput

func NewL2GenesisScript(host *script.Host) (L2GenesisScript, error) {
	return NewScriptWithoutOutputFromFile(host, l2GenesisSpec)
}
