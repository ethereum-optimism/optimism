package superfaultproofs

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/atomic"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// The paused devnet first executes a deposit-only prefix block. Its retained
// state includes the exact system updates. Replacing it with a sibling that adds
// the returned envelopes exercises full op-reth execution and normal interop
// verification. This fixture does not expose a production payload-building API.
func buildSuspendedAtomic(t devtest.T, sys *presets.SimpleInterop, setup *intraBlockSetup, router, appAddr common.Address, data []byte, sponsors map[eth.ChainID]*sponsoredAccount) (map[eth.ChainID]*txplan.PlannedTx, *atomic.SuspendedResult) {
	binary, err := (rustbin.Spec{SrcDir: "rust", Package: "op-atomic-builder", Binary: "op-atomic-builder"}).EnsureExists(t.Ctx(), t.Logger())
	t.Require().NoError(err)
	var chains []atomic.SuspendedChain
	owners := make(map[uint64]*dsl.EOA)
	for _, item := range []struct {
		eoa    *dsl.EOA
		el     *dsl.L2ELNode
		number uint64
	}{{setup.alice, sys.L2ELA, setup.expectedBlockNumA}, {setup.bob, sys.L2ELB, setup.expectedBlockNumB}} {
		id := item.eoa.ChainID()
		n := bigs.Uint64Strict(id.ToBig())
		owners[n] = item.eoa
		parent := item.el.BlockRefByLabel(eth.Unsafe)
		nonce := item.eoa.PendingNonce()
		sys.TestSequencer.SequenceBlock(t, id, parent.Hash)
		prefix := item.el.BlockRefByLabel(eth.Unsafe)
		t.Require().Equal(item.number, prefix.Number)
		t.Require().Equal(setup.nextTimestamp, prefix.Time)
		var header *types.Header
		t.Require().NoError(item.el.EthClient().RPC().CallContext(t.Ctx(), &header, "eth_getBlockByHash", prefix.Hash, false))
		t.Require().NotNil(header)
		t.Require().True(header.BaseFee.IsUint64())
		t.Require().NotNil(header.ExcessBlobGas)
		t.Require().Zero(*header.ExcessBlobGas)
		// This exact prefix must not contain pool/user transactions or emitted logs.
		var block struct {
			Transactions []common.Hash `json:"transactions"`
		}
		t.Require().NoError(item.el.EthClient().RPC().CallContext(t.Ctx(), &block, "eth_getBlockByHash", prefix.Hash, false))
		for _, hash := range block.Transactions {
			var tx *types.Transaction
			t.Require().NoError(item.el.EthClient().RPC().CallContext(t.Ctx(), &tx, "eth_getTransactionByHash", hash))
			t.Require().NotNil(tx)
			t.Require().True(tx.Type() == types.DepositTxType || tx.Type() == types.PostExecTxType, "prefix must contain only protocol transactions")
			receipt, err := item.el.EthClient().TransactionReceipt(t.Ctx(), hash)
			t.Require().NoError(err)
			t.Require().Empty(receipt.Logs)
		}
		snapshot, err := atomic.NewRPCPrefixSnapshot(item.el.EthClient().RPC(), header)
		t.Require().NoError(err)
		// The proof-history index follows canonical execution asynchronously.
		// Wait for the pinned proof before starting the one discovery attempt.
		t.Require().Eventually(func() bool {
			_, err := snapshot.Account(t.Ctx(), router)
			if err != nil {
				t.Logger().Debug("Waiting for candidate prefix proof", "chain", id, "prefix", prefix.Hash, "err", err)
			}
			return err == nil
		}, 30*time.Second, 200*time.Millisecond, "candidate prefix proof must be available")
		executor := sponsors[id].executor(&atomic.RPCExecutor{RPC: item.el.EthClient().RPC(), Parent: parent.Hash}, item.eoa)
		chains = append(chains, atomic.SuspendedChain{Chain: n, Router: router, Sender: executor.Account, ApplicationGas: 2_000_000,
			Spec: "LAGOON", Number: prefix.Number, Timestamp: prefix.Time, GasLimit: header.GasLimit, PrefixGas: header.GasUsed,
			BaseFee: bigs.Uint64Strict(header.BaseFee), Coinbase: header.Coinbase, PrevRandao: header.MixDigest, Snapshot: snapshot,
			Prepare: func(ctx context.Context, data []byte, accesses types.AccessList) (*types.Transaction, error) {
				_, tx, err := executor.Prepare(ctx, atomic.Transaction{From: executor.Account, To: router, Data: data, Gas: 8_000_000, AccessList: accesses})
				if err != nil {
					return nil, err
				}
				return types.SignNewTx(item.eoa.Key().Priv(), types.LatestSignerForChainID(id.ToBig()), &types.DynamicFeeTx{ChainID: id.ToBig(), Nonce: nonce, To: &tx.To, Data: tx.Data, Gas: tx.Gas, GasFeeCap: tx.GasFeeCap, GasTipCap: tx.GasTipCap, AccessList: tx.AccessList})
			}})
	}
	result, err := atomic.BuildSuspended(t.Ctx(), binary, chains, bigs.Uint64Strict(setup.alice.ChainID().ToBig()), 0, appAddr, data, 8)
	t.Require().NoError(err)
	planned := make(map[eth.ChainID]*txplan.PlannedTx)
	for n, inclusion := range result.Included {
		owner := owners[n]
		tx := fixedAtomicPlan(inclusion.Transaction, owner.Plan(), txintent.WithoutInteropDependencyWait())
		planned[owner.ChainID()] = tx
	}
	return planned, result
}

// Preserve the exact replayed bytes when the generic inclusion helper sets nonces.
func fixedAtomicPlan(signed *types.Transaction, options ...txplan.Option) *txplan.PlannedTx {
	tx := txplan.NewPlannedTx(options...)
	tx.Signed.ResetFnAndDependencies()
	tx.Signed.Set(signed)
	return tx
}
