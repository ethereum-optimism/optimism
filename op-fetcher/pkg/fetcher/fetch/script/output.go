package script

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen"
	"github.com/ethereum/go-ethereum/common"
)

type Addresses struct {
	interopgen.L2OpchainDeployment
	// Shared singletons
	SuperchainConfig common.Address `json:"SuperchainConfig"`
	Mips             common.Address `json:"MIPS"`
	PreimageOracle   common.Address `json:"PreimageOracle"`
	// Legacy contracts
	L2OutputOracleProxy common.Address `json:"L2OutputOracleProxy"`
}

type Roles struct {
	SystemConfigOwner      common.Address `json:"SystemConfigOwner"`
	OpChainProxyAdminOwner common.Address `json:"OpChainProxyAdminOwner"`
	Guardian               common.Address `json:"Guardian"`
	Challenger             common.Address `json:"Challenger"`
	Proposer               common.Address `json:"Proposer"`
	UnsafeBlockSigner      common.Address `json:"UnsafeBlockSigner"`
	BatchSubmitter         common.Address `json:"BatchSubmitter"`
}

type FetchChainInfoOutput struct {
	Addresses
	Roles
	FaultProofPermissioned   bool
	FaultProofPermissionless bool
	RespectedGameType        uint32
}

type ChainConfig struct {
	Addresses        Addresses        `json:"addresses"`
	Roles            Roles            `json:"roles"`
	FaultProofStatus FaultProofStatus `json:"fault_proofs"`
}

func (output *FetchChainInfoOutput) CheckOutput(input common.Address) error {
	return nil
}

// CreateChainConfig creates a nicely structured output from the flat FetchChainInfoOutput
func CreateChainConfig(output FetchChainInfoOutput) ChainConfig {
	return ChainConfig{
		Addresses: output.Addresses,
		Roles:     output.Roles,
		FaultProofStatus: FaultProofStatus{
			Permissioned:      output.FaultProofPermissioned,
			Permissionless:    output.FaultProofPermissionless,
			RespectedGameType: output.RespectedGameType,
		},
	}
}
