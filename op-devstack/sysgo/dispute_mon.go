package sysgo

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	disputeMon "github.com/ethereum-optimism/optimism/op-dispute-mon"
	disputeMonConfig "github.com/ethereum-optimism/optimism/op-dispute-mon/config"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
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

type metricsAddrLogHandler struct {
	delegate slog.Handler
	addr     chan string
}

func newMetricsAddrLogHandler(delegate slog.Handler) *metricsAddrLogHandler {
	return &metricsAddrLogHandler{
		delegate: delegate,
		addr:     make(chan string, 1),
	}
}

func (h *metricsAddrLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.delegate.Enabled(ctx, level)
}

func (h *metricsAddrLogHandler) Handle(ctx context.Context, record slog.Record) error {
	if record.Message == "started metrics server" {
		record.Attrs(func(attr slog.Attr) bool {
			if attr.Key != "addr" {
				return true
			}
			addr, ok := attr.Value.Any().(net.Addr)
			if ok {
				select {
				case h.addr <- addr.String():
				default:
				}
			}
			return false
		})
	}
	return h.delegate.Handle(ctx, record)
}

func (h *metricsAddrLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &metricsAddrLogHandler{delegate: h.delegate.WithAttrs(attrs), addr: h.addr}
}

func (h *metricsAddrLogHandler) WithGroup(name string) slog.Handler {
	return &metricsAddrLogHandler{delegate: h.delegate.WithGroup(name), addr: h.addr}
}

func (h *metricsAddrLogHandler) metricsURL() (string, error) {
	select {
	case addr := <-h.addr:
		return "http://" + addr, nil
	default:
		return "", fmt.Errorf("started metrics server log with valid addr not found")
	}
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
	logHandler := newMetricsAddrLogHandler(logger.Handler())
	service, err := disputeMon.Main(t.Ctx(), log.NewLogger(logHandler), &cfg)
	require.NoError(err, "create dispute monitor service")
	metricsURL, err := logHandler.metricsURL()
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
