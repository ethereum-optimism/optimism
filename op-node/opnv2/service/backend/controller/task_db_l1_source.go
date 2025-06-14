package controller

// L1 source-block checks: ensure that the local-safe DBs and cross-safe DBs are following the canonical L1 chain.
func (s *ChainDBState) maybeCheckL1Source() {
	// TODO: compare L1access and DB source, to force a check, if needed immediately

	// If L1 changed, then compare L1 accessor head to the local/cross-safe DB L1 source block

	// TODO trigger l1rewind.L1RewindCheckEvent{}
}
