package sources

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type SupervisorAdminAPI interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	AddL2RPC(ctx context.Context, rpc string, jwtSecret eth.Bytes32) error
}

type SupervisorQueryAPI interface {
	CheckAccessList(ctx context.Context, inboxEntries []common.Hash,
		minSafety types.SafetyLevel, executingDescriptor types.ExecutingDescriptor) error
	CrossDerivedToSource(ctx context.Context, chainID eth.ChainID, derived eth.BlockID) (derivedFrom eth.BlockRef, err error)
	LocalUnsafe(ctx context.Context, chainID eth.ChainID) (eth.BlockID, error)
	CrossSafe(ctx context.Context, chainID eth.ChainID) (types.DerivedIDPair, error)
	Finalized(ctx context.Context, chainID eth.ChainID) (eth.BlockID, error)
	FinalizedL1(ctx context.Context) (eth.BlockRef, error)
	SuperRootAtTimestamp(ctx context.Context, timestamp hexutil.Uint64) (eth.SuperRootResponse, error)
	SyncStatus(ctx context.Context) (eth.SupervisorSyncStatus, error)
	AllSafeDerivedAt(ctx context.Context, derivedFrom eth.BlockID) (derived map[eth.ChainID]eth.BlockID, err error)
}

type SupervisorClient struct {
	client client.RPC
}

// This type-check keeps the Server API and Client API in sync.
var _ SupervisorQueryAPI = (*SupervisorClient)(nil)
var _ SupervisorAdminAPI = (*SupervisorClient)(nil)

func NewSupervisorClient(client client.RPC) *SupervisorClient {
	return &SupervisorClient{
		client: client,
	}
}

func (cl *SupervisorClient) Stop(ctx context.Context) error {
	var result error
	err := cl.client.CallContext(ctx, &result, "admin_stop")
	if err != nil {
		return fmt.Errorf("failed to stop Supervisor: %w", err)
	}
	return result
}

func (cl *SupervisorClient) Start(ctx context.Context) error {
	var result error
	err := cl.client.CallContext(ctx, &result, "admin_start")
	if err != nil {
		return fmt.Errorf("failed to start Supervisor: %w", err)
	}
	return result
}

func (cl *SupervisorClient) AddL2RPC(ctx context.Context, rpc string, auth eth.Bytes32) error {
	var result error
	err := cl.client.CallContext(ctx, &result, "admin_addL2RPC", rpc, auth)
	if err != nil {
		return fmt.Errorf("failed to Add L2 to Supervisor (rpc: %s): %w", rpc, err)
	}
	return result
}

func (cl *SupervisorClient) CheckAccessList(ctx context.Context, inboxEntries []common.Hash,
	minSafety types.SafetyLevel, executingDescriptor types.ExecutingDescriptor) error {
	return cl.client.CallContext(ctx, nil, "supervisor_checkAccessList", inboxEntries, minSafety, executingDescriptor)
}

func (cl *SupervisorClient) CrossDerivedToSource(ctx context.Context, chainID eth.ChainID, derived eth.BlockID) (derivedFrom eth.BlockRef, err error) {
	var result eth.BlockRef
	err = cl.client.CallContext(ctx, &result, "supervisor_crossDerivedToSource", chainID, derived)
	return result, err
}

func (cl *SupervisorClient) LocalUnsafe(ctx context.Context, chainID eth.ChainID) (eth.BlockID, error) {
	var result eth.BlockID
	err := cl.client.CallContext(ctx, &result, "supervisor_localUnsafe", chainID)
	return result, err
}

func (cl *SupervisorClient) CrossSafe(ctx context.Context, chainID eth.ChainID) (types.DerivedIDPair, error) {
	var result types.DerivedIDPair
	err := cl.client.CallContext(ctx, &result, "supervisor_crossSafe", chainID)
	return result, err
}

func (cl *SupervisorClient) Finalized(ctx context.Context, chainID eth.ChainID) (eth.BlockID, error) {
	var result eth.BlockID
	err := cl.client.CallContext(ctx, &result, "supervisor_finalized", chainID)
	return result, err
}

func (cl *SupervisorClient) FinalizedL1(ctx context.Context) (eth.BlockRef, error) {
	var result eth.BlockRef
	err := cl.client.CallContext(ctx, &result, "supervisor_finalizedL1")
	return result, err
}

func (cl *SupervisorClient) CrossDerivedFrom(ctx context.Context, chainID eth.ChainID, derived eth.BlockID) (eth.BlockRef, error) {
	var result eth.BlockRef
	err := cl.client.CallContext(ctx, &result, "supervisor_crossDerivedFrom", chainID, derived)
	return result, err
}

func (cl *SupervisorClient) UpdateLocalUnsafe(ctx context.Context, chainID eth.ChainID, head eth.BlockRef) error {
	return cl.client.CallContext(ctx, nil, "supervisor_updateLocalUnsafe", chainID, head)
}

func (cl *SupervisorClient) UpdateLocalSafe(ctx context.Context, chainID eth.ChainID, derivedFrom eth.L1BlockRef, lastDerived eth.BlockRef) error {
	return cl.client.CallContext(ctx, nil, "supervisor_updateLocalSafe", chainID, derivedFrom, lastDerived)
}

// SuperRootAtTimestamp returns the super root at the specified timestamp.
// Returns ethereum.NotFound if one of the chain's has not yet reached the block required for the requested super root.
func (cl *SupervisorClient) SuperRootAtTimestamp(ctx context.Context, timestamp hexutil.Uint64) (eth.SuperRootResponse, error) {
	var result eth.SuperRootResponse
	err := cl.client.CallContext(ctx, &result, "supervisor_superRootAtTimestamp", timestamp)
	if isNotFound(err) {
		// Downstream users expect to get a properly typed error message for not found.
		return result, fmt.Errorf("%w: %v", ethereum.NotFound, err.Error())
	}
	return result, err
}

func (cl *SupervisorClient) AllSafeDerivedAt(ctx context.Context, derivedFrom eth.BlockID) (map[eth.ChainID]eth.BlockID, error) {
	var result map[eth.ChainID]eth.BlockID
	err := cl.client.CallContext(ctx, &result, "supervisor_allSafeDerivedAt", derivedFrom)
	return result, err
}

func (cl *SupervisorClient) SyncStatus(ctx context.Context) (eth.SupervisorSyncStatus, error) {
	var result eth.SupervisorSyncStatus
	err := cl.client.CallContext(ctx, &result, "supervisor_syncStatus")
	return result, err
}

func (cl *SupervisorClient) Close() {
	cl.client.Close()
}

func isNotFound(err error) bool {
	// The RPC server wil convert the returned error to a string so we can't match on an error type here
	return err != nil && strings.Contains(err.Error(), ethereum.NotFound.Error())
}
