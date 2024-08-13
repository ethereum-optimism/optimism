package script

import (
	"errors"

	"github.com/ethereum/go-ethereum/core/state"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
)

func (c *CheatCodesPrecompile) LoadAllocs(pathToAllocsJson string) error {
	c.h.log.Info("loading state", "target", pathToAllocsJson)
	return errors.New("state-loading is not supported")
}

func (c *CheatCodesPrecompile) DumpState(pathToStateJson string) {
	c.h.log.Info("dumping state", "target", pathToStateJson)
	var allocs foundry.ForgeAllocs
	c.h.state.DumpToCollector(&allocs, &state.DumpConfig{
		OnlyWithAddresses: true,
	})
	_ = allocs
	c.h.log.Warn("state-dumping is not supported, but have state",
		"path", pathToStateJson, "accounts", len(allocs.Accounts))
}
