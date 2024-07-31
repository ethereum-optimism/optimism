package deploy

import "github.com/ethereum-optimism/optimism/op-chain-ops/genesis"

// L1ContractsOfL2 defines how to deploy the contracts to L1 for a L2
type L1ContractsOfL2 struct {
	Args struct {
		genesis.L2InitializationConfig
	}

	Addresses struct {
		// TODO superchain implementation addresses etc.
	}
}

func (cfg *L1ContractsOfL2) ScriptTarget() string {
	return "Deploy.s.sol"
}

func (cfg *L1ContractsOfL2) ScriptSig() string {
	return "deployOPchain"
}

func (cfg *L1ContractsOfL2) ScriptDependencies() []string {
	return []string{
		"OptimismPortal.sol",
		// TODO
	}
}

func (cfg *L1ContractsOfL2) ScriptAddresses() any {
	return &cfg.Addresses
}

func (cfg *L1ContractsOfL2) ScriptArgs() any {
	return &cfg.Args
}
