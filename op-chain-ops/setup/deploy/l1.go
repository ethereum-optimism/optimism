package deploy

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
)

// DevL1 defines how to create a initial L1 chain state for dev purposes
type DevL1 struct {
	Args genesis.DevL1DeployConfig

	Addresses struct{} // No dependencies
}

func (cfg *DevL1) ScriptTarget() string {
	return "Deploy.s.sol"
}

func (cfg *DevL1) ScriptSig() string {
	return "deployDevL1"
}

func (cfg *DevL1) ScriptDependencies() []string {
	return []string{} // No default contracts in dev L1 to include.
}

func (cfg *DevL1) ScriptAddresses() any {
	return &cfg.Addresses
}

func (cfg *DevL1) ScriptArgs() any {
	return &cfg.Args
}
