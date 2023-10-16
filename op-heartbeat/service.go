package op_heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/heartbeat"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
)

const (
	HTTPMaxHeaderSize = 10 * 1024
	HTTPMaxBodySize   = 1024 * 1024
)

func Main(version string) func(ctx *cli.Context) error {
	return func(cliCtx *cli.Context) error {
		cfg := NewConfig(cliCtx)
		if err := cfg.Check(); err != nil {
			return fmt.Errorf("invalid CLI flags: %w", err)
		}

		l := oplog.NewLogger(oplog.AppOut(cliCtx), cfg.Log)
		oplog.SetGlobalLogHandler(l.GetHandler())
		l.Info("starting heartbeat monitor", "version", version)

		srv, err := Start(cliCtx.Context, l, cfg, version)
		if err != nil {
			l.Crit("error starting application", "err", err)
		}

		doneCh := make(chan os.Signal, 1)
		signal.Notify(doneCh, []os.Signal{
			os.Interrupt,
			os.Kill,
			syscall.SIGTERM,
			syscall.SIGQUIT,
		}...)
		<-doneCh
		return srv.Stop(context.Background())
	}
}

type HeartbeatService struct {
	pprof, metrics, http *httputil.HTTPServer
}

func (hs *HeartbeatService) Stop(ctx context.Context) error {
	var result error
	if hs.pprof != nil {
		result = errors.Join(result, hs.pprof.Stop(ctx))
	}
	if hs.metrics != nil {
		result = errors.Join(result, hs.metrics.Stop(ctx))
	}
	if hs.http != nil {
		result = errors.Join(result, hs.http.Stop(ctx))
	}
	return result
}

func Start(ctx context.Context, l log.Logger, cfg Config, version string) (*HeartbeatService, error) {
	hs := &HeartbeatService{}

	registry := opmetrics.NewRegistry()
	metricsCfg := cfg.Metrics
	if metricsCfg.Enabled {
		l.Debug("starting metrics server", "addr", metricsCfg.ListenAddr, "port", metricsCfg.ListenPort)
		metricsSrv, err := opmetrics.StartServer(registry, metricsCfg.ListenAddr, metricsCfg.ListenPort)
		if err != nil {
			return nil, errors.Join(fmt.Errorf("failed to start metrics server: %w", err), hs.Stop(ctx))
		}
		hs.metrics = metricsSrv
		l.Info("started metrics server", "addr", metricsSrv.Addr())
	}

	pprofCfg := cfg.Pprof
	if pprofCfg.Enabled {
		l.Debug("starting pprof", "addr", pprofCfg.ListenAddr, "port", pprofCfg.ListenPort)
		pprofSrv, err := oppprof.StartServer(pprofCfg.ListenAddr, pprofCfg.ListenPort)
		if err != nil {
			return nil, errors.Join(fmt.Errorf("failed to start pprof server: %w", err), hs.Stop(ctx))
		}
		l.Info("started pprof server", "addr", pprofSrv.Addr())
		hs.pprof = pprofSrv
	}

	metrics := NewMetrics(registry)
	metrics.RecordVersion(version)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", HealthzHandler)
	mux.Handle("/", Handler(l, metrics))
	recorder := opmetrics.NewPromHTTPRecorder(registry, MetricsNamespace)
	mw := opmetrics.NewHTTPRecordingMiddleware(recorder, mux)

	srv, err := httputil.StartHTTPServer(
		net.JoinHostPort(cfg.HTTPAddr, strconv.Itoa(cfg.HTTPPort)),
		mw,
		httputil.WithTimeouts(httputil.HTTPTimeouts{
			ReadTimeout:       30 * time.Second,
			ReadHeaderTimeout: 30 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       time.Minute,
		}),
		httputil.WithMaxHeaderBytes(HTTPMaxHeaderSize))
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to start HTTP server: %w", err), hs.Stop(ctx))
	}
	hs.http = srv

	return hs, nil
}

func Handler(l log.Logger, metrics Metrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ipStr := r.Header.Get("X-Forwarded-For")
		// XFF can be a comma-separated list. Left-most is the original client.
		if i := strings.Index(ipStr, ","); i >= 0 {
			ipStr = ipStr[:i]
		}

		innerL := l.New(
			"ip", ipStr,
			"user_agent", r.Header.Get("User-Agent"),
			"remote_addr", r.RemoteAddr,
		)

		var payload heartbeat.Payload
		dec := json.NewDecoder(io.LimitReader(r.Body, int64(HTTPMaxBodySize)))
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

		metrics.RecordHeartbeat(payload, ipStr)

		w.WriteHeader(204)
	}
}

func HealthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(204)
}
