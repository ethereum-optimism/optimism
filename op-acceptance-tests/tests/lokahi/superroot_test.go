package lokahi

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// TestLokahiServesSuperRoots checks lokahi's op-supernode RPC through the Go client proofs
// tooling uses: the super root of a verified timestamp is made of each chain's output at it, as
// the chain's kona-node reports it, the sync status reports both chains, the heartbeat answers,
// and the test control is not served without its flag.
func TestLokahiServesSuperRoots(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2Lokahi(t)
	nodes := []*dsl.L2CLNode{sys.L2ACL, sys.L2BCL}
	slices.SortFunc(nodes, func(a, b *dsl.L2CLNode) int { return a.ChainID().Cmp(b.ChainID()) })
	chainIDs := []eth.ChainID{nodes[0].ChainID(), nodes[1].ChainID()}

	ctx, cancel := context.WithTimeout(t.Ctx(), 3*time.Minute)
	defer cancel()
	rpcClient, err := rpc.DialContext(ctx, sys.Lokahi.UserRPC())
	t.Require().NoError(err, "dial lokahi")
	defer rpcClient.Close()
	lokahi := sources.NewSuperNodeClient(client.NewBaseRPCClient(rpcClient))

	// A timestamp after genesis that both chains derived already, which lokahi verifies.
	dsl.CheckAll(t, sys.L2ACL.AdvancedFn(safety.LocalSafe, 4, 60), sys.L2BCL.AdvancedFn(safety.LocalSafe, 4, 60))
	var timestamp uint64
	for _, node := range nodes {
		timestamp = max(timestamp, node.SyncStatus().LocalSafeL2.Time)
	}
	var resp eth.SuperRootAtTimestampResponse
	err = wait.For(ctx, time.Second, func() (bool, error) {
		resp, err = lokahi.SuperRootAtTimestamp(ctx, timestamp)
		if err != nil {
			t.Logger().Warn("superroot_atTimestamp failed; retrying", "timestamp", timestamp, "err", err)
			return false, nil
		}
		return resp.Data != nil, nil
	})
	t.Require().NoError(err, "lokahi did not verify timestamp %d", timestamp)

	t.Require().Equal(chainIDs, resp.ChainIDs)
	t.Require().GreaterOrEqual(resp.CurrentL1.Number, resp.Data.VerifiedRequiredL1.Number)
	super, ok := resp.Data.Super.(*eth.SuperV1)
	t.Require().True(ok, "a version 1 super root")
	t.Require().Equal(timestamp, super.Timestamp)
	t.Require().Equal(eth.SuperRoot(super), resp.Data.SuperRoot)
	t.Require().Len(super.Chains, len(nodes))
	for i, node := range nodes {
		own := dsl.NewOpNodeSuperRoots(node).SuperRootAtTimestamp(timestamp)
		ownOutput, ok := own.OptimisticAtTimestamp[node.ChainID()]
		t.Require().True(ok, "chain %s derived timestamp %d", node.ChainID(), timestamp)
		t.Require().Equal(node.ChainID(), super.Chains[i].ChainID)
		t.Require().Equal(ownOutput.OutputRoot, super.Chains[i].Output, "chain %s", node.ChainID())
		t.Require().Equal(ownOutput, resp.OptimisticAtTimestamp[node.ChainID()], "chain %s", node.ChainID())
	}

	status, err := lokahi.SyncStatus(ctx)
	t.Require().NoError(err, "supernode_syncStatus")
	t.Require().Equal(chainIDs, status.ChainIDs)
	for _, chainID := range chainIDs {
		chain, ok := status.Chains[chainID]
		t.Require().True(ok, "sync status of chain %s", chainID)
		t.Require().NotZero(chain.LocalSafeL2.Number, "chain %s", chainID)
		t.Require().LessOrEqual(status.SafeTimestamp, chain.SafeL2.Time, "chain %s", chainID)
		t.Require().LessOrEqual(status.LocalSafeTimestamp, chain.LocalSafeL2.Time, "chain %s", chainID)
		t.Require().LessOrEqual(status.FinalizedTimestamp, chain.FinalizedL2.Time, "chain %s", chainID)
		t.Require().LessOrEqual(status.CurrentL1.Number, chain.CurrentL1.Number, "chain %s", chainID)
	}
	t.Require().GreaterOrEqual(status.LocalSafeTimestamp, timestamp)

	var beat hexutil.Bytes
	t.Require().NoError(rpcClient.CallContext(ctx, &beat, "heartbeat_check"))
	t.Require().Len(beat, 4)

	err = rpcClient.CallContext(ctx, nil, "lokahitest_pauseAt", timestamp)
	var rpcErr rpc.Error
	t.Require().True(errors.As(err, &rpcErr), "the test control is not served: %v", err)
	t.Require().Equal(-32601, rpcErr.ErrorCode(), "method not found")
}
