package apis

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type SyncTester interface {
	ChainID(ctx context.Context) (eth.ChainID, error)
}
