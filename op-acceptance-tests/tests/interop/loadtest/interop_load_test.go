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
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
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

	// Start metrics collector
	metricsCollector := NewMetricsCollector(blockTime)
	wg.Add(1)
	go func() {
		defer wg.Done()
		t.Require().NoError(metricsCollector.Start(schedCtx))
	}()
	t.Cleanup(func() {
		const artifactsDir = "artifacts"
		t.Require().NoError(os.MkdirAll(artifactsDir, 0755))
		t.Require().NoError(metricsCollector.SaveGraph(artifactsDir))
	})

	workerCount := targetMessagePassesPerBlock * 7

	l2ELA := sys.L2ChainA.PublicRPC()
	l2ELB := sys.L2ChainB.PublicRPC()
	funderA := dsl.NewFunder(sys.Wallet, sys.FaucetA, l2ELA)
	numEOAs := min(workerCount, 300)
	source := &L2{
		EOAs:        NewEOAPool(funderA, numEOAs, eth.MillionEther),
		EL:          l2ELA,
		EventLogger: funderA.NewFundedEOA(eth.OneEther).DeployEventLogger(),
	}
	dest := &L2{
		EOAs: NewEOAPool(dsl.NewFunder(sys.Wallet, sys.FaucetB, l2ELB), numEOAs, eth.MillionEther),
		EL:   l2ELB,
	}

	// Start the message passing workers.
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rng := rand.New(rand.NewSource(1234))
			for range aimd.Ready() {
				inFlightMessages.Inc()
				start := time.Now()
				initMsg := source.SendInitiatingMsg(t, rng)
				if initMsg == nil {
					messageStatusCount.WithLabelValues("init_failed").Inc()
					inFlightMessages.Dec()
					aimd.Adjust(false)
					continue
				}
				success := dest.SendExecutingMsg(t, *initMsg)
				if success {
					messageLatency.WithLabelValues("e2e").Observe(time.Since(start).Seconds())
					messageStatusCount.WithLabelValues("success").Inc()
				} else {
					messageStatusCount.WithLabelValues("exec_failed").Inc()
				}
				inFlightMessages.Dec()
				aimd.Adjust(success)
			}
		}()
	}
}

type L2 struct {
	EOAs        *EOAPool
	EL          *dsl.L2ELNode
	EventLogger common.Address
}

func (l2 *L2) SendInitiatingMsg(t devtest.T, rng *rand.Rand) *types.Message {
	start := time.Now()
	eoa := l2.EOAs.Get()
	tx := txintent.NewIntent[txintent.Call, *txintent.InteropOutput](eoa.Inner.Plan(), txplan.WithStaticNonce(uint64(eoa.Nonce.Add(1))-1))
	tx.Content.Set(interop.RandomInitTrigger(rng, l2.EventLogger, rng.Intn(2), rng.Intn(5)))
	if _, err := tx.PlannedTx.Included.Eval(t.Ctx()); err != nil {
		eoa.Nonce.Add(-1)
		return nil
	}
	_, err := tx.PlannedTx.Success.Eval(t.Ctx())
	t.Require().NoError(err)
	messageLatency.WithLabelValues("init").Observe(time.Since(start).Seconds())
	out, err := tx.Result.Eval(t.Ctx())
	t.Require().NoError(err)
	t.Require().Len(out.Entries, 1)
	return &out.Entries[0]
}

func (l2 *L2) SendExecutingMsg(t devtest.T, initMsg types.Message) bool {
	start := time.Now()
	eoa := l2.EOAs.Get()
	tx := txintent.NewIntent[*txintent.ExecTrigger, txintent.Result](eoa.Inner.Plan(), txplan.WithStaticNonce(uint64(eoa.Nonce.Add(1))-1), txplan.WithGasRatio(2))
	tx.Content.Set(&txintent.ExecTrigger{
		Executor: constants.CrossL2Inbox,
		Msg:      initMsg,
	})
	// The tx is invalid until we know it will be included at a higher timestamp than any of the initiating messages, modulo reorgs.
	// Wait to plan the relay tx against a target block until the timestamp elapses.
	// NOTE: this should be `<`, but the mempool filtering in op-geth currently uses the unsafe head's timestamp instead of
	// the pending timestamp. See https://github.com/ethereum-optimism/op-geth/issues/603.
	tx.PlannedTx.AgainstBlock.Wrap(func(fn plan.Fn[eth.BlockInfo]) plan.Fn[eth.BlockInfo] {
		for l2.EL.BlockRefByLabel(eth.Unsafe).Time <= initMsg.Identifier.Timestamp {
			l2.EL.WaitForBlock()
		}
		return fn
	})
	if _, err := tx.PlannedTx.Included.Eval(t.Ctx()); err != nil {
		eoa.Nonce.Add(-1)
		return false
	}
	_, err := tx.PlannedTx.Success.Eval(t.Ctx())
	t.Require().NoError(err)
	messageLatency.WithLabelValues("exec").Observe(time.Since(start).Seconds())
	return true
}
