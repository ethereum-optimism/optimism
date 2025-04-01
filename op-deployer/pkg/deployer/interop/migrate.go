package interop

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

type InteropMigrationInput struct {
	Prank common.Address `json:"prank"`
	Opcm  common.Address `json:"opcm"`

	UsePermissionlessGame          bool           `json:"usePermissionlessGame"`
	StartingAnchorRoot             common.Hash    `json:"startingAnchorRoot"`
	StartingAnchorL2SequenceNumber *big.Int       `json:"startingAnchorL2SequenceNumber"`
	Proposer                       common.Address `json:"proposer"`
	Challenger                     common.Address `json:"challenger"`
	MaxGameDepth                   uint64         `json:"maxGameDepth"`
	SplitDepth                     uint64         `json:"splitDepth"`
	InitBond                       *big.Int       `json:"initBond"`
	ClockExtension                 uint64         `json:"clockExtension"`
	MaxClockDuration               uint64         `json:"maxClockDuration"`

	EncodedChainConfigs []OPChainConfig `evm:"-" json:"chainConfigs"`
}

func (u *InteropMigrationInput) OpChainConfigs() ([]byte, error) {
	data, err := opChainConfigEncoder.EncodeArgs(u.EncodedChainConfigs)
	if err != nil {
		return nil, fmt.Errorf("failed to encode chain configs: %w", err)
	}
	return data[4:], nil
}

type OPChainConfig struct {
	SystemConfigProxy common.Address `json:"systemConfigProxy"`
	ProxyAdmin        common.Address `json:"proxyAdmin"`
	AbsolutePrestate  common.Hash    `json:"absolutePrestate"`
}

type InteropMigrationOutput struct {
	DisputeGameFactory common.Address `json:"disputeGameFactory"`
}

func (output *InteropMigrationOutput) CheckOutput(input common.Address) error {
	return nil
}

var opChainConfigEncoder = w3.MustNewFunc("dummy((address systemConfigProxy,address proxyAdmin,bytes32 absolutePrestate)[])", "")

type InteropMigration struct {
	Run func(input common.Address)
}

func Migrate(host *script.Host, input InteropMigrationInput) (InteropMigrationOutput, error) {
	return opcm.RunScriptSingle[InteropMigrationInput, InteropMigrationOutput](host, input, "InteropMigration.s.sol", "InteropMigration")
}
