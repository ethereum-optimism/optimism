package script

import (
	"math"

	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
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

type FaultProofStatus struct {
	Permissioned      bool   `toml:"permissioned" json:"permissioned"`
	Permissionless    bool   `toml:"permissionless" json:"permissionless"`
	RespectedGameType uint32 `toml:"respected_game_type" json:"respectedGameType"`
}

type FetchChainInfoInput struct {
	SystemConfigProxy     common.Address
	L1StandardBridgeProxy common.Address
}

type FetchChainInfoOutput struct {
	Addresses
	Roles
	FaultProofStatus
}

func (output *FetchChainInfoOutput) CheckOutput(input common.Address) error {
	return nil
}

type ChainConfig struct {
	Addresses        Addresses         `json:"addresses"`
	Roles            Roles             `json:"roles"`
	FaultProofStatus *FaultProofStatus `json:"faultProofs,omitempty" toml:"fault_proofs,omitempty"`
}

// CreateChainConfig creates a nicely structured output from the flat FetchChainInfoOutput
func CreateChainConfig(output FetchChainInfoOutput) ChainConfig {
	chain := ChainConfig{
		Addresses: output.Addresses,
		Roles:     output.Roles,
	}

	if output.FaultProofStatus.RespectedGameType == math.MaxUint32 {
		chain.FaultProofStatus = nil
	} else {
		chain.FaultProofStatus = &output.FaultProofStatus
	}
	return chain
}

func FetchChainInfo(h *script.Host, input FetchChainInfoInput) (FetchChainInfoOutput, error) {
	return opcm.RunScriptSingle[FetchChainInfoInput, FetchChainInfoOutput](h, input, "FetchChainInfo.s.sol", "FetchChainInfo")
}
