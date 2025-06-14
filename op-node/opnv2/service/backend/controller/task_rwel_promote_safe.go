package controller

import "github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/rwel"

func (c *RWELState) maybePromoteCrossSafe() {
	if c.Forkchoice.IsBusy() {
		return
	}
	db, ok := c.state.ChainDB(c.ChainID())
	if !ok {
		return
	}
	if c.crossSafeBackoff.IsBackedOff(c.state.Now()) {
		return
	}
	if db.crossSafe.Derived.Number > c.crossSafe.Number &&
		db.crossSafe.Derived.Number <= c.localUnsafe.Number {
		// If EL is on a different chain then this will error, and backoff cross-safe work
		c.Forkchoice.Emit(c.rootCtx, rwel.PromoteCrossSafeEvent{
			ID: db.crossSafe.Derived.ID()}, nil)
		// TODO: handle errors, backoff as needed
	}
}
