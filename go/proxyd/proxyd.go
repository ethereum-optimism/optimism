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
	backendsByName := make(map[string]*Backend)
	groupsByName := make(map[string]*BackendGroup)

	if len(config.Backends) == 0 {
		return errors.New("must define at least one backend")
	}
	if len(config.BackendGroups) == 0 {
		return errors.New("must define at least one backend group")
	}
	if len(config.MethodMappings) == 0 {
		return errors.New("must define at least one method mapping")
	}

	for name, cfg := range config.Backends {
		opts := make([]BackendOpt, 0)

		if cfg.BaseURL == "" {
			return fmt.Errorf("must define a base URL for backend %s", name)
		}

		if config.BackendOptions.ResponseTimeoutSeconds != 0 {
			timeout := time.Duration(config.BackendOptions.ResponseTimeoutSeconds) * time.Second
			opts = append(opts, WithTimeout(timeout))
		}
		if config.BackendOptions.MaxRetries != 0 {
			opts = append(opts, WithMaxRetries(config.BackendOptions.MaxRetries))
		}
		if config.BackendOptions.MaxResponseSizeBytes != 0 {
			opts = append(opts, WithMaxResponseSize(config.BackendOptions.MaxResponseSizeBytes))
		}
		if config.BackendOptions.UnhealthyBackendRetryIntervalSeconds != 0 {
			opts = append(opts, WithUnhealthyRetryInterval(config.BackendOptions.UnhealthyBackendRetryIntervalSeconds))
		}
		if cfg.Password != "" {
			opts = append(opts, WithBasicAuth(cfg.Username, cfg.Password))
		}
		backendsByName[name] = NewBackend(name, cfg.BaseURL, opts...)
		log.Info("configured backend", "name", name, "base_url", cfg.BaseURL)
	}

	for groupName, cfg := range config.BackendGroups {
		backs := make([]*Backend, 0)
		for _, backName := range cfg.Backends {
			if backendsByName[backName] == nil {
				return fmt.Errorf("undefined backend %s", backName)
			}
			backs = append(backs, backendsByName[backName])
			log.Info("configured backend group", "name", groupName)
		}

		groupsByName[groupName] = &BackendGroup{
			Name:     groupName,
			backends: backs,
		}
	}

	mappings := make(map[string]*BackendGroup)
	for method, groupName := range config.MethodMappings {
		if groupsByName[groupName] == nil {
			return fmt.Errorf("undefined backend group %s", groupName)
		}
		mappings[method] = groupsByName[groupName]
	}
	methodMappings := NewMethodMapping(mappings)

	srv := NewServer(methodMappings, config.Server.MaxBodySizeBytes)

	if config.Metrics.Enabled {
		addr := fmt.Sprintf("%s:%d", config.Metrics.Host, config.Metrics.Port)
		log.Info("starting metrics server", "addr", addr)
		go http.ListenAndServe(addr, promhttp.Handler())
	}

	go func() {
		if err := srv.ListenAndServe(config.Server.Host, config.Server.Port); err != nil {
			log.Crit("error starting server", "err", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	recvSig := <-sig
	log.Info("caught signal, shutting down", "signal", recvSig)
	return nil
}
