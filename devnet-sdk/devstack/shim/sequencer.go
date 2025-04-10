package shim

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type SequencerConfig struct {
	CommonConfig
	ID        stack.SequencerID
	Client    client.RPC
	IndClient client.RPC
}

type rpcSequencer struct {
	commonImpl
	id stack.SequencerID

	client client.RPC
	api    apis.SequencerAPI
	ind    apis.SequencerIndividualAPI
}

var _ stack.Sequencer = (*rpcSequencer)(nil)

func NewSequencer(cfg SequencerConfig) stack.Sequencer {
	cfg.Log = cfg.Log.New("id", cfg.ID)
	return &rpcSequencer{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		client:     cfg.Client,
		api:        sources.NewBuilderClient(cfg.Client),
		ind:        sources.NewIndividualClient(cfg.IndClient),
	}
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

// TODO: convert to a collection, as we might have multiple sequencers within a given op-test-sequencer
func (r *rpcSequencer) IndividualAPI() apis.SequencerIndividualAPI {
	return r.ind
}
