package controller

func (s *ChainDBState) maybeDBInit() {

	// emit superevents.UnsafeActivationBlockEvent when we have an unsafe block that holds as exact activation block
	// emit superevents.SafeActivationBlockEvent when we have a local-safe block that holds as exact activation block

	// TODO: the controller should ideally not deal with chain configs

	// TODO: somewhere we need to init the DB, if we are not bootstrapping from a different point than genesis
	//genesis := b.cfgSet.Genesis(chainID)
	//if b.cfgSet.IsInterop(chainID, genesis.L2.Timestamp) { // This genesis check is old
	//	b.emitter.Emit(superevents.SafeActivationBlockEvent{
	//		ChainID: chainID,
	//		Safe: types.DerivedBlockRefPair{
	//			// Initialization skips parent checks, so zero parents are ok.
	//			Source:  genesis.L1.WithZeroParent(),
	//			Derived: genesis.L2.WithZeroParent(),
	//		},
	//		Ctx: event.WrapCtx(b.sysContext),
	//	})
	//}
}
