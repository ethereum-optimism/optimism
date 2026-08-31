package sysgo

import (
	"errors"
)

// PrivateInteropConfig is the devstack's description of a private-interop chain pair.
//
// A pair is two chains sharing ONE chain ID: the PRIVATE half, which is what tests send
// transactions to, and its public RENDERING, which is what the supernode judges and what every
// counterparty means when it names the chain. See op-private-interop/docs/DESIGN.md.
//
// Everything here has a working default, because the point of the preset is that an ordinary
// interop test does not have to know it is running against a pair.
type PrivateInteropConfig struct {
	// MaxBlocksPerRange is the builder's cadence: how many private blocks one span batch, and one
	// range claim, covers.
	//
	// Production is ~300 blocks (ten minutes at 2 s). A devstack wants the opposite trade: the
	// rendering only advances when a range lands, and a test that waits ten minutes for its
	// message to become public is a test nobody runs. Small enough that the rendering tracks the
	// private chain closely, large enough that a range is a range.
	MaxBlocksPerRange uint64

	// SkipRenderingInvariant opts out of the standing rendering-invariant checker.
	//
	// The checker is on by default (the `NewSingleChainMultiNode` / `...WithoutCheck` precedent):
	// a background assertion that costs nothing when the system is healthy is an assertion that
	// should not need opting into. Tests that deliberately break the correspondence -- by stopping
	// the builder, by diverging the private chain from a landed claim -- turn it off.
	SkipRenderingInvariant bool

	// OperatorPremine is the operator EOA's opening balance on the rendering, in wei.
	//
	// It is the rendering's entire gas supply for its lifetime: the chain has no sequencer, no
	// mempool, and deposits are impossible on it, so nothing can ever top the operator up.
	OperatorPremine uint64
}

// PrivateInteropOption mutates a pair's configuration.
type PrivateInteropOption func(cfg *PrivateInteropConfig)

// DefaultPrivateInteropConfig is the devstack pair every preset gets unless a test says otherwise.
func DefaultPrivateInteropConfig() PrivateInteropConfig {
	return PrivateInteropConfig{
		MaxBlocksPerRange: 4,
		// 10_000 ETH. The rendering pays for every replay and every claim out of this and is never
		// refilled, so a devstack premine is deliberately far past anything a test can spend.
		OperatorPremine: 10_000,
	}
}

// WithPrivateInteropCadence sets how many private blocks one range covers.
func WithPrivateInteropCadence(blocks uint64) PrivateInteropOption {
	return func(cfg *PrivateInteropConfig) { cfg.MaxBlocksPerRange = blocks }
}

// WithoutRenderingInvariantCheck turns off the standing rendering-invariant checker for tests that
// deliberately break block-for-block correspondence.
func WithoutRenderingInvariantCheck() PrivateInteropOption {
	return func(cfg *PrivateInteropConfig) { cfg.SkipRenderingInvariant = true }
}

// Check validates the pair's configuration.
func (c *PrivateInteropConfig) Check() error {
	if c.MaxBlocksPerRange == 0 {
		return errors.New("private interop: the range cadence must be at least one block")
	}
	if c.OperatorPremine == 0 {
		return errors.New("private interop: the rendering's operator premine is the chain's only gas, and must be positive")
	}
	return nil
}
