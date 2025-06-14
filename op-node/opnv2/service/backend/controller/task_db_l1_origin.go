package controller

func (s *ChainDBState) maybeCheckL1Origin() {
	// If L1 changed, then compare L1 accessor head to the DB | REL | RWEL L1Origin
	//
	// for any DB | REL | RWEL, if local-unsafe head does not have validated L1 origin in the last 60 seconds, then validate it.
	//   if it does not match we need to reorg away, and not use the REL|RWEL until reorged, or until revalidated.
	//   if it is the DB, then we need to rewind it

	// TODO
}
