package shim

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type InteropFilterConfig struct {
	CommonConfig
	ID     stack.InteropFilterID
	Client client.RPC
}

type rpcInteropFilter struct {
	commonImpl
	id     stack.InteropFilterID
	client client.RPC
}

var _ stack.InteropFilter = (*rpcInteropFilter)(nil)

func NewInteropFilter(cfg InteropFilterConfig) stack.InteropFilter {
	cfg.T = cfg.T.WithCtx(stack.ContextWithID(cfg.T.Ctx(), cfg.ID))
	return &rpcInteropFilter{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		client:     cfg.Client,
	}
}

func (r *rpcInteropFilter) ID() stack.InteropFilterID {
	return r.id
}

func (r *rpcInteropFilter) QueryAPI() stack.InteropFilterQueryAPI {
	return &rpcInteropFilterQueryAPI{client: r.client}
}

type rpcInteropFilterQueryAPI struct {
	client client.RPC
}

func (q *rpcInteropFilterQueryAPI) CheckAccessList(ctx context.Context, inboxEntries []common.Hash,
	minSafety types.SafetyLevel, executingDescriptor types.ExecutingDescriptor) error {
	return q.client.CallContext(ctx, nil, "supervisor_checkAccessList", inboxEntries, minSafety, executingDescriptor)
}
