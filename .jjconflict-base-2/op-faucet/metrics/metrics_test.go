package metrics

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	ftypes "github.com/ethereum-optimism/optimism/op-faucet/faucet/backend/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
)

func TestFaucetMetrics(t *testing.T) {
	m := NewMetrics("")

	require.NotEmpty(t, m.Document(), "sanity check there are generated metrics docs")

	version := "v3.4.5"
	m.RecordInfo(version)
	m.RecordUp()

	faucetA := ftypes.FaucetID("faucetA")
	faucetB := ftypes.FaucetID("faucetB")
	chainX := eth.ChainIDFromUInt64(123)
	chainY := eth.ChainIDFromUInt64(420)

	onDone := m.RecordFundAction(faucetA, chainX, eth.Ether(2))
	onDone(nil)
	onDone = m.RecordFundAction(faucetA, chainX, eth.Ether(3))
	onDone(nil)

	onDone = m.RecordFundAction(faucetB, chainY, eth.Ether(1000))
	onDone(errors.New("test err"))

	c := opmetrics.NewMetricChecker(t, m.Registry())

	prefix := Namespace + "_default_"

	labelsA := map[string]string{
		"faucet": faucetA.String(),
		"chain":  chainX.String(),
	}
	labelsB := map[string]string{
		"faucet": faucetB.String(),
		"chain":  chainY.String(),
	}

	//t.Log(c.Dump())

	record := c.FindByName(prefix + "funding_eth_total").FindByLabels(labelsA)
	require.Equal(t, eth.Ether(5).WeiFloat(), record.Counter.GetValue())

	record = c.FindByName(prefix + "funding_txs_total").FindByLabels(labelsA)
	require.Equal(t, 2.0, record.Counter.GetValue())

	record = c.FindByName(prefix + "funding_duration_seconds").FindByLabels(labelsA)
	require.NotZero(t, record.Histogram.GetSampleSum())

	record = c.FindByName(prefix + "funding_eth_total").FindByLabels(labelsB)
	require.Equal(t, eth.Ether(1000).WeiFloat(), record.Counter.GetValue())

	record = c.FindByName(prefix + "funding_txs_total").FindByLabels(labelsB)
	require.Equal(t, 1.0, record.Counter.GetValue())

	record = c.FindByName(prefix + "funding_duration_seconds").FindByLabels(labelsB)
	require.NotZero(t, record.Histogram.GetSampleSum())

	record = c.FindByName(prefix + "up").FindByLabels(nil)
	require.Equal(t, 1.0, record.Gauge.GetValue())

	record = c.FindByName(prefix + "info").FindByLabels(map[string]string{"version": version})
	require.Equal(t, 1.0, record.Gauge.GetValue())
}
