package supernode

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/internal/syncstatus"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum/go-ethereum/common"
	gethlog "github.com/ethereum/go-ethereum/log"
)

var _ activity.RPCActivity = (*Activity)(nil)

type Activity struct {
	log    gethlog.Logger
	chains map[eth.ChainID]cc.ChainContainer
}

func New(log gethlog.Logger, chains map[eth.ChainID]cc.ChainContainer) *Activity {
	return &Activity{
		log:    log,
		chains: chains,
	}
}

func (a *Activity) Name() string { return "supernode" }

func (a *Activity) Reset(chainID eth.ChainID, timestamp uint64, invalidatedBlock eth.BlockRef) {
	// No-op: sync status queries chain containers directly.
}

func (a *Activity) RPCNamespace() string    { return "supernode" }
func (a *Activity) RPCService() interface{} { return &api{a: a} }

type api struct{ a *Activity }

// SyncStatus returns all the per-node SyncStatus responses and computes the current localsafe/safe/finalized timestamps.
func (api *api) SyncStatus(ctx context.Context) (eth.SuperNodeSyncStatusResponse, error) {
	return api.a.syncStatus(ctx)
}

// IsDenied exposes the durable, read-only SuperAuthority verdict for an execution service that
// maintains its own proof index. It does not mutate the chain or initiate a rewind.
func (api *api) IsDenied(chainID eth.ChainID, blockNumber uint64, payloadHash common.Hash) (bool, error) {
	chain, ok := api.a.chains[chainID]
	if !ok {
		return false, fmt.Errorf("unknown chain %s", chainID)
	}
	return chain.IsDenied(blockNumber, payloadHash)
}

func (a *Activity) syncStatus(ctx context.Context) (eth.SuperNodeSyncStatusResponse, error) {
	return syncstatus.Aggregate(ctx, a.log, a.chains)
}
