package msg

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-interop-mon/monitor"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInteropMon is testing that the op-interop-mon metrics are correctly collected
func TestInteropMon(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)

	l2Rpcs := []string{
		sys.L2ELA.Escape().RPCURL(),
		sys.L2ELB.Escape().RPCURL(),
	}

	// Start op-interop-mon in the test context and attach to the devstack
	t.Logf("Starting op-interop-mon with l2 rpcs: %v", l2Rpcs)
	im, err := monitor.InteropMonitorServiceFromCLIConfig(t.Ctx(), "test", &monitor.CLIConfig{
		PollInterval: 50 * time.Millisecond,
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

	// send initiating message on chain A
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	initTx, _ := alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLoggerAddress, rng.Intn(3), rng.Intn(10)))

	// send executing message on chain B
	_, _ = bob.SendExecMessage(initTx, 0)

	// Ensure the metrics are generated
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		checker := opmetrics.NewMetricChecker(t, im.Metrics.(opmetrics.RegistryMetricer).Registry())
		checker.FindByName("op_interop_mon_default_executing_messages")
	}, 1*time.Second, 100*time.Millisecond)
	t.Log("op-interop-mon metrics check successful")
}
