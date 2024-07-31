package deploy

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
)

// Superchain defines how to deploy superchain contracts to L1
type Superchain struct {
	Args struct {
		genesis.SuperchainL1DeployConfig
	}

	Addresses struct {
		ProxyAdmin common.Address `json:"ProxyAdmin"`
	}
}

func (cfg *Superchain) ScriptTarget() string {
	return "Deploy.s.sol"
}

func (cfg *Superchain) ScriptSig() string {
	return "deploySuperchain"
}

func (cfg *Superchain) ScriptDependencies() []string {
	return []string{"SuperchainConfig.sol", "ProtocolVersions.sol"}
}

func (cfg *Superchain) ScriptAddresses() any {
	return &cfg.Addresses
}

func (cfg *Superchain) ScriptArgs() any {
	return &cfg.Args
}
