package controller

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/runcfg2"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const runConfigPollInterval = time.Second * 60

func (s *RunCfgState) maybeUpdate() {
	if s.IsBusy() {
		return
	}
	now := s.state.Now()
	if !s.NeedPoll(runConfigPollInterval, now) {
		return
	}
	// TODO backoff
	l1Block := s.state.L1State().confirmedL1.ID()
	if l1Block == (eth.BlockID{}) {
		return
	}
	// TODO register poll attempt

	s.Emit(s.rootCtx, runcfg2.RunCfgUpdateRequestEvent{L1Block: l1Block}, nil)

	// TODO: on the latest L1 blocks, the receipts cache is warm,
	//  and we can parse the receipts to get the run-config updates,
	//  without overwhelming the L1 with state-reads or only doing the less frequent polling.
}
