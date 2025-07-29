package shim

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type TestSequencerConfig struct {
	CommonConfig
	ID             stack.TestSequencerID
	Client         client.RPC
	ControlClients map[eth.ChainID]client.RPC
}

type rpcTestSequencer struct {
	commonImpl
	id stack.TestSequencerID

	client   client.RPC
	api      apis.TestSequencerAPI
	controls map[eth.ChainID]apis.TestSequencerControlAPI
}

var _ stack.TestSequencer = (*rpcTestSequencer)(nil)

func NewTestSequencer(cfg TestSequencerConfig) stack.TestSequencer {
	cfg.T = cfg.T.WithCtx(stack.ContextWithID(cfg.T.Ctx(), cfg.ID))
	s := &rpcTestSequencer{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		client:     cfg.Client,
		api:        sources.NewBuilderClient(cfg.Client),
	}

	s.controls = make(map[eth.ChainID]apis.TestSequencerControlAPI)
	for k, v := range cfg.ControlClients {
		s.controls[k] = sources.NewControlClient(v)
	}
	return s
}

func (r *rpcTestSequencer) ID() stack.TestSequencerID {
	return r.id
}

func (r *rpcTestSequencer) AdminAPI() apis.TestSequencerAdminAPI {
	return r.api
}

func (r *rpcTestSequencer) BuildAPI() apis.TestSequencerBuildAPI {
	return r.api
}

func (r *rpcTestSequencer) ControlAPI(chainID eth.ChainID) apis.TestSequencerControlAPI {
	return r.controls[chainID]
}
