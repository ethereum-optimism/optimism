package script

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen"
	"github.com/ethereum/go-ethereum/common"
)

// Copied struct from superchain-registry/ops/internal/config/chain.go
type Addresses struct {
	interopgen.L2OpchainDeployment
	L2OutputOracleProxy common.Address `json:"L2OutputOracleProxy"`
	SuperchainConfig    common.Address `json:"SuperchainConfig"`
	Mips                common.Address `json:"MIPS"`
	PreimageOracle      common.Address `json:"PreimageOracle"`
}

type Roles struct {
	SystemConfigOwner common.Address `toml:"SystemConfigOwner,omitempty" json:"SystemConfigOwner,omitempty"`
	ProxyAdminOwner   common.Address `toml:"ProxyAdminOwner,omitempty" json:"ProxyAdminOwner,omitempty"`
	Guardian          common.Address `toml:"Guardian,omitempty" json:"Guardian,omitempty"`
	Challenger        common.Address `toml:"Challenger,omitempty" json:"Challenger,omitempty"`
	Proposer          common.Address `toml:"Proposer,omitempty" json:"Proposer,omitempty"`
	UnsafeBlockSigner common.Address `toml:"UnsafeBlockSigner,omitempty" json:"UnsafeBlockSigner,omitempty"`
	BatchSubmitter    common.Address `toml:"BatchSubmitter,omitempty" json:"BatchSubmitter,omitempty"`
}

type FetchChainInfoOutput struct {
	Addresses
	Roles
	FaultProofPermissioned   bool
	FaultProofPermissionless bool
	RespectedGameType        uint32
}

func (output *FetchChainInfoOutput) CheckOutput(input common.Address) error {
	return nil
}

func WriteChainConfigToFile(outputDir string, output FetchChainInfoOutput, chainName string, chainId uint64) error {
	fileData := ChainConfig{
		ChainName: chainName,
		ChainId:   chainId,
		Addresses: output.Addresses,
		Roles:     output.Roles,
		FaultProofStatus: FaultProofStatus{
			Permissioned:      output.FaultProofPermissioned,
			Permissionless:    output.FaultProofPermissionless,
			RespectedGameType: output.RespectedGameType,
		},
	}

	json, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outputFile := filepath.Join(outputDir, fmt.Sprintf("%d.json", chainId))
	err = os.WriteFile(outputFile, json, 0644)
	if err != nil {
		return fmt.Errorf("failed to write output to file: %w", err)
	}

	return nil
}
