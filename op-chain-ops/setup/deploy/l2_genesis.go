package deploy

import "github.com/ethereum-optimism/optimism/op-chain-ops/genesis"

// FullL2 defines how to create a full L2 genesis, no splits in work
type FullL2 struct {
	Args struct {
		genesis.L2InitializationConfig
		genesis.L1DependenciesConfig
	}

	Addresses struct{} // No dependencies in L2
}

func (cfg *FullL2) ScriptTarget() string {
	return "L2Genesis.s.sol"
}

func (cfg *FullL2) ScriptSig() string {
	return "l2Genesis"
}

func (cfg *FullL2) ScriptDependencies() []string {
	return []string{
		"L2Genesis.s.sol",
		"Predeploys.sol",
		"Proxy.sol",
		"LegacyMessagePasser.sol",
		"L1MessageSender.sol",
		"DeployerWhitelist.sol",
		"WETH.sol",
		"L2CrossDomainMessenger.sol",
		"GasPriceOracle.sol",
		"L2StandardBridge.sol",
		"SequencerFeeVault.sol",
		"OptimismMintableERC20Factory.sol",
		"L1BlockNumber.sol",
		"L2ERC721Bridge.sol",
		"L1Block.sol",
		"L2ToL1MessagePasser.sol",
		"OptimismMintableERC721Factory.sol",
		"ProxyAdmin.sol",
		"BaseFeeVault.sol",
		"L1FeeVault.sol",
		"SchemaRegistry.sol",
		"EAS.sol",
		"GovernanceToken.sol",
		"LegacyERC20ETH.sol",
		"CrossL2Inbox.sol",
		"L2ToL2CrossDomainMessenger.sol",
	}
}

func (cfg *FullL2) ScriptAddresses() any {
	return &cfg.Addresses
}

func (cfg *FullL2) ScriptArgs() any {
	return &cfg.Args
}
