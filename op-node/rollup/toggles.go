package rollup

// This file contains ephemeral feature toggles for the next
// fork while it is in development. They should be removed
// after the fork scope is locked.

// Example:
// func (c *Config) IsMinBaseFee(time uint64) bool {
// 	return c.IsJovian(time) // Replace with return false to disable
// }

// IsL2CM gates the L2 Contracts Manager upgrade transactions at the Karst fork.
// Replace with return false to disable NUT bundle execution during development.
func (c *Config) IsL2CM(time uint64) bool {
	return c.IsKarst(time)
}

// IsL2CMActivationBlock returns true only at the exact activation block.
func (c *Config) IsL2CMActivationBlock(l2BlockTime uint64) bool {
	if !c.IsL2CM(l2BlockTime) {
		return false
	}
	return c.IsKarstActivationBlock(l2BlockTime)
}

// IsSDM gates Sequencer-Defined Metering. When this returns false, span batches
// carrying PostExec transactions are rejected during derivation.
func (c *Config) IsSDM(time uint64) bool {
	return c.IsInterop(time)
}
