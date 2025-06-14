package frontend

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type SupervisorQueryFrontend struct {
	Supervisor apis.SupervisorQueryAPI
}

var _ apis.SupervisorQueryAPI = (*SupervisorQueryFrontend)(nil)

func (q *SupervisorQueryFrontend) CheckAccessList(ctx context.Context, inboxEntries []common.Hash,
	minSafety types.SafetyLevel, executingDescriptor types.ExecutingDescriptor) error {
	return q.Supervisor.CheckAccessList(ctx, inboxEntries, minSafety, executingDescriptor)
}

func (q *SupervisorQueryFrontend) LocalUnsafe(ctx context.Context, chainID eth.ChainID) (eth.BlockID, error) {
	return q.Supervisor.LocalUnsafe(ctx, chainID)
}

func (q *SupervisorQueryFrontend) LocalSafe(ctx context.Context, chainID eth.ChainID) (types.DerivedIDPair, error) {
	return q.Supervisor.LocalSafe(ctx, chainID)
}

func (q *SupervisorQueryFrontend) CrossSafe(ctx context.Context, chainID eth.ChainID) (types.DerivedIDPair, error) {
	return q.Supervisor.CrossSafe(ctx, chainID)
}

func (q *SupervisorQueryFrontend) Finalized(ctx context.Context, chainID eth.ChainID) (eth.BlockID, error) {
	return q.Supervisor.Finalized(ctx, chainID)
}

func (q *SupervisorQueryFrontend) FinalizedL1(ctx context.Context) (eth.BlockRef, error) {
	return q.Supervisor.FinalizedL1(ctx)
}

func (q *SupervisorQueryFrontend) CrossDerivedToSource(ctx context.Context, chainID eth.ChainID, derived eth.BlockID) (derivedFrom eth.BlockRef, err error) {
	return q.Supervisor.CrossDerivedToSource(ctx, chainID, derived)
}

func (q *SupervisorQueryFrontend) SuperRootAtTimestamp(ctx context.Context, timestamp hexutil.Uint64) (eth.SuperRootResponse, error) {
	return q.Supervisor.SuperRootAtTimestamp(ctx, timestamp)
}

func (q *SupervisorQueryFrontend) AllSafeDerivedAt(ctx context.Context, derivedFrom eth.BlockID) (derived map[eth.ChainID]eth.BlockID, err error) {
	return q.Supervisor.AllSafeDerivedAt(ctx, derivedFrom)
}

func (q *SupervisorQueryFrontend) SyncStatus(ctx context.Context) (eth.SupervisorSyncStatus, error) {
	return q.Supervisor.SyncStatus(ctx)
}

type SupervisorAdminFrontend struct {
}

var _ apis.SupervisorAdminAPI = (*SupervisorAdminFrontend)(nil)

// Start starts the service, if it was previously stopped.
func (a *SupervisorAdminFrontend) Start(ctx context.Context) error {
	return ErrNotImplemented
}

// Stop stops the service, if it was previously started.
func (a *SupervisorAdminFrontend) Stop(ctx context.Context) error {
	return ErrNotImplemented
}

// AddL2RPC adds a new L2 chain to the supervisor backend
func (a *SupervisorAdminFrontend) AddL2RPC(ctx context.Context, rpc string, jwtSecret eth.Bytes32) error {
	return ErrNotImplemented
}

// Rewind removes some L2 chain data from the supervisor backend, starting from the given block.
func (a *SupervisorAdminFrontend) Rewind(ctx context.Context, chain eth.ChainID, block eth.BlockID) error {
	return ErrNotImplemented
}
