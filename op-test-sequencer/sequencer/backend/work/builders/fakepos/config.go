package fakepos

import (
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum/go-ethereum/core/types"
)

type Config struct {
	ChainConfig       types.BlockType
	Backend           Blockchain
	EngineAPI         geth.EngineAPI
	Beacon            Beacon
	FinalizedDistance uint64
	SafeDistance      uint64
	BlockTime         uint64
}
