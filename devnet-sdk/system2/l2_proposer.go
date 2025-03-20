package system2

import (
	"github.com/ethereum-optimism/optimism/op-service/client"
)

// L2ProposerID identifies a L2Proposer by name and chainID, is type-safe, and can be value-copied and used as map key.
type L2ProposerID idWithChain

const L2ProposerKind Kind = "L2Proposer"

func (id L2ProposerID) String() string {
	return idWithChain(id).string(L2ProposerKind)
}

func (id L2ProposerID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText(L2ProposerKind)
}

func (id *L2ProposerID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText(L2ProposerKind, data)
}

func SortL2ProposerIDs(ids []L2ProposerID) []L2ProposerID {
	return copyAndSort(ids, func(a, b L2ProposerID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

// L2Proposer is a L2 output proposer, posting claims of L2 state to L1.
type L2Proposer interface {
	Common
	ID() L2ProposerID
}

type L2ProposerConfig struct {
	CommonConfig
	ID     L2ProposerID
	Client client.RPC
}

type rpcL2Proposer struct {
	commonImpl
	id     L2ProposerID
	client client.RPC
}

var _ L2Proposer = (*rpcL2Proposer)(nil)

func NewL2Proposer(cfg L2ProposerConfig) L2Proposer {
	cfg.Log = cfg.Log.New("chainID", cfg.ID.ChainID, "id", cfg.ID)
	return &rpcL2Proposer{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		client:     cfg.Client,
	}
}

func (r *rpcL2Proposer) ID() L2ProposerID {
	return r.id
}
