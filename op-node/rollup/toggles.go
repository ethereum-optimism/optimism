package rollup

// This file contains ephemeral feature toggles which should be removed
// after the fork scope is locked.

func (c *Config) IsMinBaseFee(time uint64) bool {
	return c.IsJovian(time) // Replace with return false to disable
}

func (c *Config) IsDAFootprintBlockLimit(time uint64) bool {
	return c.IsJovian(time) // Replace with return false to disable
}

func (c *Config) IsOperatorFeeFix(time uint64) bool {
	return c.IsJovian(time) // Replace with return false to disable
}
