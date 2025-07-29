package opcm

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type AddGameTypeInput struct {
	L1ProxyAdminOwner       common.Address `abi:"prank"`
	OPCMImpl                common.Address `abi:"opcmImpl"`
	SystemConfigProxy       common.Address
	OPChainProxyAdmin       common.Address `abi:"opChainProxyAdmin"`
	DelayedWETHProxy        common.Address
	DisputeGameType         uint32
	DisputeAbsolutePrestate common.Hash
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

	a.L1ProxyAdminOwner = alias.Prank
	a.OPCMImpl = alias.OPCMImpl
	a.SystemConfigProxy = alias.SystemConfigProxy
	a.OPChainProxyAdmin = alias.OPChainProxyAdmin
	a.DelayedWETHProxy = alias.DelayedWETHProxy
	a.DisputeGameType = alias.DisputeGameType
	a.DisputeAbsolutePrestate = alias.DisputeAbsolutePrestate

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
		Prank:                   a.L1ProxyAdminOwner,
		OPCMImpl:                a.OPCMImpl,
		SystemConfigProxy:       a.SystemConfigProxy,
		OPChainProxyAdmin:       a.OPChainProxyAdmin,
		DelayedWETHProxy:        a.DelayedWETHProxy,
		DisputeGameType:         a.DisputeGameType,
		DisputeAbsolutePrestate: a.DisputeAbsolutePrestate,
		VM:                      a.VM,
		Permissioned:            a.Permissioned,
		SaltMixer:               a.SaltMixer,
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
