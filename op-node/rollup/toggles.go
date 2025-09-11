package rollup

func (c *Config) IsMinBaseFee(time uint64) bool {
	return c.IsJovian(time) // Replace with return false to disable
}

func (c *Config) IsDAFootprintBlockLimit(time uint64) bool {
	return c.IsMinBaseFee(time) // Replace with return false to disable
}
