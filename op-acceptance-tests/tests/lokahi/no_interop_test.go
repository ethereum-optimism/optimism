package lokahi

import (
	"context"
	"slices"
	"testing"
	"time"

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

// TestLokahiWithoutInterop checks lokahi on chains that schedule no interop, which it serves as
// op-supernode does with its interop activity off: it verifies nothing and answers both heads as
// pre-activation, each node's safe head follows its local-safe head and finalizes on its own, and
// the super root at a timestamp both chains derived is made of each chain's output at it, as the
// chain's node reports it.
func TestLokahiWithoutInterop(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2LokahiWithoutInterop(t)
	t.Require().Nil(sys.L2A.Escape().RollupConfig().LagoonTime, "chain A must schedule no interop")
	t.Require().Nil(sys.L2B.Escape().RollupConfig().LagoonTime, "chain B must schedule no interop")
	nodes := []*dsl.L2CLNode{sys.L2ACL, sys.L2BCL}
	slices.SortFunc(nodes, func(a, b *dsl.L2CLNode) int { return a.ChainID().Cmp(b.ChainID()) })

	var checks []dsl.CheckFunc
	for _, node := range nodes {
		checks = append(checks, node.AdvancedFn(safety.CrossSafe, 4, 60), node.ReachedFn(safety.Finalized, 1, 120))
	}
	dsl.CheckAll(t, checks...)
	for _, node := range nodes {
		status := node.SyncStatus()
		t.Require().Equal(status.LocalSafeL2, status.SafeL2, "%s: the safe head must follow local-safe", node)
	}

	ctx, cancel := context.WithTimeout(t.Ctx(), 2*time.Minute)
	defer cancel()
	rpcClient, err := rpc.DialContext(ctx, sys.Lokahi.UserRPC())
	t.Require().NoError(err, "dial lokahi")
	defer rpcClient.Close()
	for _, node := range nodes {
		chainID, ok := node.ChainID().Uint64()
		t.Require().True(ok, "chain ID fits in 64 bits")
		status, err := chainStatus(ctx, rpcClient, chainID)
		t.Require().NoError(err, "interop_chainStatus of chain %d", chainID)
		t.Require().NotNil(status.Verified, "chain %d: lokahi answers heads at once", chainID)
		t.Require().NotNil(status.Finalized, "chain %d: lokahi answers heads at once", chainID)
		t.Require().Equal("pre-activation", status.Verified.Source, "chain %d", chainID)
		t.Require().Equal("pre-activation", status.Finalized.Source, "chain %d", chainID)
		t.Require().Empty(status.Denied, "chain %d", chainID)
	}

	timestamp := nodes[0].SyncStatus().LocalSafeL2.Time
	for _, node := range nodes[1:] {
		timestamp = min(timestamp, node.SyncStatus().LocalSafeL2.Time)
	}
	lokahi := sources.NewSuperNodeClient(client.NewBaseRPCClient(rpcClient))
	var resp eth.SuperRootAtTimestampResponse
	err = wait.For(ctx, time.Second, func() (bool, error) {
		resp, err = lokahi.SuperRootAtTimestamp(ctx, timestamp)
		if err != nil {
			t.Logger().Warn("superroot_atTimestamp failed; retrying", "timestamp", timestamp, "err", err)
			return false, nil
		}
		return resp.Data != nil, nil
	})
	t.Require().NoError(err, "lokahi must serve the super root at %d, which both chains derived", timestamp)

	outputs := make([]eth.ChainIDAndOutput, 0, len(nodes))
	var requiredL1 eth.BlockID
	for _, node := range nodes {
		own := dsl.NewOpNodeSuperRoots(node).SuperRootAtTimestamp(timestamp)
		ownOutput, ok := own.OptimisticAtTimestamp[node.ChainID()]
		t.Require().True(ok, "chain %s derived timestamp %d", node.ChainID(), timestamp)
		t.Require().Equal(ownOutput, resp.OptimisticAtTimestamp[node.ChainID()], "chain %s", node.ChainID())
		outputs = append(outputs, eth.ChainIDAndOutput{ChainID: node.ChainID(), Output: ownOutput.OutputRoot})
		if ownOutput.RequiredL1.Number > requiredL1.Number {
			requiredL1 = ownOutput.RequiredL1
		}
	}
	t.Require().Equal(eth.SuperRoot(eth.NewSuperV1(timestamp, outputs...)), resp.Data.SuperRoot,
		"the super root must be made of the nodes' outputs")
	t.Require().Equal(requiredL1, resp.Data.VerifiedRequiredL1, "the highest L1 block the outputs need")
	t.Require().NotZero(resp.CurrentL1.Number, "the current L1 block is the sources'")
	t.Require().GreaterOrEqual(resp.CurrentL1.Number, requiredL1.Number)
}
