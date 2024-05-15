package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	t.Run("should load an example config file", func(t *testing.T) {
		config, err := New("../../config.example.yaml")
		require.NoError(t, err)
		require.NotNil(t, config)

		require.Equal(t, "info", config.LogLevel)
		require.Equal(t, false, config.DryRun)

		require.Equal(t, false, config.Metrics.Debug)
		require.Equal(t, true, config.Metrics.Enabled)
		require.Equal(t, "0.0.0.0", config.Metrics.Host)
		require.Equal(t, "7300", config.Metrics.Port)

		require.Equal(t, true, config.Healthz.Enabled)
		require.Equal(t, "0.0.0.0", config.Healthz.Host)
		require.Equal(t, "8080", config.Healthz.Port)

		require.Equal(t, mustParseDuration("30s"), config.PollInterval)
		require.Equal(t, mustParseDuration("1h"), config.NodeStateExpiration)
		require.Equal(t, mustParseDuration("15s"), config.RPCTimeout)

		require.Equal(t, 2, len(config.Nodes))
		require.Equal(t, "http://op-conductor-0:9545", config.Nodes["op-conductor-0"].RPCAddress)
		require.Equal(t, "http://op-conductor-1:9545", config.Nodes["op-conductor-1"].RPCAddress)

		require.NoError(t, config.Validate())
	})
}

func mustParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return d
}
