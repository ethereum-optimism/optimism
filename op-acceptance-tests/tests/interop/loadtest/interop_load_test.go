package loadtest

import (
	"context"
	"math/rand"
	"sync"
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithSimpleInterop())
}

// TestExecFromSameAddressInALoop spams transactions that each execute hundreds of initiating messages.
func TestExecFromSameAddressInALoop(gt *testing.T) {
	if testing.Short() {
		gt.Skip("skipping load test in short mode")
	}
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)
	eoaA := sys.FunderA.NewFundedEOA(eth.MillionEther)
	eoaB := sys.FunderB.NewFundedEOA(eth.MillionEther)

	eventLoggerAddress := eoaA.DeployEventLogger()

	// Create a transaction that emits numMsgs logs in a single transaction using multicall and the event logger.
	const numMsgs = 275 // About the max number of msgs we can create before hitting tx size limits.
	initMsgsTx := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](eoaA.Plan())
	initCalls := make([]txintent.Call, 0, numMsgs)
	rng := rand.New(rand.NewSource(1234))
	for range numMsgs {
		initCalls = append(initCalls, interop.RandomInitTrigger(rng, eventLoggerAddress, rng.Intn(5), rng.Intn(10)))
	}
	initMsgsTx.Content.Set(&txintent.MultiTrigger{
		Emitter: constants.MultiCall3,
		Calls:   initCalls,
	})
	initResult, err := initMsgsTx.Result.Eval(t.Ctx())
	t.Require().NoError(err)
	t.Require().Len(initResult.Entries, numMsgs)
	_, err = initMsgsTx.PlannedTx.Success.Eval(t.Ctx())
	t.Require().NoError(err)

	// Wait to include the exec txs until we know it will be included at a higher timestamp than initMsgsTx, modulo reorgs.
	// NOTE: this should be `<`, but the mempool filtering in op-geth currently uses the unsafe head's timestamp instead of
	// the pending timestamp. See https://github.com/ethereum-optimism/op-geth/issues/603.
	for sys.L2ChainB.UnsafeHeadRef().Time <= initMsgsTx.PlannedTx.IncludedBlock.Value().Time {
		sys.L2ChainB.WaitForBlock()
	}

	execCalls := make([]txintent.Call, 0, numMsgs)
	for i := range numMsgs {
		execCalls = append(execCalls, &txintent.ExecTrigger{
			Executor: constants.CrossL2Inbox,
			Msg:      initResult.Entries[i],
		})
	}
	const numExecTxs = 1_000
	execMsgsTxs := make([]*txintent.IntentTx[*txintent.MultiTrigger, txintent.Result], 0, numExecTxs)
	for i := range numExecTxs {
		execMsgsTx := txintent.NewIntent[*txintent.MultiTrigger, txintent.Result](eoaB.Plan(), txplan.WithRetryInclusion(sys.L2ELB.Escape().EthClient(), 50, retry.Exponential()))
		execMsgsTx.Content.Set(&txintent.MultiTrigger{
			Emitter: constants.MultiCall3,
			Calls:   execCalls,
		})
		if i != 0 {
			prevNonce := &execMsgsTxs[i-1].PlannedTx.Nonce
			execMsgsTx.PlannedTx.Nonce.DependOn(prevNonce)
			execMsgsTx.PlannedTx.Nonce.Fn(func(_ context.Context) (uint64, error) {
				prevNonceU64, err := prevNonce.Get()
				if err != nil {
					return 0, err
				}
				return prevNonceU64 + 1, nil
			})
		}
		execMsgsTxs = append(execMsgsTxs, execMsgsTx)
	}
	var wg sync.WaitGroup
	wg.Add(len(execMsgsTxs))
	for _, execMsgsTx := range execMsgsTxs {
		go func(tx *txplan.PlannedTx) {
			defer wg.Done()
			receipt, err := tx.Included.Eval(t.Ctx())
			t.Require().NoError(err)
			t.Require().Len(receipt.Logs, numMsgs)
			_, err = tx.Success.Eval(t.Ctx())
			t.Require().NoError(err)

			// Wait for the transaction to be cross-safe.
			includedBlock, err := tx.IncludedBlock.Eval(t.Ctx())
			t.Require().NoError(err)
			for {
				crossSafeID, err := sys.Supervisor.Escape().QueryAPI().CrossSafe(t.Ctx(), sys.L2ChainB.ChainID())
				t.Require().NoError(err)
				if includedBlock.ID().Number <= crossSafeID.Derived.Number {
					break
				}
				sys.L2ChainB.WaitForBlock()
			}
			// Sanity check that includedBlock is still in the canonical chain.
			_, err = sys.L2ELB.Escape().EthClient().BlockRefByHash(t.Ctx(), includedBlock.Hash)
			t.Require().NoError(err)
		}(execMsgsTx.PlannedTx)
	}
	wg.Wait()
}
