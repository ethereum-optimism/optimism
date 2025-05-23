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
	ID                 stack.TestSequencerID
	Client             client.RPC
	L2SequencerClients map[eth.ChainID]client.RPC
}

type rpcTestSequencer struct {
	commonImpl
	id stack.TestSequencerID

	client       client.RPC
	api          apis.TestSequencerAPI
	l2sequencers map[eth.ChainID]apis.TestSequencerIndividualAPI
}

var _ stack.TestSequencer = (*rpcTestSequencer)(nil)

func NewTestSequencer(cfg TestSequencerConfig) stack.TestSequencer {
	cfg.T = cfg.T.WithCtx(stack.ContextWithKind(cfg.T.Ctx(), stack.TestSequencerKind), "id", cfg.ID)
	s := &rpcTestSequencer{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		client:     cfg.Client,
		api:        sources.NewBuilderClient(cfg.Client),
	}

	s.l2sequencers = make(map[eth.ChainID]apis.TestSequencerIndividualAPI)
	for k, v := range cfg.L2SequencerClients {
		s.l2sequencers[k] = sources.NewIndividualClient(v)
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

func (r *rpcTestSequencer) IndividualAPI(chainID eth.ChainID) apis.TestSequencerIndividualAPI {
	return r.l2sequencers[chainID]
}
