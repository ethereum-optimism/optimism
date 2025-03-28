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

type FormattedFetchChainInfoOutput struct {
	Addresses        Addresses        `toml:"addresses" json:"addresses"`
	Roles            Roles            `toml:"roles" json:"roles"`
	FaultProofStatus FaultProofStatus `toml:"fault_proofs" json:"faultProofs"`
}

type FaultProofStatus struct {
	Permissioned      bool   `toml:"permissioned" json:"permissioned"`
	Permissionless    bool   `toml:"permissionless" json:"permissionless"`
	RespectedGameType uint32 `toml:"respected_game_type" json:"respectedGameType"`
}

func FetchChainInfo(h *script.Host, input FetchChainInfoInput) (FetchChainInfoOutput, error) {
	return opcm.RunScriptSingle[FetchChainInfoInput, FetchChainInfoOutput](h, input, "FetchChainInfo.s.sol", "FetchChainInfo")
}
