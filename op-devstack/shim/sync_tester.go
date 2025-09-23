package shim

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-sync-tester/synctester"
)

type SyncTesterConfig struct {
	CommonConfig
	ID     stack.SyncTesterID
	Addr   string
	Client client.RPC
}

// presetSyncTester wraps around a syncTester-service,
type presetSyncTester struct {
	commonImpl
	id stack.SyncTesterID
	// Endpoint for initializing RPC Client per session
	addr string
	// RPC Client initialized without session
	syncTesterClient *sources.SyncTesterClient
}

var _ stack.SyncTester = (*presetSyncTester)(nil)

func NewSyncTester(cfg SyncTesterConfig) stack.SyncTester {
	cfg.T = cfg.T.WithCtx(stack.ContextWithID(cfg.T.Ctx(), cfg.ID))
	return &presetSyncTester{
		id:               cfg.ID,
		commonImpl:       newCommon(cfg.CommonConfig),
		addr:             cfg.Addr,
		syncTesterClient: sources.NewSyncTesterClient(cfg.Client),
	}
}

func (p *presetSyncTester) ID() stack.SyncTesterID {
	return p.id
}

func (p *presetSyncTester) API() apis.SyncTester {
	return p.syncTesterClient
}

func (p *presetSyncTester) APIWithSession(sessionID string) apis.SyncTester {
	require := p.T().Require()
	require.NoError(synctester.IsValidSessionID(sessionID))
	rpcCl, err := client.NewRPC(p.T().Ctx(), p.Logger(), p.addr+fmt.Sprintf("/%s", sessionID), client.WithLazyDial())
	require.NoError(err, "sync tester failed to initialize rpc per session")
	return sources.NewSyncTesterClient(rpcCl)
}
