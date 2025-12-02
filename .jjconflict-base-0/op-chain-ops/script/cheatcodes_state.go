package script

import (
	"errors"
)

func (c *CheatCodesPrecompile) LoadAllocs(pathToAllocsJson string) error {
	c.h.log.Info("loading state", "target", pathToAllocsJson)
	return errors.New("state-loading is not supported")
}

func (c *CheatCodesPrecompile) DumpState(pathToStateJson string) error {
	c.h.log.Info("dumping state", "target", pathToStateJson)

	allocs, err := c.h.StateDump()
	if err != nil {
		return err
	}
	// This may be written somewhere in the future (or run some callback to collect the state dump)
	_ = allocs
	c.h.log.Info("state-dumping is not supported, but have state",
		"path", pathToStateJson, "accounts", len(allocs.Accounts))
	return nil
}
