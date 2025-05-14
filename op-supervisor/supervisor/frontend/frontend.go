package frontend

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type Backend interface {
	apis.SupervisorAdminAPI
	apis.SupervisorQueryAPI
}

type QueryFrontend struct {
	Supervisor apis.SupervisorQueryAPI
}

var _ apis.SupervisorQueryAPI = (*QueryFrontend)(nil)

func (q *QueryFrontend) CheckAccessList(ctx context.Context, inboxEntries []common.Hash,
	minSafety types.SafetyLevel, executingDescriptor types.ExecutingDescriptor) error {
	return q.Supervisor.CheckAccessList(ctx, inboxEntries, minSafety, executingDescriptor)
}

func (q *QueryFrontend) LocalUnsafe(ctx context.Context, chainID eth.ChainID) (eth.BlockID, error) {
	return q.Supervisor.LocalUnsafe(ctx, chainID)
}

func (q *QueryFrontend) LocalSafe(ctx context.Context, chainID eth.ChainID) (types.DerivedIDPair, error) {
	return q.Supervisor.LocalSafe(ctx, chainID)
}

func (q *QueryFrontend) CrossSafe(ctx context.Context, chainID eth.ChainID) (types.DerivedIDPair, error) {
	return q.Supervisor.CrossSafe(ctx, chainID)
}

func (q *QueryFrontend) Finalized(ctx context.Context, chainID eth.ChainID) (eth.BlockID, error) {
	return q.Supervisor.Finalized(ctx, chainID)
}

func (q *QueryFrontend) FinalizedL1(ctx context.Context) (eth.BlockRef, error) {
	return q.Supervisor.FinalizedL1(ctx)
}

func (q *QueryFrontend) CrossDerivedToSource(ctx context.Context, chainID eth.ChainID, derived eth.BlockID) (derivedFrom eth.BlockRef, err error) {
	return q.Supervisor.CrossDerivedToSource(ctx, chainID, derived)
}

func (q *QueryFrontend) SuperRootAtTimestamp(ctx context.Context, timestamp hexutil.Uint64) (eth.SuperRootResponse, error) {
	return q.Supervisor.SuperRootAtTimestamp(ctx, timestamp)
}

func (q *QueryFrontend) AllSafeDerivedAt(ctx context.Context, derivedFrom eth.BlockID) (derived map[eth.ChainID]eth.BlockID, err error) {
	return q.Supervisor.AllSafeDerivedAt(ctx, derivedFrom)
}

func (q *QueryFrontend) SyncStatus(ctx context.Context) (eth.SupervisorSyncStatus, error) {
	return q.Supervisor.SyncStatus(ctx)
}

type AdminFrontend struct {
	Supervisor Backend
}

var _ apis.SupervisorAdminAPI = (*AdminFrontend)(nil)

// Start starts the service, if it was previously stopped.
func (a *AdminFrontend) Start(ctx context.Context) error {
	return a.Supervisor.Start(ctx)
}

// Stop stops the service, if it was previously started.
func (a *AdminFrontend) Stop(ctx context.Context) error {
	return a.Supervisor.Stop(ctx)
}

// AddL2RPC adds a new L2 chain to the supervisor backend
func (a *AdminFrontend) AddL2RPC(ctx context.Context, rpc string, jwtSecret eth.Bytes32) error {
	return a.Supervisor.AddL2RPC(ctx, rpc, jwtSecret)
}

// Rewind removes some L2 chain data from the supervisor backend, starting from the given block.
func (a *AdminFrontend) Rewind(ctx context.Context, chain eth.ChainID, block eth.BlockID) error {
	// TODO(#15665) add logging here to track when rewinds are requested
	return a.Supervisor.Rewind(ctx, chain, block)
}
