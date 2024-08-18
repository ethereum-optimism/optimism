package script

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core/state"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
)

func (c *CheatCodesPrecompile) LoadAllocs(pathToAllocsJson string) error {
	c.h.log.Info("loading state", "target", pathToAllocsJson)
	return errors.New("state-loading is not supported")
}

func (c *CheatCodesPrecompile) DumpState(pathToStateJson string) error {
	c.h.log.Info("dumping state", "target", pathToStateJson)

	// We have to commit the existing state to the trie,
	// for all the state-changes to be captured by the trie iterator.
	root, err := c.h.state.Commit(c.h.env.Context.BlockNumber.Uint64(), true)
	if err != nil {
		return fmt.Errorf("failed to commit state: %w", err)
	}
	// We need a state object around the state DB
	st, err := state.New(root, c.h.stateDB, nil)
	if err != nil {
		return fmt.Errorf("failed to create state object for state-dumping: %w", err)
	}
	// After Commit we cannot reuse the old State, so we update the host to use the new one
	c.h.state = st
	c.h.env.StateDB = st

	var allocs foundry.ForgeAllocs
	allocs.FromState(st)
	// This may be written somewhere in the future (or run some callback to collect the state dump)
	_ = allocs
	c.h.log.Info("state-dumping is not supported, but have state",
		"path", pathToStateJson, "accounts", len(allocs.Accounts))
	return nil
}
