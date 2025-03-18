package script

import (
	_ "embed"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum/go-ethereum/common"
)

type FetchChainInfoInput struct {
	SystemConfigProxy     common.Address
	L1StandardBridgeProxy common.Address
}

type ChainConfig struct {
	ChainId          uint64           `toml:"chain_id" json:"chain_id"`
	ChainName        string           `toml:"chain_name" json:"chain_name"`
	Addresses        Addresses        `toml:"addresses" json:"addresses"`
	Roles            Roles            `toml:"roles" json:"roles"`
	FaultProofStatus FaultProofStatus `toml:"fault_proof_status" json:"fault_proof_status"`
}

type FaultProofStatus struct {
	Permissioned      bool
	Permissionless    bool
	RespectedGameType uint32
}

func FetchChainInfo(h *script.Host, input FetchChainInfoInput) (FetchChainInfoOutput, error) {
	return opcm.RunScriptSingle[FetchChainInfoInput, FetchChainInfoOutput](h, input, "FetchChainInfo.s.sol", "FetchChainInfo")
}
