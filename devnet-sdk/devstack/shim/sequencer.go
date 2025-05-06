package shim

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type SequencerConfig struct {
	CommonConfig
	ID                 stack.SequencerID
	Client             client.RPC
	L2SequencerClients map[eth.ChainID]client.RPC
}

type rpcSequencer struct {
	commonImpl
	id stack.SequencerID

	client       client.RPC
	api          apis.SequencerAPI
	l2sequencers map[eth.ChainID]apis.SequencerIndividualAPI
}

var _ stack.Sequencer = (*rpcSequencer)(nil)

func NewSequencer(cfg SequencerConfig) stack.Sequencer {
	cfg.Log = cfg.Log.New("id", cfg.ID)
	s := &rpcSequencer{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		client:     cfg.Client,
		api:        sources.NewBuilderClient(cfg.Client),
	}

	s.l2sequencers = make(map[eth.ChainID]apis.SequencerIndividualAPI)
	for k, v := range cfg.L2SequencerClients {
		s.l2sequencers[k] = sources.NewIndividualClient(v)
	}
	return s
}

func (r *rpcSequencer) ID() stack.SequencerID {
	return r.id
}

func (r *rpcSequencer) AdminAPI() apis.SequencerAdminAPI {
	return r.api
}

func (r *rpcSequencer) BuildAPI() apis.SequencerBuildAPI {
	return r.api
}

func (r *rpcSequencer) IndividualAPI(chainID eth.ChainID) apis.SequencerIndividualAPI {
	return r.l2sequencers[chainID]
}
