package opcm

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
)

type DeployProxy2Input struct {
	Owner common.Address
}

type DeployProxy2Output struct {
	Proxy common.Address
}

type DeployProxyScript script.DeployScriptWithOutput[DeployProxy2Input, DeployProxy2Output]

// NewDeployProxyScript loads and validates the DeployProxy2 script contract
func NewDeployProxyScript(host *script.Host) (DeployProxyScript, error) {
	return script.NewDeployScriptWithOutputFromFile[DeployProxy2Input, DeployProxy2Output](host, "DeployProxy2.s.sol", "DeployProxy2")
}
