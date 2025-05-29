package loadtest

import (
	"math"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/plan"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/net/context"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithSimpleInterop())
}

// TestLoad passes messages from one chain to another.
// It assumes that both chains have the same block time.
// Set NAT_INTEROP_LOADTEST_TARGET to the initial amount of messages that should be passed per block time.
// The test will run until the test deadline.
func TestLoad(gt *testing.T) {
	if testing.Short() {
		gt.Skip("skipping load test in short mode")
	}
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)

	targetMessagePassesPerBlock := uint64(100)
	if targetMsgPassesStr, exists := os.LookupEnv("NAT_INTEROP_LOADTEST_TARGET"); exists {
		var err error
		targetMessagePassesPerBlock, err = strconv.ParseUint(targetMsgPassesStr, 10, 0)
		t.Require().NoError(err)
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	// Start the AIMD schedule.
	deadline := time.Unix(math.MaxInt64, 0)
	testCtxDeadline, testCtxDeadlineExsts := t.Ctx().Deadline()
	if testCtxDeadlineExsts {
		deadline = testCtxDeadline.Add(-10 * time.Second) // Give some time for cleanup.
	}
	schedCtx, schedCancel := context.WithDeadline(t.Ctx(), deadline)
	t.Cleanup(schedCancel)
	blockTime := time.Duration(sys.L2ChainB.Escape().RollupConfig().BlockTime) * time.Second
	aimd := NewAIMD(targetMessagePassesPerBlock, blockTime, WithAdjustWindow(targetMessagePassesPerBlock/2))
	wg.Add(1)
	go func() {
		defer wg.Done()
		aimd.Start(schedCtx)
	}()

	workerCount := targetMessagePassesPerBlock * 7

	l2ELA := sys.L2ChainA.PublicRPC()
	l2ELB := sys.L2ChainB.PublicRPC()
	funderA := dsl.NewFunder(sys.Wallet, sys.FaucetA, l2ELA)
	source := &L2{
		EOAs:        NewEOAPool(funderA, workerCount, eth.MillionEther),
		EL:          l2ELA,
		EventLogger: funderA.NewFundedEOA(eth.OneEther).DeployEventLogger(),
	}
	dest := &L2{
		EOAs: NewEOAPool(dsl.NewFunder(sys.Wallet, sys.FaucetB, l2ELB), workerCount, eth.MillionEther),
		EL:   l2ELB,
	}

	// Start the message passing workers.
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			relayTxsWhenReady(t, aimd, source, dest)
		}()
	}
}

type L2 struct {
	EOAs        *EOAPool
	EL          *dsl.L2ELNode
	EventLogger common.Address
}

func relayTxsWhenReady(t devtest.T, sched *AIMD, source *L2, dest *L2) {
	rng := rand.New(rand.NewSource(1234))
	for range sched.Ready() {
		initiator := source.EOAs.Get()
		initMsgTx := txintent.NewIntent[txintent.Call, *txintent.InteropOutput](initiator.Inner.Plan(), txplan.WithStaticNonce(uint64(initiator.Nonce.Add(1))-1))
		initMsgTx.Content.Set(interop.RandomInitTrigger(rng, source.EventLogger, rng.Intn(2), rng.Intn(5)))
		if _, err := initMsgTx.PlannedTx.Included.Eval(t.Ctx()); err != nil {
			initiator.Nonce.Add(-1)
			sched.Adjust(false)
			continue
		}
		_, err := initMsgTx.PlannedTx.Success.Eval(t.Ctx())
		t.Require().NoError(err)
		out, err := initMsgTx.Result.Eval(t.Ctx())
		t.Require().NoError(err)
		t.Require().Len(out.Entries, 1)
		initMsg := out.Entries[0]

		executor := dest.EOAs.Get()
		execTx := txintent.NewIntent[*txintent.ExecTrigger, txintent.Result](executor.Inner.Plan(), txplan.WithStaticNonce(uint64(executor.Nonce.Add(1))-1), txplan.WithGasRatio(2))
		execTx.Content.Set(&txintent.ExecTrigger{
			Executor: constants.CrossL2Inbox,
			Msg:      initMsg,
		})

		// The relay tx is invalid until we know it will be included at a higher timestamp than any of the initiating messages, modulo reorgs.
		// Wait to plan the relay tx against a target block until the timestamp elapses.
		// NOTE: this should be `<`, but the mempool filtering in op-geth currently uses the unsafe head's timestamp instead of
		// the pending timestamp. See https://github.com/ethereum-optimism/op-geth/issues/603.
		execTx.PlannedTx.AgainstBlock.Wrap(func(fn plan.Fn[eth.BlockInfo]) plan.Fn[eth.BlockInfo] {
			for dest.EL.BlockRefByLabel(eth.Unsafe).Time <= initMsg.Identifier.Timestamp {
				dest.EL.WaitForBlock()
			}
			return fn
		})
		if _, err := execTx.PlannedTx.Included.Eval(t.Ctx()); err != nil {
			executor.Nonce.Add(-1)
			sched.Adjust(false)
			continue
		}
		_, err = execTx.PlannedTx.Success.Eval(t.Ctx())
		t.Require().NoError(err)
		sched.Adjust(true)
	}
}
