package bindings

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type MigrateInput struct {
	UsePermissionlessGame bool
	StartingAnchorRoot    Proposal
	GameParameters        GameParameters
	OpChainConfigs        []OPChainConfig
}

type Proposal struct {
	Root             common.Hash
	L2SequenceNumber *big.Int
}

type GameParameters struct {
	Proposer         common.Address
	Challenger       common.Address
	MaxGameDepth     *big.Int
	SplitDepth       *big.Int
	InitBond         *big.Int
	ClockExtension   uint64
	MaxClockDuration uint64
}

type OPChainConfig struct {
	SystemConfigProxy  common.Address
	ProxyAdmin         common.Address
	CannonPrestate     common.Hash
	CannonKonaPrestate common.Hash
}

type OPContractsManager struct {
	Migrate func(input MigrateInput) TypedCall[any] `sol:"migrate"`
}
