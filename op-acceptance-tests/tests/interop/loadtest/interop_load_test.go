package loadtest

import (
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/accounting"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/flags"
	"github.com/ethereum-optimism/optimism/op-service/log/logfilter"
	"github.com/ethereum-optimism/optimism/op-service/plan"
	"github.com/ethereum-optimism/optimism/op-service/txinclude"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// Override this with the env var NAT_STEADY_TIMEOUT.
const defaultSteadyTestTimeout = time.Minute * 3

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithSimpleInterop(),
		presets.WithLogFilter(
			logfilter.DefaultMute(
				logfilter.Level(slog.LevelWarn).Show(),
			),
		),
	)
}

// TestSteady attempts to approach but not exceed the gas target in every block by spamming interop
// messages, simulating benign but heavy activity. The test will exit successfully after the global
// go test deadline or the timeout specified by the NAT_STEADY_TIMEOUT environment variable
// elapses, whichever comes first. Also see: https://github.com/golang/go/issues/48157.
func TestSteady(gt *testing.T) {
	t := setupT(gt)
	t, ctx, cancel := setupTestDeadline(t, "NAT_STEADY_TIMEOUT")

	var wg sync.WaitGroup
	defer wg.Wait()

	// The scheduler will adjust every slot to stay within 95-100% of the gas target.
	aimd, source, dest := setupLoadTest(t, ctx, &wg, WithAdjustWindow(1), WithDecreaseFactor(0.95))

	elasticityMultiplier := dest.Config.ElasticityMultiplier()
	wg.Add(1)
	go func() {
		defer wg.Done()
		blockTime := time.Duration(dest.RollupConfig.BlockTime) * time.Second
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(blockTime):
				unsafe, err := dest.EL.Escape().EthClient().InfoByLabel(ctx, eth.Unsafe)
				if err != nil {
					if isBenignCancellationError(err) {
						return
					}
					t.Require().NoError(err)
				}
				gasTarget := unsafe.GasLimit() / elasticityMultiplier
				// Apply backpressure when we meet or exceed the gas target.
				aimd.Adjust(unsafe.GasUsed() < gasTarget)
			}
		}
	}()

	for range aimd.Ready() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var overdraft *accounting.OverdraftError
			if err := relayMessage(ctx, t, source, dest); errors.As(err, &overdraft) {
				cancel()
				t.Require().NoError(err)
			}
		}()
	}
}

// TestBurst spams interop messages and exits successfully when the budget is depleted, simulating
// adversarial behavior. The test will exit successfully after the global go test deadline or the
// timeout specified by the NAT_BURST_TIMEOUT environment variable elapses, whichever comes first.
// Also see: https://github.com/golang/go/issues/48157.
func TestBurst(gt *testing.T) {
	t := setupT(gt)
	t, ctx, cancel := setupTestDeadline(t, "NAT_BURST_TIMEOUT")

	var wg sync.WaitGroup
	defer wg.Wait()
	aimd, source, dest := setupLoadTest(t, ctx, &wg)
	for range aimd.Ready() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := relayMessage(ctx, t, source, dest)
			if err == nil {
				aimd.Adjust(true)
				return
			}
			var overdraft *accounting.OverdraftError
			if errors.As(err, &overdraft) {
				cancel()
			}
			aimd.Adjust(false)
		}()
	}
}

func setupT(t *testing.T) devtest.T {
	if testing.Short() || !flags.ReadTestConfig().EnableLoadTests {
		t.Skip("skipping load test in short mode or if load tests are disabled (enable with -loadtest or NAT_LOADTEST=true)")
	}
	return devtest.SerialT(t)
}

func setupTestDeadline(t devtest.T, varName string) (devtest.T, context.Context, func()) {
	// Configure a context that will allow us to exit the test on time.
	var deadline time.Time
	if timeoutStr, exists := os.LookupEnv(varName); exists {
		timeout, err := time.ParseDuration(timeoutStr)
		t.Require().NoError(err)
		envVarDeadline := time.Now().Add(timeout)
		deadline = envVarDeadline
	}
	if deadline == (time.Time{}) {
		deadline = time.Now().Add(defaultSteadyTestTimeout)
	}
	ctx, cancel := context.WithDeadline(t.Ctx(), deadline)
	t = t.WithCtx(ctx)
	t.Cleanup(cancel)
	return t, ctx, cancel
}

func setupLoadTest(t devtest.T, ctx context.Context, wg *sync.WaitGroup, aimdOpts ...AIMDOption) (*AIMD, *L2, *L2) {
	sys := presets.NewSimpleInterop(t)
	blockTime := time.Duration(sys.L2ChainB.Escape().RollupConfig().BlockTime) * time.Second

	// Scheduler.
	targetMessagePassesPerBlock := uint64(100)
	if targetMsgPassesStr, exists := os.LookupEnv("NAT_INTEROP_LOADTEST_TARGET"); exists {
		var err error
		targetMessagePassesPerBlock, err = strconv.ParseUint(targetMsgPassesStr, 10, 0)
		t.Require().NoError(err)
	}
	aimd := NewAIMD(targetMessagePassesPerBlock, blockTime, aimdOpts...)
	wg.Add(1)
	go func() {
		defer wg.Done()
		aimd.Start(ctx)
	}()

	// Chains.
	budget := eth.OneEther
	if budgetStr, exists := os.LookupEnv("NAT_INTEROP_LOADTEST_BUDGET"); exists {
		amount, err := strconv.ParseUint(budgetStr, 10, 64)
		t.Require().NoError(err)
		budget = eth.Ether(amount)
	}
	l2ELA := sys.L2ChainA.PublicRPC()
	l2ELB := sys.L2ChainB.PublicRPC()
	funderA := dsl.NewFunder(sys.Wallet, sys.FaucetA, l2ELA)
	funderB := dsl.NewFunder(sys.Wallet, sys.FaucetB, l2ELB)
	const numEOAs = 300
	innerEOAsA := funderA.NewFundedEOAs(numEOAs, budget)
	innerEOAsB := funderB.NewFundedEOAs(numEOAs, budget)
	reliableELA := newReliableEL(l2ELA.Escape().EthClient(), blockTime, ResubmitterObserver("source"))
	reliableELB := newReliableEL(l2ELB.Escape().EthClient(), blockTime, ResubmitterObserver("destination"))
	eoasA := make([]*SyncEOA, 0, len(innerEOAsA))
	eoasB := make([]*SyncEOA, 0, len(innerEOAsA))
	for _, eoa := range innerEOAsA {
		p := txinclude.NewPersistent(
			txinclude.NewPkSigner(eoa.Key().Priv(), eoa.ChainID().ToBig()),
			reliableELA,
			txinclude.WithBudget(accounting.NewBudget(budget)),
		)
		eoasA = append(eoasA, &SyncEOA{
			Plan:     eoa.Plan(),
			Includer: p,
		})
	}
	for _, eoa := range innerEOAsB {
		p := txinclude.NewPersistent(
			txinclude.NewPkSigner(eoa.Key().Priv(), eoa.ChainID().ToBig()),
			reliableELB,
			txinclude.WithBudget(accounting.NewBudget(budget)),
		)
		eoasB = append(eoasB, &SyncEOA{
			Plan:     eoa.Plan(),
			Includer: p,
		})
	}
	l2A := &L2{
		Config:       sys.L2ChainA.Escape().ChainConfig(),
		RollupConfig: sys.L2ChainA.Escape().RollupConfig(),
		EOAs:         NewRoundRobin(eoasA),
		EL:           l2ELA,
	}
	l2B := &L2{
		Config:       sys.L2ChainB.Escape().ChainConfig(),
		RollupConfig: sys.L2ChainB.Escape().RollupConfig(),
		EOAs:         NewRoundRobin(eoasB),
		EL:           l2ELB,
	}
	l2A.DeployEventLogger(ctx, t)
	l2B.DeployEventLogger(ctx, t)

	// Metrics.
	metricsCollector := NewMetricsCollector(blockTime)
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := metricsCollector.Start(ctx)
		if isBenignCancellationError(err) {
			return
		}
		t.Require().NoError(err)
	}()
	t.Cleanup(func() {
		dir := filepath.Join("artifacts", t.Name()+"_"+time.Now().Format("20060102-150405"))
		t.Require().NoError(os.MkdirAll(dir, 0755))
		t.Require().NoError(metricsCollector.SaveGraphs(dir))
	})

	return aimd, l2A, l2B
}

func relayMessage(ctx context.Context, t devtest.T, source, dest *L2) error {
	rng := rand.New(rand.NewSource(1234))
	inFlightMessages.Inc()
	defer func() {
		inFlightMessages.Dec()
	}()
	startE2E := time.Now()

	startInit := startE2E
	initTx, err := source.Include(ctx, t, planCall(t, interop.RandomInitTrigger(rng, source.EventLogger, rng.Intn(2), rng.Intn(5))))
	if err != nil {
		return err
	}
	messageLatency.WithLabelValues("init").Observe(time.Since(startInit).Seconds())
	ref, err := source.EL.Escape().EthClient().BlockRefByHash(ctx, initTx.Receipt.BlockHash)
	if isBenignCancellationError(err) {
		return err
	}
	t.Require().NoError(err)
	out := new(txintent.InteropOutput)
	err = out.FromReceipt(t.Ctx(), initTx.Receipt, ref, source.EL.ChainID())
	if isBenignCancellationError(err) {
		return err
	}
	t.Require().NoError(err)
	t.Require().Len(out.Entries, 1)
	initMsg := out.Entries[0]

	startExec := time.Now()
	if _, err = dest.Include(ctx, t, planCall(t, &txintent.ExecTrigger{
		Executor: constants.CrossL2Inbox,
		Msg:      initMsg,
	}), func(tx *txplan.PlannedTx) {
		tx.AgainstBlock.Wrap(func(fn plan.Fn[eth.BlockInfo]) plan.Fn[eth.BlockInfo] {
			// The tx is invalid until we know it will be included at a higher timestamp than any
			// of the initiating messages, modulo reorgs. Wait to plan the relay tx against a
			// target block until the timestamp elapses. NOTE: this should be `>=`, but the mempool
			// filtering in op-geth currently uses the unsafe head's timestamp instead of the
			// pending timestamp. See https://github.com/ethereum-optimism/op-geth/issues/603.
			// TODO(16371): if every txintent.Call had a Plan() method, this Option could be
			// included with ExecTrigger.
			ctxErrFn := func(_ context.Context) (eth.BlockInfo, error) {
				return nil, ctx.Err()
			}
			for {
				ref, err := dest.EL.Escape().EthClient().BlockRefByLabel(ctx, eth.Unsafe)
				if isBenignCancellationError(err) {
					return ctxErrFn
				}
				t.Require().NoError(err)
				if ref.Time > initMsg.Identifier.Timestamp {
					break
				}
				select {
				case <-time.After(time.Duration(dest.RollupConfig.BlockTime) * time.Second):
				case <-ctx.Done():
					return ctxErrFn
				}
			}
			return fn
		})
	}); err != nil {
		return err
	}
	endExec := time.Now()
	messageLatency.WithLabelValues("exec").Observe(endExec.Sub(startExec).Seconds())

	messageLatency.WithLabelValues("e2e").Observe(endExec.Sub(startE2E).Seconds())
	return nil
}

// TODO(16371) every txintent.Call implementation should probably just be a txplan.Option.
func planCall(t devtest.T, call txintent.Call) txplan.Option {
	plan := make([]txplan.Option, 0)
	accessList, err := call.AccessList()
	t.Require().NoError(err)
	if accessList != nil {
		plan = append(plan, txplan.WithAccessList(accessList))
	}
	data, err := call.EncodeInput()
	t.Require().NoError(err)
	if data != nil {
		plan = append(plan, txplan.WithData(data))
	}
	to, err := call.To()
	t.Require().NoError(err)
	if to != nil {
		plan = append(plan, txplan.WithTo(to))
	}
	return txplan.Combine(plan...)
}

type reliableEL struct {
	*txinclude.Resubmitter
	*txinclude.Monitor
}

var _ txinclude.EL = (*reliableEL)(nil)

func newReliableEL(el txinclude.EL, blockTime time.Duration, observer txinclude.ResubmitterObserver) *reliableEL {
	return &reliableEL{
		Resubmitter: txinclude.NewResubmitter(el, blockTime, txinclude.WithObserver(observer)),
		Monitor:     txinclude.NewMonitor(el, blockTime),
	}
}
