package controller

func (s *RWELState) maybeBackupSync() {
	// for any RWEL, if same-chain REL local-unsafe > RWEL local-unsafe, then try copy the block over
	// TODO: nice-to-have, until we have REL support
}
