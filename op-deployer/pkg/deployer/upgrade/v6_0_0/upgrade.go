package v6_0_0

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/scriptbackend"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

type UpgradeOPChainInput struct {
	Prank               common.Address  `json:"prank"`
	Opcm                common.Address  `json:"opcm"`
	EncodedChainConfigs []OPChainConfig `evm:"-" json:"chainConfigs"`
}

type OPChainConfig struct {
	SystemConfigProxy  common.Address `json:"systemConfigProxy"`
	CannonPrestate     common.Hash    `json:"cannonPrestate"`
	CannonKonaPrestate common.Hash    `json:"cannonKonaPrestate"`
}

var opChainConfigEncoder = w3.MustNewFunc("dummy((address systemConfigProxy,bytes32 cannonPrestate,bytes32 cannonKonaPrestate)[])", "")

func (u *UpgradeOPChainInput) OpChainConfigs() ([]byte, error) {
	data, err := opChainConfigEncoder.EncodeArgs(u.EncodedChainConfigs)
	if err != nil {
		return nil, fmt.Errorf("failed to encode chain configs: %w", err)
	}
	return data[4:], nil
}

func (u *UpgradeOPChainInput) UpgradeInput() ([]byte, error) {
	return u.OpChainConfigs()
}

type UpgradeOPChain struct {
	Run func(input common.Address)
}

func Upgrade(backend scriptbackend.Backend, input UpgradeOPChainInput) error {
	return scriptbackend.RunScriptVoid(backend, input, "UpgradeOPChain.s.sol", "UpgradeOPChain")
}

type Upgrader struct{}

func (u *Upgrader) Upgrade(backend scriptbackend.Backend, input json.RawMessage) error {
	var upgradeInput UpgradeOPChainInput
	if err := json.Unmarshal(input, &upgradeInput); err != nil {
		return fmt.Errorf("failed to unmarshal input: %w", err)
	}
	return Upgrade(backend, upgradeInput)
}

func (u *Upgrader) ArtifactsURL() string {
	return artifacts.EmbeddedLocatorString
}

var DefaultUpgrader = new(Upgrader)
