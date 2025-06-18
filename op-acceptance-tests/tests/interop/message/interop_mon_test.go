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
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/stretchr/testify/assert"
)

// TestInteropMon is testing that the op-interop-mon metrics are correctly collected
func TestInteropMon(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)

	clients := map[eth.ChainID]*sources.EthClient{
		sys.L2ELA.Escape().ChainID(): sys.L2ELA.Escape().EthClient().(*sources.EthClient),
		sys.L2ELB.Escape().ChainID(): sys.L2ELB.Escape().EthClient().(*sources.EthClient),
	}

	require := t.Require()

	// Start op-interop-mon in the test context and attach to the devstack
	im, err := monitor.InteropMonitorServiceFromClients(t.Ctx(), "test", &monitor.CLIConfig{
		PollInterval: 50 * time.Millisecond,
		L2Rpcs:       []string{}, // unused here
		MetricsConfig: opmetrics.CLIConfig{
			Enabled: true,
		},
	}, clients, t.Logger())
	t.Require().NoError(err)
	require.NoError(im.Start(t.Ctx()))

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
	require.EventuallyWithT(func(t *assert.CollectT) {
		checker := opmetrics.NewMetricChecker(t, im.Metrics.(opmetrics.RegistryMetricer).Registry())
		checker.FindByName("op_interop_mon_default_message_status")
	}, 1*time.Second, 100*time.Millisecond)
	t.Log("op-interop-mon metrics check successful")

	require.NoError(im.Stop(t.Ctx()))
}
