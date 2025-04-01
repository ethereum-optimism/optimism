package stack

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// L2CLNodeID identifies a L2CLNode by name and chainID, is type-safe, and can be value-copied and used as map key.
type L2CLNodeID idWithChain

const L2CLNodeKind Kind = "L2CLNode"

func (id L2CLNodeID) String() string {
	return idWithChain(id).string(L2CLNodeKind)
}

func (id L2CLNodeID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText(L2CLNodeKind)
}

func (id *L2CLNodeID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText(L2CLNodeKind, data)
}

func SortL2CLNodeIDs(ids []L2CLNodeID) []L2CLNodeID {
	return copyAndSort(ids, func(a, b L2CLNodeID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

type RollupAPI interface {
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
}

// L2CLNode is a L2 ethereum consensus-layer node
type L2CLNode interface {
	Common
	ID() L2CLNodeID

	RollupAPI() RollupAPI
}
