package msg

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-interop-mon/monitor"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"

	stypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// getExecutingMessagesMetricValue extracts the executing messages metric value from the InteropMonitor
func getExecutingMessagesMetricValue(im *monitor.InteropMonitorService) (float64, error) {
	// Get the metrics registry from the InteropMonitor
	metricsRegistry := im.Metrics.(opmetrics.RegistryMetricer).Registry()

	// Gather all metrics
	metricFamilies, err := metricsRegistry.Gather()
	if err != nil {
		return 0, err
	}

	// Look for the executing_messages metric
	for _, metricFamily := range metricFamilies {
		if metricFamily.GetName() == "op_interop_mon_default_executing_messages" {
			metrics := metricFamily.GetMetric()
			// Sum up all the executing messages across all labels
			var total float64
			for _, metric := range metrics {
				if metric.Gauge != nil && metric.Gauge.Value != nil {
					total += *metric.Gauge.Value
				}
			}
			return total, nil
		}
	}
	return 0, nil
}

// TestInteropHappyTx is testing that a valid init message, followed by a valid exec message are correctly
// included in two L2 chains and that the cross-safe ref for both of them progresses as expected beyond
// the block number where the messages were included
func TestInteropHappyTx(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)

	l2Rpcs := []string{
		sys.L2ELA.Escape().RPCURL(),
		sys.L2ELB.Escape().RPCURL(),
	}
	t.Logf("l2 rpcs: %v", l2Rpcs)
	im, err := monitor.InteropMonitorServiceFromCLIConfig(t.Ctx(), "0.0.1", &monitor.CLIConfig{
		PollInterval: 1 * time.Second,
		L2Rpcs:       l2Rpcs,
		MetricsConfig: opmetrics.CLIConfig{
			Enabled: true,
		},
	}, t.Logger())
	t.Require().NoError(err)

	im.Start(t.Ctx())
	defer im.Stop(t.Ctx())

	// two EOAs for triggering the init and exec interop txs
	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)

	eventLoggerAddress := alice.DeployEventLogger()

	// wait for chain B to catch up to chain A if necessary
	sys.L2ChainB.CatchUpTo(sys.L2ChainA)

	// send initiating message on chain A
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	initTx, initReceipt := alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLoggerAddress, rng.Intn(3), rng.Intn(10)))

	// at least one block between the init tx on chain A and the exec tx on chain B
	sys.L2ChainB.WaitForBlock()

	// send executing message on chain B
	_, execReceipt := bob.SendExecMessage(initTx, 0)

	// Wait for metrics to be collected and check that executing messages metric is above zero
	time.Sleep(2 * time.Second) // Allow time for metric collection
	executingMessagesValue, err := getExecutingMessagesMetricValue(im)
	t.Require().NoError(err, "should be able to get executing messages metric")
	t.Require().Greater(executingMessagesValue, 0.0, "executing messages metric should be greater than zero")

	t.Logf("executing messages worked! metric value: %v", executingMessagesValue)

	// confirm that the cross-safe safety passed init and exec receipts and that blocks were not reorged
	dsl.CheckAll(t,
		sys.L2CLA.ReachedRefFn(stypes.CrossSafe, eth.BlockID{
			Number: initReceipt.BlockNumber.Uint64(),
			Hash:   initReceipt.BlockHash,
		}, 30),
		sys.L2CLB.ReachedRefFn(stypes.CrossSafe, eth.BlockID{
			Number: execReceipt.BlockNumber.Uint64(),
			Hash:   execReceipt.BlockHash,
		}, 30),
	)

	sys.L2ChainA.PrintChain()
	sys.L2ChainB.PrintChain()
}
