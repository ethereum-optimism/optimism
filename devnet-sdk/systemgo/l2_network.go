package systemgo

import (
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/core"
)

type L2Network struct {
	genesis   *core.Genesis
	rollupCfg *rollup.Config
}
