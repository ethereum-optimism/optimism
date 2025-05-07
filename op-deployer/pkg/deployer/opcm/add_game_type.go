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
	OPCMImpl                common.Address `abi:"opcmImpl"`
	SystemConfigProxy       common.Address
	OPChainProxyAdmin       common.Address `abi:"opChainProxyAdmin"`
	DelayedWETHProxy        common.Address
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

type addGameTypeInputJSON struct {
	Prank                   common.Address `json:"prank"`
	OPCMImpl                common.Address `json:"opcmimpl"`
	SystemConfigProxy       common.Address `json:"systemConfigProxy"`
	OPChainProxyAdmin       common.Address `json:"opChainProxyAdmin"`
	DelayedWETHProxy        common.Address `json:"delayedWETHProxy"`
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

func (a *AddGameTypeInput) UnmarshalJSON(b []byte) error {
	var alias addGameTypeInputJSON
	if err := json.Unmarshal(b, &alias); err != nil {
		return err
	}

	a.Prank = alias.Prank
	a.OPCMImpl = alias.OPCMImpl
	a.SystemConfigProxy = alias.SystemConfigProxy
	a.OPChainProxyAdmin = alias.OPChainProxyAdmin
	a.DelayedWETHProxy = alias.DelayedWETHProxy
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

func (a AddGameTypeInput) MarshalJSON() ([]byte, error) {
	alias := addGameTypeInputJSON{
		Prank:                   a.Prank,
		OPCMImpl:                a.OPCMImpl,
		SystemConfigProxy:       a.SystemConfigProxy,
		OPChainProxyAdmin:       a.OPChainProxyAdmin,
		DelayedWETHProxy:        a.DelayedWETHProxy,
		DisputeGameType:         a.DisputeGameType,
		DisputeAbsolutePrestate: a.DisputeAbsolutePrestate,
		DisputeClockExtension:   a.DisputeClockExtension,
		DisputeMaxClockDuration: a.DisputeMaxClockDuration,
		VM:                      a.VM,
		Permissioned:            a.Permissioned,
		SaltMixer:               a.SaltMixer,
	}

	if a.DisputeMaxGameDepth != nil {
		alias.DisputeMaxGameDepth = (*hexutil.Big)(a.DisputeMaxGameDepth)
	}

	if a.DisputeSplitDepth != nil {
		alias.DisputeSplitDepth = (*hexutil.Big)(a.DisputeSplitDepth)
	}

	if a.InitialBond != nil {
		alias.InitialBond = (*hexutil.Big)(a.InitialBond)
	}

	return json.Marshal(alias)
}

type AddGameTypeOutput struct {
	DelayedWETHProxy      common.Address `json:"delayedWETHProxy"`
	FaultDisputeGameProxy common.Address `json:"faultDisputeGameProxy"`
}

type AddGameTypeScript script.DeployScriptWithOutput[AddGameTypeInput, AddGameTypeOutput]

func NewAddGameTypeScript(host *script.Host) (AddGameTypeScript, error) {
	return script.NewDeployScriptWithOutputFromFile[AddGameTypeInput, AddGameTypeOutput](host, "AddGameType.s.sol", "AddGameType")
}
