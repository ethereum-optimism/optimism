package shim

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type SyncTesterConfig struct {
	CommonConfig
	ID     stack.SyncTesterID
	Client client.RPC
}

// presetSyncTester wraps around a syncTester-service,
type presetSyncTester struct {
	commonImpl
	id               stack.SyncTesterID
	syncTesterClient *sources.SyncTesterClient
}

var _ stack.SyncTester = (*presetSyncTester)(nil)

func NewSyncTester(cfg SyncTesterConfig) stack.SyncTester {
	cfg.T = cfg.T.WithCtx(stack.ContextWithID(cfg.T.Ctx(), cfg.ID))
	return &presetSyncTester{
		id:               cfg.ID,
		commonImpl:       newCommon(cfg.CommonConfig),
		syncTesterClient: sources.NewSyncTesterClient(cfg.Client),
	}
}

func (p *presetSyncTester) ID() stack.SyncTesterID {
	return p.id
}

func (p *presetSyncTester) API() apis.SyncTester {
	return p.syncTesterClient
}
