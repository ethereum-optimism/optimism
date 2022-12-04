package op_heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/ethereum-optimism/optimism/op-node/heartbeat"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"
)

func Main(version string) func(ctx *cli.Context) error {
	return func(cliCtx *cli.Context) error {
		cfg := NewConfig(cliCtx)
		if err := cfg.Check(); err != nil {
			return fmt.Errorf("invalid CLI flags: %w", err)
		}

		l := oplog.NewLogger(cfg.Log)
		l.Info("starting heartbeat monitor", "version", version)

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			if err := Start(ctx, l, cfg, version); err != nil {
				l.Crit("error starting application", "err", err)
			}
		}()

		doneCh := make(chan os.Signal, 1)
		signal.Notify(doneCh, []os.Signal{
			os.Interrupt,
			os.Kill,
			syscall.SIGTERM,
			syscall.SIGQUIT,
		}...)
		<-doneCh
		cancel()
		return nil
	}
}

func Start(ctx context.Context, l log.Logger, cfg Config, version string) error {
	registry := opmetrics.NewRegistry()

	metricsCfg := cfg.Metrics
	if metricsCfg.Enabled {
		l.Info("starting metrics server", "addr", metricsCfg.ListenAddr, "port", metricsCfg.ListenPort)
		go func() {
			if err := opmetrics.ListenAndServe(ctx, registry, metricsCfg.ListenAddr, metricsCfg.ListenPort); err != nil {
				l.Error("error starting metrics server", err)
			}
		}()
	}

	pprofCfg := cfg.Pprof
	if pprofCfg.Enabled {
		l.Info("starting pprof server", "addr", pprofCfg.ListenAddr, "port", pprofCfg.ListenPort)
		go func() {
			if err := oppprof.ListenAndServe(ctx, pprofCfg.ListenAddr, pprofCfg.ListenPort); err != nil {
				l.Error("error starting pprof server", err)
			}
		}()
	}

	metrics := NewMetrics(registry)
	metrics.RecordVersion(version)
	handler := Handler(l, cfg.HTTPMaxBodySize, metrics)
	recorder := opmetrics.NewPromHTTPRecorder(registry, MetricsNamespace)
	mw := opmetrics.NewHTTPRecordingMiddleware(recorder, handler)

	server := &http.Server{
		Addr:           net.JoinHostPort(cfg.HTTPAddr, strconv.Itoa(cfg.HTTPPort)),
		MaxHeaderBytes: cfg.HTTPMaxBodySize,
		Handler:        mw,
	}

	return httputil.ListenAndServeContext(ctx, server)
}

func Handler(l log.Logger, maxBodySize int, metrics Metrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		innerL := l.New(
			"xff", r.Header.Get("X-Forwarded-For"),
			"user_agent", r.Header.Get("User-Agent"),
			"remote_addr", r.RemoteAddr,
		)

		defer r.Body.Close()
		var payload heartbeat.Payload
		dec := json.NewDecoder(io.LimitReader(r.Body, int64(maxBodySize)))
		if err := dec.Decode(&payload); err != nil {
			innerL.Info("error decoding request payload", "err", err)
			w.WriteHeader(400)
			return
		}

		innerL.Info(
			"got heartbeat",
			"version", payload.Version,
			"meta", payload.Meta,
			"moniker", payload.Moniker,
			"peer_id", payload.PeerID,
			"chain_id", payload.ChainID,
		)

		metrics.RecordHeartbeat(payload)

		w.WriteHeader(204)
	}
}
