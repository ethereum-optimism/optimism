package sysgo

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	disputeMon "github.com/ethereum-optimism/optimism/op-dispute-mon"
	disputeMonConfig "github.com/ethereum-optimism/optimism/op-dispute-mon/config"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type DisputeMonConfig struct {
	L1RPC              string
	GameFactoryAddress common.Address
	RollupRPCs         []string
	SupernodeRPCs      []string
}

type DisputeMonRuntime struct {
	metricsURL string
}

func (r *DisputeMonRuntime) MetricsURL() string {
	return r.metricsURL
}

func StartDisputeMon(t devtest.T, input DisputeMonConfig) *DisputeMonRuntime {
	require := t.Require()
	require.NotEmpty(input.L1RPC, "L1 RPC is required")
	require.NotEqual(common.Address{}, input.GameFactoryAddress, "dispute game factory address is required")
	require.NotEmpty(append(append([]string{}, input.RollupRPCs...), input.SupernodeRPCs...), "at least one rollup or supernode RPC is required")

	cfg := disputeMonConfig.NewCombinedConfig(
		input.GameFactoryAddress,
		input.L1RPC,
		input.RollupRPCs,
		input.SupernodeRPCs,
	)
	cfg.MonitorInterval = time.Second
	cfg.MetricsConfig = opmetrics.CLIConfig{
		Enabled:    true,
		ListenAddr: "127.0.0.1",
		ListenPort: 0,
	}

	logger := t.Logger().New("component", "op-dispute-mon")
	logHandler := testlog.WrapCaptureLogger(logger.Handler())
	capturedLogs := logHandler.(*testlog.CapturingHandler)
	service, err := disputeMon.Main(t.Ctx(), log.NewLogger(logHandler), &cfg)
	require.NoError(err, "create dispute monitor service")
	metricsURL, err := metricsURLFromLogs(capturedLogs)
	require.NoError(err, "find dispute monitor metrics URL")
	require.NoError(service.Start(t.Ctx()), "start dispute monitor service")

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		require.NoError(service.Stop(ctx), "stop dispute monitor service")
	})
	waitTCPReady(t, metricsURL, 10*time.Second)
	return &DisputeMonRuntime{metricsURL: metricsURL}
}

func metricsURLFromLogs(logs *testlog.CapturingHandler) (string, error) {
	record := logs.FindLog(testlog.NewMessageFilter("started metrics server"))
	if record == nil {
		return "", fmt.Errorf("started metrics server log not found")
	}
	addr, ok := record.AttrValue("addr").(net.Addr)
	if !ok {
		return "", fmt.Errorf("started metrics server log has invalid addr attribute")
	}
	return "http://" + addr.String(), nil
}
