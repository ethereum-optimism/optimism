package loadtest

import (
	"context"
	"math/rand"
	"sync"
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/crypto"
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

	// Use proxyd if available.
	l2ELA := sys.L2ELA
	l2ELB := sys.L2ELB
	if proxydsA := match.Proxyd.Match(sys.L2ChainA.Escape().L2ELNodes()); len(proxydsA) > 0 {
		l2ELA = dsl.NewL2ELNode(proxydsA[0])
	}
	if proxydsB := match.Proxyd.Match(sys.L2ChainB.Escape().L2ELNodes()); len(proxydsB) > 0 {
		l2ELB = dsl.NewL2ELNode(proxydsB[0])
	}

	// Setup EOAs. We are careful to use the specific ELs configured above to ensure we
	// hit proxyd (if configured) to simulate real load.
	skA, err := crypto.GenerateKey()
	t.Require().NoError(err)
	skB, err := crypto.GenerateKey()
	t.Require().NoError(err)
	eoaA := dsl.NewEOA(dsl.NewKey(t, skA), l2ELA)
	eoaB := dsl.NewEOA(dsl.NewKey(t, skB), l2ELB)
	sys.FaucetA.Fund(eoaA.Address(), eth.MillionEther)
	sys.FaucetB.Fund(eoaB.Address(), eth.MillionEther)

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
	for l2ELA.BlockRefByLabel(eth.Unsafe).Time <= initMsgsTx.PlannedTx.IncludedBlock.Value().Time {
		l2ELA.WaitForBlock()
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
		execMsgsTx := txintent.NewIntent[*txintent.MultiTrigger, txintent.Result](eoaB.Plan(), txplan.WithRetryInclusion(l2ELB.Escape().EthClient(), 50, retry.Exponential()))
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
				// NOTE: it may be desirable to query proxyd instead of the supervisor if/when the devstack supports it.
				crossSafeID, err := sys.Supervisor.Escape().QueryAPI().CrossSafe(t.Ctx(), l2ELB.ChainID())
				t.Require().NoError(err)
				if includedBlock.ID().Number <= crossSafeID.Derived.Number {
					break
				}
				l2ELB.WaitForBlock()
			}
			// Sanity check that includedBlock is still in the canonical chain.
			_, err = l2ELB.Escape().EthClient().BlockRefByHash(t.Ctx(), includedBlock.Hash)
			t.Require().NoError(err)
		}(execMsgsTx.PlannedTx)
	}
	wg.Wait()
}
