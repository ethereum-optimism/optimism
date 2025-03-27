package system2

import (
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// L1CLNodeID identifies a L1CLNode by name and chainID, is type-safe, and can be value-copied and used as map key.
type L1CLNodeID idWithChain

const L1CLNodeKind Kind = "L1CLNode"

func (id L1CLNodeID) String() string {
	return idWithChain(id).string(L1CLNodeKind)
}

func (id L1CLNodeID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText(L1CLNodeKind)
}

func (id *L1CLNodeID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText(L1CLNodeKind, data)
}

func SortL1CLNodeIDs(ids []L1CLNodeID) []L1CLNodeID {
	return copyAndSort(ids, func(a, b L1CLNodeID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

// L1CLNode is a L1 ethereum consensus-layer node, aka Beacon node.
// This node may not be a full beacon node, and instead run a mock L1 consensus node.
type L1CLNode interface {
	Common
	ID() L1CLNodeID

	BeaconClient() sources.BeaconClient
}

type L1CLNodeConfig struct {
	CommonConfig
	ID     L1CLNodeID
	Client client.HTTP
}

type rpcL1CLNode struct {
	commonImpl
	id     L1CLNodeID
	client sources.BeaconClient
}

var _ L1CLNode = (*rpcL1CLNode)(nil)

func NewL1CLNode(cfg L1CLNodeConfig) L1CLNode {
	cfg.Log = cfg.Log.New("chainID", cfg.ID.ChainID, "id", cfg.ID)
	return &rpcL1CLNode{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		client:     sources.NewBeaconHTTPClient(cfg.Client),
	}
}

func (r *rpcL1CLNode) ID() L1CLNodeID {
	return r.id
}

func (r *rpcL1CLNode) BeaconClient() sources.BeaconClient {
	return r.client
}
