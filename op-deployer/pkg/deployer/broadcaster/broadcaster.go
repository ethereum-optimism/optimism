package broadcaster

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type BroadcastResult struct {
	Broadcast Broadcast      `json:"broadcast"`
	TxHash    common.Hash    `json:"txHash"`
	Receipt   *types.Receipt `json:"receipt"`
	Err       error          `json:"-"`
}
