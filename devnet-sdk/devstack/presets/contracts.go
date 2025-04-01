package presets

import "github.com/ethereum-optimism/optimism/devnet-sdk/devstack/sysgo"

func contractPaths() sysgo.ContractPaths {
	return sysgo.ContractPaths{
		FoundryArtifacts: "../../../packages/contracts-bedrock/forge-artifacts",
		SourceMap:        "../../../packages/contracts-bedrock",
	}
}
