package apis

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L2FollowClient interface {
	SafeL2(ctx context.Context) (eth.L2BlockRef, error)
	FinalizedL2(ctx context.Context) (eth.L2BlockRef, error)
	CurrentL1(ctx context.Context) (eth.L1BlockRef, error)
}
