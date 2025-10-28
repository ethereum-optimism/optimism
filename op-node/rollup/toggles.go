package rollup

// IsMinBaseFee implements eip1559.ForkChecker interface.
// MinBaseFee feature is part of the Jovian hardfork.
func (c *Config) IsMinBaseFee(time uint64) bool {
	return c.IsJovian(time)
}
