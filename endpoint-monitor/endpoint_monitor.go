package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/opio"
)

var (
	MetricWsSubscribeStatus = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ws_subscribe_status",
			Help: "eth_subscribe over websocket check status"},
		[]string{"status", "provider", "error"},
	)
)

func Main(version string) func(cliCtx *cli.Context) error {
	return func(cliCtx *cli.Context) error {
		cfg := NewConfig(cliCtx)
		if err := cfg.Check(); err != nil {
			return fmt.Errorf("invalid CLI flags: %w", err)
		}

		l := oplog.NewLogger(oplog.AppOut(cliCtx), cfg.LogConfig)
		oplog.SetGlobalLogHandler(l.GetHandler())

		endpointMonitor := NewEndpointMonitor(cfg, l)
		l.Info(fmt.Sprintf("starting endpoint monitor with checkInterval=%s checkDuration=%s", cfg.CheckInterval, cfg.CheckDuration))
		endpointMonitor.Start()

		registry := opmetrics.NewRegistry()
		registry.MustRegister(MetricWsSubscribeStatus)
		metricsCfg := cfg.MetricsConfig

		l.Info("starting metrics server", "addr", metricsCfg.ListenAddr, "port", metricsCfg.ListenPort)
		srv, err := opmetrics.StartServer(registry, metricsCfg.ListenAddr, metricsCfg.ListenPort)
		if err != nil {
			l.Error("error starting metrics server", err)
			return err
		}
		defer func() {
			if err := srv.Stop(cliCtx.Context); err != nil {
				l.Error("failed to stop metrics server", "err", err)
			}
		}()
		opio.BlockOnInterrupts()

		return nil
	}
}

type EndpointMonitor struct {
	cfg    Config
	logger log.Logger
}

func NewEndpointMonitor(cfg Config, l log.Logger) EndpointMonitor {
	return EndpointMonitor{cfg: cfg, logger: l}
}

func (e EndpointMonitor) Start() {
	for _, providerConfig := range e.cfg.GetProviderConfigs() {
		go e.runWebsocketCheckLoop(providerConfig, e.cfg.CheckInterval, e.cfg.CheckDuration)
	}
}

// getWrappingErrorMsg returns the most recently wrapped error message
// it's used in this case to get the error type reported by runSubscribeCallCheck
func getWrappingErrorMsg(err error) string {
	cause := errors.Cause(err)
	return strings.TrimSuffix(err.Error(), fmt.Sprintf(": %s", cause.Error()))
}

// runWebsocketCheckLoop runs subscribe call checks every checkInterval and reports status metrics to prometheus
func (e EndpointMonitor) runWebsocketCheckLoop(p ProviderConfig, checkInterval, checkDuration time.Duration) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		e.logger.Info("running websocket check", "provider", p.Name)
		err := e.runWebsocketCheck(p, checkDuration)
		if err != nil {
			errType := getWrappingErrorMsg(err)
			MetricWsSubscribeStatus.With(prometheus.Labels{"provider": p.Name, "status": "error", "error": errType}).Inc()
			e.logger.Error("finished websocket check", "provider", p.Name, "error", errType)
		} else {
			MetricWsSubscribeStatus.With(prometheus.Labels{"provider": p.Name, "status": "success", "error": ""}).Inc()
			e.logger.Info("finished websocket check", "provider", p.Name)
		}
		<-ticker.C
	}
}

// runWebsocketCheck creates a client and subscribes to blockchain head notifications and returns any errors encountered for reporting
func (e EndpointMonitor) runWebsocketCheck(p ProviderConfig, duration time.Duration) error {
	client, err := ethclient.Dial(p.Url)
	if err != nil {
		return errors.Wrap(err, "dial")
	}
	defer client.Close()

	headers := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		return errors.Wrap(err, "eth_subscribe_failed")
	}

	receivedData := false
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sub.Unsubscribe()
			if !receivedData {
				return errors.New("nodata")
			}
			return nil
		case err := <-sub.Err():
			return errors.Wrap(err, "read")
		case header := <-headers:
			e.logger.Debug(header.Hash().Hex(), "provider", p.Name)
			receivedData = true
		}
	}
}
