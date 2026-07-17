package sysgo

import (
	"net"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestMetricsURLFromLogs(t *testing.T) {
	logger, logs := testlog.CaptureLogger(t, log.LevelInfo)
	logger.Info("started metrics server", "addr", &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 3456,
	})

	metricsURL, err := metricsURLFromLogs(logs)

	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:3456", metricsURL)
}
