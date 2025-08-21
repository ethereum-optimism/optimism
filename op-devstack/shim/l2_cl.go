package shim

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type L2CLNodeConfig struct {
	CommonConfig
	ID     stack.L2CLNodeID
	Client client.RPC

	InteropEndpoint  string
	InteropJwtSecret eth.Bytes32
}

type rpcL2CLNode struct {
	commonImpl
	id           stack.L2CLNodeID
	client       client.RPC
	rollupClient apis.RollupClient
	p2pClient    apis.P2PClient
	els          locks.RWMap[stack.L2ELNodeID, stack.L2ELNode]

	// Store interop ws endpoints and secrets to provide to the supervisor,
	// when reconnection happens using the supervisor's admin_addL2RPC method.
	// These fields are not intended for manual dial-in or initializing client.RPC
	interopEndpoint  string
	interopJwtSecret eth.Bytes32
}

var _ stack.L2CLNode = (*rpcL2CLNode)(nil)
var _ stack.LinkableL2CLNode = (*rpcL2CLNode)(nil)

func NewL2CLNode(cfg L2CLNodeConfig) stack.L2CLNode {
	cfg.T = cfg.T.WithCtx(stack.ContextWithID(cfg.T.Ctx(), cfg.ID))
	return &rpcL2CLNode{
		commonImpl:       newCommon(cfg.CommonConfig),
		id:               cfg.ID,
		client:           cfg.Client,
		rollupClient:     sources.NewRollupClient(cfg.Client),
		p2pClient:        sources.NewP2PClient(cfg.Client),
		interopEndpoint:  cfg.InteropEndpoint,
		interopJwtSecret: cfg.InteropJwtSecret,
	}
}

func (r *rpcL2CLNode) ClientRPC() client.RPC {
	return r.client
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

func (r *rpcL2CLNode) InteropRPC() (endpoint string, jwtSecret eth.Bytes32) {
	return r.interopEndpoint, r.interopJwtSecret
}
