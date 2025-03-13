package frontend

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type FrontendMetrics interface {
	opmetrics.RPCServerMetricer
}

type Backend interface {
	sources.SupervisorAdminAPI
	sources.SupervisorQueryAPI
}

type QueryFrontend struct {
	Supervisor sources.SupervisorQueryAPI
	Metrics    FrontendMetrics
}

var _ sources.SupervisorQueryAPI = (*QueryFrontend)(nil)

func (q *QueryFrontend) CheckAccessList(ctx context.Context, inboxEntries []common.Hash,
	minSafety types.SafetyLevel, executingDescriptor types.ExecutingDescriptor) error {
	defer q.maybeRecordMetric("checkAccessList")
	return q.Supervisor.CheckAccessList(ctx, inboxEntries, minSafety, executingDescriptor)
}

func (q *QueryFrontend) LocalUnsafe(ctx context.Context, chainID eth.ChainID) (eth.BlockID, error) {
	defer q.maybeRecordMetric("localUnsafe")
	return q.Supervisor.LocalUnsafe(ctx, chainID)
}

func (q *QueryFrontend) CrossSafe(ctx context.Context, chainID eth.ChainID) (types.DerivedIDPair, error) {
	defer q.maybeRecordMetric("crossSafe")
	return q.Supervisor.CrossSafe(ctx, chainID)
}

func (q *QueryFrontend) Finalized(ctx context.Context, chainID eth.ChainID) (eth.BlockID, error) {
	defer q.maybeRecordMetric("finalized")
	return q.Supervisor.Finalized(ctx, chainID)
}

func (q *QueryFrontend) FinalizedL1(ctx context.Context) (eth.BlockRef, error) {
	defer q.maybeRecordMetric("finalizedL1")
	return q.Supervisor.FinalizedL1(ctx)
}

// CrossDerivedFrom is deprecated, but remains for backwards compatibility to callers
// it is equivalent to CrossDerivedToSource
func (q *QueryFrontend) CrossDerivedFrom(ctx context.Context, chainID eth.ChainID, derived eth.BlockID) (derivedFrom eth.BlockRef, err error) {
	defer q.maybeRecordMetric("crossDerivedFrom")
	return q.Supervisor.CrossDerivedToSource(ctx, chainID, derived)
}

func (q *QueryFrontend) CrossDerivedToSource(ctx context.Context, chainID eth.ChainID, derived eth.BlockID) (derivedFrom eth.BlockRef, err error) {
	defer q.maybeRecordMetric("crossDerivedToSource")
	return q.Supervisor.CrossDerivedToSource(ctx, chainID, derived)
}

func (q *QueryFrontend) SuperRootAtTimestamp(ctx context.Context, timestamp hexutil.Uint64) (eth.SuperRootResponse, error) {
	defer q.maybeRecordMetric("superRootAtTimestamp")
	return q.Supervisor.SuperRootAtTimestamp(ctx, timestamp)
}

func (q *QueryFrontend) AllSafeDerivedAt(ctx context.Context, derivedFrom eth.BlockID) (derived map[eth.ChainID]eth.BlockID, err error) {
	defer q.maybeRecordMetric("allSafeDerivedAt")
	return q.Supervisor.AllSafeDerivedAt(ctx, derivedFrom)
}

func (q *QueryFrontend) SyncStatus(ctx context.Context) (eth.SupervisorSyncStatus, error) {
	defer q.maybeRecordMetric("syncStatus")
	return q.Supervisor.SyncStatus(ctx)
}

func (q *QueryFrontend) maybeRecordMetric(name string) func() {
	return maybeRecordMetric(q.Metrics, "supervisor", name)
}

type AdminFrontend struct {
	Supervisor Backend
	Metrics    FrontendMetrics
}

var _ sources.SupervisorAdminAPI = (*AdminFrontend)(nil)

// Start starts the service, if it was previously stopped.
func (a *AdminFrontend) Start(ctx context.Context) error {
	defer a.maybeRecordMetric("start")
	return a.Supervisor.Start(ctx)
}

// Stop stops the service, if it was previously started.
func (a *AdminFrontend) Stop(ctx context.Context) error {
	defer a.maybeRecordMetric("stop")
	return a.Supervisor.Stop(ctx)
}

// AddL2RPC adds a new L2 chain to the supervisor backend
func (a *AdminFrontend) AddL2RPC(ctx context.Context, rpc string, jwtSecret eth.Bytes32) error {
	defer a.maybeRecordMetric("addL2RPC")
	return a.Supervisor.AddL2RPC(ctx, rpc, jwtSecret)
}

func (a *AdminFrontend) maybeRecordMetric(name string) func() {
	return maybeRecordMetric(a.Metrics, "admin", name)
}

// maybeRecordMetric emits metrics in the given namespace, only if we have an instance.
// It returns a function to be invoked at the end of the method being recorded.
func maybeRecordMetric(m FrontendMetrics, ns string, endpoint string) func() {
	if m != nil {
		method := fmt.Sprintf("%s_%s", ns, endpoint)
		return m.RecordRPCServerRequest(method)
	}
	return func() {}
}
