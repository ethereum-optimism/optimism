package shim

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/locks"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type L2CLNodeConfig struct {
	CommonConfig
	ID     stack.L2CLNodeID
	Client client.RPC
}

type rpcL2CLNode struct {
	commonImpl
	id           stack.L2CLNodeID
	client       client.RPC
	rollupClient apis.RollupClient
	p2pClient    apis.P2PClient
	els          locks.RWMap[stack.L2ELNodeID, stack.L2ELNode]
}

var _ stack.L2CLNode = (*rpcL2CLNode)(nil)
var _ stack.LinkableL2CLNode = (*rpcL2CLNode)(nil)

func NewL2CLNode(cfg L2CLNodeConfig) stack.L2CLNode {
	cfg.Log = cfg.Log.New("chainID", cfg.ID.ChainID, "id", cfg.ID)
	return &rpcL2CLNode{
		commonImpl:   newCommon(cfg.CommonConfig),
		id:           cfg.ID,
		client:       cfg.Client,
		rollupClient: sources.NewRollupClient(cfg.Client),
		p2pClient:    sources.NewP2PClient(cfg.Client),
	}
}

func (r *rpcL2CLNode) ID() stack.L2CLNodeID {
	return r.id
}

func (r *rpcL2CLNode) RollupAPI() apis.RollupClient {
	return r.rollupClient
}

func (r *rpcL2CLNode) P2PAPI() apis.P2PClient {
	return r.p2pClient
}

func (r *rpcL2CLNode) LinkEL(el stack.L2ELNode) {
	r.els.Set(el.ID(), el)
}

func (r *rpcL2CLNode) ELs() []stack.L2ELNode {
	return stack.SortL2ELNodes(r.els.Values())
}
