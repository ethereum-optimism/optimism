// Package loadtest contains interop load tests.
//
// Set NAT_INTEROP_LOADTEST_TARGET to the initial amount of messages that should be passed per L2 slot in each test (default: 100).
//
// Each test increases the message throughput until some threshold is reached (e.g., the gas target).
// The throughput is decreased if the threshold is exceeded or if errors are encountered (e.g., transaction inclusion failures).
package loadtest

import (
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"golang.org/x/net/context"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithSimpleInterop())
}

// TestSteady attempts to approach but not exceed the gas target in every block by spamming interop messages,
// simulating benign but heavy activity.
// The test will exit successfully after the global go test deadline or the timeout specified by the
// NAT_STEADY_TIMEOUT environment variable elapses, whichever comes first.
// Also see: https://github.com/golang/go/issues/48157.
func TestSteady(gt *testing.T) {
	t := setupT(gt)
	var wg sync.WaitGroup
	defer wg.Wait()

	// Configure a context that will allow us to exit the test on time.
	deadline := time.Unix(math.MaxInt64, 0)
	testCtxDeadline, testCtxDeadlineExsts := t.Ctx().Deadline()
	if testCtxDeadlineExsts {
		deadline = testCtxDeadline.Add(-10 * time.Second) // Give some time for cleanup.
	}
	ctx, cancel := context.WithDeadline(t.Ctx(), deadline)
	t.Cleanup(cancel) // We only care about the deadline.
	if timeoutStr, exists := os.LookupEnv("NAT_STEADY_TIMEOUT"); exists {
		timeout, err := time.ParseDuration(timeoutStr)
		t.Require().NoError(err)
		ctx, cancel = context.WithTimeout(ctx, timeout)
		t.Cleanup(cancel) // We only care about the timeout.
	}

	aimd, source, dest := setupLoadTest(t, ctx, &wg)
	elasticityMultiplier := dest.Config.ElasticityMultiplier()
	for range aimd.Ready() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !relayMessage(t, source, dest) {
				aimd.Adjust(false)
				return
			}
			unsafe := dest.Unsafe()
			gasTarget := unsafe.GasLimit() / elasticityMultiplier
			// Apply backpressure when we meet or exceed the gas target.
			aimd.Adjust(unsafe.GasUsed() < gasTarget)
		}()
	}
}

// TestBurst spams interop messages and exits successfully when the base fee is raised to one gwei.
// This simulates adversarial behavior.
func TestBurst(gt *testing.T) {
	t := setupT(gt)
	var wg sync.WaitGroup
	defer wg.Wait()
	ctx, cancel := context.WithCancel(t.Ctx())
	defer cancel()
	aimd, source, dest := setupLoadTest(t, ctx, &wg)
	targetBaseFee := eth.OneGWei.ToBig()
	for {
		select {
		case <-ctx.Done():
			return
		case <-aimd.Ready():
			wg.Add(1)
			go func() {
				defer wg.Done()
				success := relayMessage(t, source, dest)
				if dest.Unsafe().BaseFee().Cmp(targetBaseFee) >= 0 {
					cancel() // End the test when we've met or exceeded the target base fee.
				}
				aimd.Adjust(success)
			}()
		}
	}
}

func setupT(t *testing.T) devtest.T {
	if testing.Short() {
		t.Skip("skipping load test in short mode")
	}
	return devtest.SerialT(t)
}

func setupLoadTest(t devtest.T, ctx context.Context, wg *sync.WaitGroup) (*AIMD, *L2, *L2) {
	sys := presets.NewSimpleInterop(t)
	blockTime := time.Duration(sys.L2ChainB.Escape().RollupConfig().BlockTime) * time.Second

	// Scheduler.
	targetMessagePassesPerBlock := uint64(100)
	if targetMsgPassesStr, exists := os.LookupEnv("NAT_INTEROP_LOADTEST_TARGET"); exists {
		var err error
		targetMessagePassesPerBlock, err = strconv.ParseUint(targetMsgPassesStr, 10, 0)
		t.Require().NoError(err)
	}
	aimd := NewAIMD(targetMessagePassesPerBlock, blockTime, WithAdjustWindow(targetMessagePassesPerBlock/2))
	wg.Add(1)
	go func() {
		defer wg.Done()
		aimd.Start(ctx)
	}()

	// Chains.
	l2ELA := sys.L2ChainA.PublicRPC()
	l2ELB := sys.L2ChainB.PublicRPC()
	funderA := dsl.NewFunder(sys.Wallet, sys.FaucetA, l2ELA)
	const numEOAs = 300
	source := &L2{
		Config:       sys.L2ChainA.Escape().ChainConfig(),
		RollupConfig: sys.L2ChainA.Escape().RollupConfig(),
		eoas:         NewEOAPool(funderA, numEOAs, eth.MillionEther),
		el:           l2ELA,
		eventLogger:  funderA.NewFundedEOA(eth.OneEther).DeployEventLogger(),
	}
	dest := &L2{
		Config:       sys.L2ChainB.Escape().ChainConfig(),
		RollupConfig: sys.L2ChainB.Escape().RollupConfig(),
		eoas:         NewEOAPool(dsl.NewFunder(sys.Wallet, sys.FaucetB, l2ELB), numEOAs, eth.MillionEther),
		el:           l2ELB,
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		dest.PollUnsafe(ctx, t, blockTime)
	}()

	// Metrics.
	metricsCollector := NewMetricsCollector(blockTime)
	wg.Add(1)
	go func() {
		defer wg.Done()
		t.Require().NoError(metricsCollector.Start(ctx))
	}()
	t.Cleanup(func() {
		dir := filepath.Join("artifacts", t.Name())
		t.Require().NoError(os.MkdirAll(dir, 0755))
		t.Require().NoError(metricsCollector.SaveGraph(dir))
	})

	return aimd, source, dest
}

func relayMessage(t devtest.T, source, dest *L2) bool {
	rng := rand.New(rand.NewSource(1234))
	inFlightMessages.Inc()
	start := time.Now()
	initMsg := source.SendInitiatingMsg(t, rng)
	if initMsg == nil {
		messageStatusCount.WithLabelValues("init_failed").Inc()
		inFlightMessages.Dec()
		return false
	}
	success := dest.SendExecutingMsg(t, *initMsg)
	if success {
		messageLatency.WithLabelValues("e2e").Observe(time.Since(start).Seconds())
		messageStatusCount.WithLabelValues("success").Inc()
	} else {
		messageStatusCount.WithLabelValues("exec_failed").Inc()
	}
	inFlightMessages.Dec()
	return success
}
