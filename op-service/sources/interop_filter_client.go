package sources

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

type InteropFilterClient struct {
	client client.RPC
}

// This type-check keeps the Interop Filter server API and client API in sync.
var _ apis.InteropFilterQueryAPI = (*InteropFilterClient)(nil)

func NewInteropFilterClient(client client.RPC) *InteropFilterClient {
	return &InteropFilterClient{
		client: client,
	}
}

func (cl *InteropFilterClient) CheckAccessList(ctx context.Context, inboxEntries []common.Hash,
	minSafety safety.Level, executingDescriptor messages.ExecutingDescriptor) error {
	return cl.client.CallContext(ctx, nil, "interop_checkAccessList", inboxEntries, minSafety, executingDescriptor)
}

func (cl *InteropFilterClient) GetBlockHashByNumber(ctx context.Context, chainID eth.ChainID, blockNum rpc.BlockNumber) (common.Hash, error) {
	var result common.Hash
	err := cl.client.CallContext(ctx, &result, "interop_getBlockHashByNumber", chainID, blockNum)
	if isNotFound(err) {
		err = fmt.Errorf("%w: %v", ethereum.NotFound, err.Error())
		return result, err
	}
	return result, err
}

func (cl *InteropFilterClient) Close() {
	cl.client.Close()
}
