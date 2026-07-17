package sysgo

import (
	"net"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestMetricsAddrLogHandlerForwardsLogsAndCapturesFirstAddress(t *testing.T) {
	delegate, logs := testlog.CaptureLogger(t, log.LevelInfo)
	handler := newMetricsAddrLogHandler(delegate.Handler())
	logger := log.NewLogger(handler)

	logger.Info("unrelated log")
	logger.Info("started metrics server", "addr", &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 3456,
	})
	logger.Info("started metrics server", "addr", &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 7890,
	})

	metricsURL, err := handler.metricsURL()

	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:3456", metricsURL)
	require.NotNil(t, logs.FindLog(testlog.NewMessageFilter("unrelated log")))
	require.Len(t, logs.FindLogs(testlog.NewMessageFilter("started metrics server")), 2)
}
