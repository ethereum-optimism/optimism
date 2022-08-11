package actions

import (
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/log"
)

type OutputProvider interface {
	OutputRootAPI
	SyncStatusAPI
}

type L2Proposer struct {
	log       log.Logger
	rollupCfg *rollup.Config
	l1        L1TXAPI
	prov      OutputProvider
}

var _ ActorL2Proposer = (*L2Proposer)(nil)

func NewL2Proposer(log log.Logger, rollupCfg *rollup.Config, l1 L1TXAPI, prov OutputProvider) *L2Proposer {
	return &L2Proposer{
		log:       log,
		rollupCfg: rollupCfg,
		l1:        l1,
		prov:      prov,
	}
}

func (s *L2Proposer) actProposeOutputRoot(t Testing) {
	// TODO refactor proposer driver Config to use test-friendly interfaces instead of RPC bindings
	t.InvalidAction("todo propose output root")
}
