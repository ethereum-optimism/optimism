package opcm

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type AddGameTypeInput struct {
	Prank                   common.Address
	OPCM                    common.Address `abi:"opcm"`
	SystemConfig            common.Address
	ProxyAdmin              common.Address
	DelayedWETH             common.Address
	DisputeGameType         uint32
	DisputeAbsolutePrestate common.Hash
	DisputeMaxGameDepth     *big.Int
	DisputeSplitDepth       *big.Int
	DisputeClockExtension   uint64
	DisputeMaxClockDuration uint64
	InitialBond             *big.Int
	VM                      common.Address `abi:"vm"`
	Permissioned            bool
	SaltMixer               string
}

func (a *AddGameTypeInput) UnmarshalJSON(b []byte) error {
	type addGameTypeInputJSON struct {
		Prank                   common.Address `json:"prank"`
		OPCM                    common.Address `json:"opcm"`
		SystemConfig            common.Address `json:"systemConfig"`
		ProxyAdmin              common.Address `json:"proxyAdmin"`
		DelayedWETH             common.Address `json:"delayedWETH"`
		DisputeGameType         uint32         `json:"disputeGameType"`
		DisputeAbsolutePrestate common.Hash    `json:"disputeAbsolutePrestate"`
		DisputeMaxGameDepth     *hexutil.Big   `json:"disputeMaxGameDepth"`
		DisputeSplitDepth       *hexutil.Big   `json:"disputeSplitDepth"`
		DisputeClockExtension   uint64         `json:"disputeClockExtension"`
		DisputeMaxClockDuration uint64         `json:"disputeMaxClockDuration"`
		InitialBond             *hexutil.Big   `json:"initialBond"`
		VM                      common.Address `json:"vm"`
		Permissioned            bool           `json:"permissioned"`
		SaltMixer               string         `json:"saltMixer"`
	}

	var alias addGameTypeInputJSON
	if err := json.Unmarshal(b, &alias); err != nil {
		return err
	}

	a.Prank = alias.Prank
	a.OPCM = alias.OPCM
	a.SystemConfig = alias.SystemConfig
	a.ProxyAdmin = alias.ProxyAdmin
	a.DelayedWETH = alias.DelayedWETH
	a.DisputeGameType = alias.DisputeGameType
	a.DisputeAbsolutePrestate = alias.DisputeAbsolutePrestate

	if alias.DisputeMaxGameDepth != nil {
		a.DisputeMaxGameDepth = (*big.Int)(alias.DisputeMaxGameDepth)
	}

	if alias.DisputeSplitDepth != nil {
		a.DisputeSplitDepth = (*big.Int)(alias.DisputeSplitDepth)
	}

	a.DisputeClockExtension = alias.DisputeClockExtension
	a.DisputeMaxClockDuration = alias.DisputeMaxClockDuration

	if alias.InitialBond != nil {
		a.InitialBond = (*big.Int)(alias.InitialBond)
	}

	a.VM = alias.VM
	a.Permissioned = alias.Permissioned
	a.SaltMixer = alias.SaltMixer

	return nil
}

type AddGameTypeOutput struct {
	DelayedWETH      common.Address `json:"delayedWETH"`
	FaultDisputeGame common.Address `json:"faultDisputeGame"`
}

type AddGameTypeScript script.DeployScriptWithOutput[AddGameTypeInput, AddGameTypeOutput]

func NewAddGameTypeScript(host *script.Host) (AddGameTypeScript, error) {
	return script.NewDeployScriptWithOutputFromFile[AddGameTypeInput, AddGameTypeOutput](host, "AddGameType.s.sol", "AddGameType")
}
