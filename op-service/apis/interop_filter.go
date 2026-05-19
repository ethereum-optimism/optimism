package apis

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type InteropFilterQueryAPI interface {
	CheckAccessList(ctx context.Context, inboxEntries []common.Hash,
		minSafety types.SafetyLevel, executingDescriptor messages.ExecutingDescriptor) error
	GetBlockHashByNumber(ctx context.Context, chainID eth.ChainID, blockNum rpc.BlockNumber) (common.Hash, error)
}
