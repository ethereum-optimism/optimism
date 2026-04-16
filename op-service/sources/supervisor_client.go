package sources

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor-lib/types"
)

type SupervisorClient struct {
	rpc client.RPC
}

func NewSupervisorClient(rpc client.RPC) *SupervisorClient {
	return &SupervisorClient{rpc: rpc}
}

func (cl *SupervisorClient) Close() {
	cl.rpc.Close()
}

func (cl *SupervisorClient) Start(ctx context.Context) error {
	return cl.rpc.CallContext(ctx, nil, "admin_start")
}

func (cl *SupervisorClient) Stop(ctx context.Context) error {
	return cl.rpc.CallContext(ctx, nil, "admin_stop")
}

func (cl *SupervisorClient) AddL2RPC(ctx context.Context, rpc string, jwtSecret eth.Bytes32) error {
	return cl.rpc.CallContext(ctx, nil, "admin_addL2RPC", rpc, jwtSecret)
}

func (cl *SupervisorClient) Rewind(ctx context.Context, chain eth.ChainID, block eth.BlockID) error {
	return cl.rpc.CallContext(ctx, nil, "admin_rewind", chain, block)
}

func (cl *SupervisorClient) SetFailsafeEnabled(ctx context.Context, enabled bool) error {
	return cl.rpc.CallContext(ctx, nil, "admin_setFailsafeEnabled", enabled)
}

func (cl *SupervisorClient) GetFailsafeEnabled(ctx context.Context) (bool, error) {
	var result bool
	err := cl.rpc.CallContext(ctx, &result, "admin_getFailsafeEnabled")
	return result, err
}

func (cl *SupervisorClient) CheckAccessList(ctx context.Context, inboxEntries []common.Hash,
	minSafety types.SafetyLevel, executingDescriptor types.ExecutingDescriptor) error {
	return cl.rpc.CallContext(ctx, nil, "supervisor_checkAccessList", inboxEntries, minSafety, executingDescriptor)
}

func (cl *SupervisorClient) CrossDerivedToSource(ctx context.Context, chainID eth.ChainID, derived eth.BlockID) (eth.BlockRef, error) {
	var result eth.BlockRef
	err := cl.rpc.CallContext(ctx, &result, "supervisor_crossDerivedToSource", chainID, derived)
	return result, err
}

func (cl *SupervisorClient) LocalUnsafe(ctx context.Context, chainID eth.ChainID) (eth.BlockID, error) {
	var result eth.BlockID
	err := cl.rpc.CallContext(ctx, &result, "supervisor_localUnsafe", chainID)
	return result, err
}

func (cl *SupervisorClient) LocalSafe(ctx context.Context, chainID eth.ChainID) (types.DerivedIDPair, error) {
	var result types.DerivedIDPair
	err := cl.rpc.CallContext(ctx, &result, "supervisor_localSafe", chainID)
	return result, err
}

func (cl *SupervisorClient) CrossSafe(ctx context.Context, chainID eth.ChainID) (types.DerivedIDPair, error) {
	var result types.DerivedIDPair
	err := cl.rpc.CallContext(ctx, &result, "supervisor_crossSafe", chainID)
	return result, err
}

func (cl *SupervisorClient) Finalized(ctx context.Context, chainID eth.ChainID) (eth.BlockID, error) {
	var result eth.BlockID
	err := cl.rpc.CallContext(ctx, &result, "supervisor_finalized", chainID)
	return result, err
}

func (cl *SupervisorClient) FinalizedL1(ctx context.Context) (eth.BlockRef, error) {
	var result eth.BlockRef
	err := cl.rpc.CallContext(ctx, &result, "supervisor_finalizedL1")
	return result, err
}

func (cl *SupervisorClient) SuperRootAtTimestamp(ctx context.Context, timestamp hexutil.Uint64) (eth.SuperRootResponse, error) {
	var result eth.SuperRootResponse
	err := cl.rpc.CallContext(ctx, &result, "supervisor_superRootAtTimestamp", timestamp)
	return result, err
}

func (cl *SupervisorClient) SyncStatus(ctx context.Context) (eth.SupervisorSyncStatus, error) {
	var result eth.SupervisorSyncStatus
	err := cl.rpc.CallContext(ctx, &result, "supervisor_syncStatus")
	return result, err
}

func (cl *SupervisorClient) AllSafeDerivedAt(ctx context.Context, derivedFrom eth.BlockID) (map[eth.ChainID]eth.BlockID, error) {
	var result map[eth.ChainID]eth.BlockID
	err := cl.rpc.CallContext(ctx, &result, "supervisor_allSafeDerivedAt", derivedFrom)
	return result, err
}
