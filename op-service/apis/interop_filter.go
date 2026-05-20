package apis

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

type InteropFilterQueryAPI interface {
	CheckAccessList(ctx context.Context, inboxEntries []common.Hash,
		minSafety safety.Level, executingDescriptor messages.ExecutingDescriptor) error
	GetBlockHashByNumber(ctx context.Context, chainID eth.ChainID, blockNum rpc.BlockNumber) (common.Hash, error)
}
