package sources

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type SupervisorClient struct {
	client client.RPC
}

// This type-check keeps the Server API and Client API in sync.
var _ apis.SupervisorAPI = (*SupervisorClient)(nil)

func NewSupervisorClient(client client.RPC) *SupervisorClient {
	return &SupervisorClient{
		client: client,
	}
}

func (cl *SupervisorClient) Stop(ctx context.Context) error {
	err := cl.client.CallContext(ctx, nil, "admin_stop")
	if err != nil {
		return fmt.Errorf("failed to stop Supervisor: %w", err)
	}
	return nil
}

func (cl *SupervisorClient) Start(ctx context.Context) error {
	err := cl.client.CallContext(ctx, nil, "admin_start")
	if err != nil {
		return fmt.Errorf("failed to start Supervisor: %w", err)
	}
	return nil
}

func (cl *SupervisorClient) AddL2RPC(ctx context.Context, rpc string, auth eth.Bytes32) error {
	err := cl.client.CallContext(ctx, nil, "admin_addL2RPC", rpc, auth)
	if err != nil {
		return fmt.Errorf("failed to Add L2 to Supervisor (rpc: %s): %w", rpc, err)
	}
	return nil
}

func (cl *SupervisorClient) Rewind(ctx context.Context, chain eth.ChainID, block eth.BlockID) error {
	return cl.client.CallContext(ctx, nil, "admin_rewind", chain, block)
}

func (cl *SupervisorClient) SetFailsafeEnabled(ctx context.Context, enabled bool) error {
	err := cl.client.CallContext(ctx, nil, "admin_setFailsafeEnabled", enabled)
	if err != nil {
		return fmt.Errorf("failed to set failsafe mode for Supervisor: %w", err)
	}
	return nil
}

func (cl *SupervisorClient) GetFailsafeEnabled(ctx context.Context) (bool, error) {
	var enabled bool
	err := cl.client.CallContext(ctx, &enabled, "admin_getFailsafeEnabled")
	if err != nil {
		return false, fmt.Errorf("failed to get failsafe mode for Supervisor: %w", err)
	}
	return enabled, nil
}

func (cl *SupervisorClient) CheckAccessList(ctx context.Context, inboxEntries []common.Hash,
	minSafety types.SafetyLevel, executingDescriptor types.ExecutingDescriptor) error {
	return cl.client.CallContext(ctx, nil, "supervisor_checkAccessList", inboxEntries, minSafety, executingDescriptor)
}

func (cl *SupervisorClient) CrossDerivedToSource(ctx context.Context, chainID eth.ChainID, derived eth.BlockID) (derivedFrom eth.BlockRef, err error) {
	err = cl.client.CallContext(ctx, &derivedFrom, "supervisor_crossDerivedToSource", chainID, derived)
	return derivedFrom, err
}

func (cl *SupervisorClient) LocalUnsafe(ctx context.Context, chainID eth.ChainID) (result eth.BlockID, err error) {
	err = cl.client.CallContext(ctx, &result, "supervisor_localUnsafe", chainID)
	return result, err
}

func (cl *SupervisorClient) LocalSafe(ctx context.Context, chainID eth.ChainID) (result types.DerivedIDPair, err error) {
	err = cl.client.CallContext(ctx, &result, "supervisor_localSafe", chainID)
	return result, err
}

func (cl *SupervisorClient) CrossSafe(ctx context.Context, chainID eth.ChainID) (result types.DerivedIDPair, err error) {
	err = cl.client.CallContext(ctx, &result, "supervisor_crossSafe", chainID)
	return result, err
}

func (cl *SupervisorClient) Finalized(ctx context.Context, chainID eth.ChainID) (result eth.BlockID, err error) {
	err = cl.client.CallContext(ctx, &result, "supervisor_finalized", chainID)
	return result, err
}

func (cl *SupervisorClient) FinalizedL1(ctx context.Context) (result eth.BlockRef, err error) {
	err = cl.client.CallContext(ctx, &result, "supervisor_finalizedL1")
	return result, err
}

func (cl *SupervisorClient) UpdateLocalUnsafe(ctx context.Context, chainID eth.ChainID, head eth.BlockRef) (err error) {
	err = cl.client.CallContext(ctx, nil, "supervisor_updateLocalUnsafe", chainID, head)
	return err
}

func (cl *SupervisorClient) UpdateLocalSafe(ctx context.Context, chainID eth.ChainID, derivedFrom eth.L1BlockRef, lastDerived eth.BlockRef) (err error) {
	err = cl.client.CallContext(ctx, nil, "supervisor_updateLocalSafe", chainID, derivedFrom, lastDerived)
	return err
}

// SuperRootAtTimestamp returns the super root at the specified timestamp.
// Returns ethereum.NotFound if one of the chain's has not yet reached the block required for the requested super root.
func (cl *SupervisorClient) SuperRootAtTimestamp(ctx context.Context, timestamp hexutil.Uint64) (result eth.SuperRootResponse, err error) {
	err = cl.client.CallContext(ctx, &result, "supervisor_superRootAtTimestamp", timestamp)
	if isNotFound(err) {
		// Downstream users expect to get a properly typed error message for not found.
		err = fmt.Errorf("%w: %v", ethereum.NotFound, err.Error())
		return result, err
	}
	return result, err
}

func (cl *SupervisorClient) AllSafeDerivedAt(ctx context.Context, derivedFrom eth.BlockID) (result map[eth.ChainID]eth.BlockID, err error) {
	err = cl.client.CallContext(ctx, &result, "supervisor_allSafeDerivedAt", derivedFrom)
	return result, err
}

func (cl *SupervisorClient) SyncStatus(ctx context.Context) (result eth.SupervisorSyncStatus, err error) {
	err = cl.client.CallContext(ctx, &result, "supervisor_syncStatus")
	return result, err
}

func (cl *SupervisorClient) Close() {
	cl.client.Close()
}

func isNotFound(err error) bool {
	// The RPC server will convert the returned error to a string so we can't match on an error type here
	return err != nil && strings.Contains(err.Error(), ethereum.NotFound.Error())
}
