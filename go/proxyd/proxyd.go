package proxyd

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Start(config *Config) error {
	if len(config.Backends) == 0 {
		return errors.New("must define at least one backend")
	}
	if len(config.AllowedRPCMethods) == 0 {
		return errors.New("must define at least one allowed RPC method")
	}

	allowedRPCs := NewStringSetFromStrings(config.AllowedRPCMethods)
	allowedWSRPCs := allowedRPCs.Extend(config.AllowedWSMethods)

	redis, err := NewRedis(config.Redis.URL)
	if err != nil {
		return err
	}

	backends := make([]*Backend, 0)
	backendNames := make([]string, 0)
	for name, cfg := range config.Backends {
		opts := make([]BackendOpt, 0)

		if cfg.RPCURL == "" {
			return fmt.Errorf("must define an RPC URL for backend %s", name)
		}
		if cfg.WSURL == "" {
			return fmt.Errorf("must define a WS URL for backend %s", name)
		}

		if config.BackendOptions.ResponseTimeoutSeconds != 0 {
			timeout := secondsToDuration(config.BackendOptions.ResponseTimeoutSeconds)
			opts = append(opts, WithTimeout(timeout))
		}
		if config.BackendOptions.MaxRetries != 0 {
			opts = append(opts, WithMaxRetries(config.BackendOptions.MaxRetries))
		}
		if config.BackendOptions.MaxResponseSizeBytes != 0 {
			opts = append(opts, WithMaxResponseSize(config.BackendOptions.MaxResponseSizeBytes))
		}
		if config.BackendOptions.OutOfServiceSeconds != 0 {
			opts = append(opts, WithOutOfServiceDuration(secondsToDuration(config.BackendOptions.OutOfServiceSeconds)))
		}
		if cfg.MaxRPS != 0 {
			opts = append(opts, WithMaxRPS(cfg.MaxRPS))
		}
		if cfg.MaxWSConns != 0 {
			opts = append(opts, WithMaxWSConns(cfg.MaxWSConns))
		}
		if cfg.Password != "" {
			opts = append(opts, WithBasicAuth(cfg.Username, cfg.Password))
		}
		back := NewBackend(name, cfg.RPCURL, cfg.WSURL, allowedRPCs, allowedWSRPCs, redis, opts...)
		backends = append(backends, back)
		backendNames = append(backendNames, name)
		log.Info("configured backend", "name", name, "rpc_url", cfg.RPCURL, "ws_url", cfg.WSURL)
	}

	backendGroup := &BackendGroup{
		Name:     "main",
		Backends: backends,
	}
	srv := NewServer(backendGroup, config.Server.MaxBodySizeBytes)

	if config.Metrics.Enabled {
		addr := fmt.Sprintf("%s:%d", config.Metrics.Host, config.Metrics.Port)
		log.Info("starting metrics server", "addr", addr)
		go http.ListenAndServe(addr, promhttp.Handler())
	}

	go func() {
		if err := srv.ListenAndServe(config.Server.Host, config.Server.Port); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				log.Info("server shut down")
				return
			}
			log.Crit("error starting server", "err", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	recvSig := <-sig
	log.Info("caught signal, shutting down", "signal", recvSig)
	srv.Shutdown()
	if err := redis.FlushBackendWSConns(backendNames); err != nil {
		log.Error("error flushing backend ws conns", "err", err)
	}
	return nil
}

func secondsToDuration(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}
