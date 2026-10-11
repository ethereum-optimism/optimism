package expiry

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-interop-mon/monitor"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/prometheus/client_golang/prometheus"
	gocl "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

const (
	expiryChecksMetric     = "op_interop_mon_default_expiry_checks_total"
	expiryViolationsMetric = "op_interop_mon_default_expiry_violations_total"
	ethInFlightMetric      = "op_interop_mon_default_eth_in_flight"
	ethInFlightTimeMetric  = "op_interop_mon_default_eth_in_flight_timestamp"
)

// startInteropMonitor attaches op-interop-mon to both chains of the system, reading up to their
// finalized blocks, and stops it when the test ends.
func startInteropMonitor(t devtest.T, sys *presets.SimpleInterop) *prometheus.Registry {
	clients := map[eth.ChainID]*sources.EthClient{
		sys.L2ChainA.ChainID(): sys.L2ELA.Escape().EthClient().(*sources.EthClient),
		sys.L2ChainB.ChainID(): sys.L2ELB.Escape().EthClient().(*sources.EthClient),
	}
	mon, err := monitor.InteropMonitorServiceFromClients(t.Ctx(), "test", &monitor.CLIConfig{
		PollInterval:    50 * time.Millisecond,
		MetricsConfig:   opmetrics.CLIConfig{Enabled: true},
		ExpiryCheckHead: string(eth.Finalized),
	}, clients, t.Logger())
	t.Require().NoError(err)
	t.Require().NoError(mon.Start(t.Ctx()))
	t.Cleanup(func() {
		if err := mon.Stop(t.Ctx()); err != nil {
			t.Logger().Warn("failed to stop op-interop-mon", "err", err)
		}
	})
	return mon.Metrics.(opmetrics.RegistryMetricer).Registry()
}

// verifyMonitorPassedChecks waits until the monitor has checked an expiry and a refund on chain A
// as valid, and has checked ETH conservation up to A's current head with no ETH left in flight. It
// then requires that the monitor found no violation of any check.
func verifyMonitorPassedChecks(t devtest.T, registry *prometheus.Registry, sys *presets.SimpleInterop) {
	chainA := sys.L2ChainA.ChainID().String()
	upTo := sys.L2ELA.BlockRefByLabel(eth.Unsafe).Time
	t.Require().EventuallyWithT(func(ct *assert.CollectT) {
		families, err := registry.Gather()
		if !assert.NoError(ct, err, "gather op-interop-mon metrics") {
			return
		}
		for _, check := range []string{"expiry", "refund"} {
			assert.Positive(ct, metricValue(families, expiryChecksMetric,
				map[string]string{"chain_id": chainA, "check": check, "result": "valid"}),
				"op-interop-mon must have checked a %s on chain %s as valid", check, chainA)
		}
		assert.GreaterOrEqual(ct, metricValue(families, ethInFlightTimeMetric, nil), float64(upTo),
			"op-interop-mon must have checked ETH conservation up to time %d", upTo)
		assert.Zero(ct, metricValue(families, ethInFlightMetric, nil), "no ETH must be left in flight")
	}, 5*time.Minute, time.Second, "op-interop-mon did not finish its checks")

	families, err := registry.Gather()
	t.Require().NoError(err)
	t.Require().Zero(metricValue(families, expiryViolationsMetric, nil), "op-interop-mon must find no violation")
	t.Require().Zero(metricValue(families, expiryChecksMetric, map[string]string{"result": "violation"}),
		"op-interop-mon must find no check failed")
}

// metricValue sums the counters or gauges of a metric family whose labels include `labels`.
func metricValue(families []*gocl.MetricFamily, name string, labels map[string]string) float64 {
	var sum float64
	for _, f := range families {
		if f.GetName() != name {
			continue
		}
	metrics:
		for _, m := range f.GetMetric() {
			for k, v := range labels {
				found := false
				for _, l := range m.GetLabel() {
					if l.GetName() == k && l.GetValue() == v {
						found = true
					}
				}
				if !found {
					continue metrics
				}
			}
			sum += m.GetCounter().GetValue() + m.GetGauge().GetValue()
		}
	}
	return sum
}
