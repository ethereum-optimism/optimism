package broadcaster

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Broadcaster interface {
	Broadcast(ctx context.Context) ([]BroadcastResult, error)
	Hook(bcast Broadcast)
}

type BroadcastResult struct {
	Broadcast Broadcast      `json:"broadcast"`
	TxHash    common.Hash    `json:"txHash"`
	Receipt   *types.Receipt `json:"receipt"`
	Err       error          `json:"-"`
}
