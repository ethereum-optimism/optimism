package proxyd

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/log"
)

type ServerConfig struct {
	RPCHost          string `toml:"rpc_host"`
	RPCPort          int    `toml:"rpc_port"`
	WSHost           string `toml:"ws_host"`
	WSPort           int    `toml:"ws_port"`
	MaxBodySizeBytes int64  `toml:"max_body_size_bytes"`
}

type CacheConfig struct {
	Enabled               bool   `toml:"enabled"`
	BlockSyncRPCURL       string `toml:"block_sync_rpc_url"`
	NumBlockConfirmations int    `toml:"num_block_confirmations"`
}

type RedisConfig struct {
	URL string `toml:"url"`
}

type MetricsConfig struct {
	Enabled bool   `toml:"enabled"`
	Host    string `toml:"host"`
	Port    int    `toml:"port"`
}

type BackendOptions struct {
	ResponseTimeoutSeconds int   `toml:"response_timeout_seconds"`
	MaxResponseSizeBytes   int64 `toml:"max_response_size_bytes"`
	MaxRetries             int   `toml:"max_retries"`
	OutOfServiceSeconds    int   `toml:"out_of_service_seconds"`
}

type BackendConfig struct {
	Username         string `toml:"username"`
	Password         string `toml:"password"`
	RPCURL           string `toml:"rpc_url"`
	WSURL            string `toml:"ws_url"`
	MaxRPS           int    `toml:"max_rps"`
	MaxWSConns       int    `toml:"max_ws_conns"`
	CAFile           string `toml:"ca_file"`
	ClientCertFile   string `toml:"client_cert_file"`
	ClientKeyFile    string `toml:"client_key_file"`
	StripTrailingXFF bool   `toml:"strip_trailing_xff"`
}

type BackendsConfig map[string]*BackendConfig

type BackendGroupConfig struct {
	Backends []string `toml:"backends"`
}

type BackendGroupsConfig map[string]*BackendGroupConfig

type MethodMappingsConfig map[string]string

type EthConfig struct {
	L2ChainID          *big.Int `toml:"l2_chain_id"`
	BedrockCutoffBlock *big.Int `toml:"bedrock_cutoff_block"`
}

type Config struct {
	WSBackendGroup    string              `toml:"ws_backend_group"`
	Server            ServerConfig        `toml:"server"`
	Cache             CacheConfig         `toml:"cache"`
	Redis             RedisConfig         `toml:"redis"`
	Metrics           MetricsConfig       `toml:"metrics"`
	BackendOptions    BackendOptions      `toml:"backend"`
	Backends          BackendsConfig      `toml:"backends"`
	Authentication    map[string]string   `toml:"authentication"`
	BackendGroups     BackendGroupsConfig `toml:"backend_groups"`
	RPCMethodMappings map[string]string   `toml:"rpc_method_mappings"`
	WSMethodWhitelist []string            `toml:"ws_method_whitelist"`
	Eth               EthConfig           `toml:"eth_config"`
	LogFormat         string              `toml:"log_format"`
}

func (c *Config) ResolveAuth() (map[string]string, error) {
	var resolvedAuth map[string]string
	if c.Authentication != nil {
		resolvedAuth = make(map[string]string)
		for secret, alias := range c.Authentication {
			resolvedSecret, err := ReadFromEnvOrConfig(secret)
			if err != nil {
				return nil, err
			}
			resolvedAuth[resolvedSecret] = alias
		}
	}
	return resolvedAuth, nil
}

func (c *Config) BuildBackends(lim RateLimiter) ([]string, map[string]*Backend, error) {
	backendNames := make([]string, 0)
	backendsByName := make(map[string]*Backend)

	for name, cfg := range c.Backends {
		opts := make([]BackendOpt, 0)

		rpcURL, err := ReadFromEnvOrConfig(cfg.RPCURL)
		if err != nil {
			return nil, nil, err
		}
		wsURL, err := ReadFromEnvOrConfig(cfg.WSURL)
		if err != nil {
			return nil, nil, err
		}
		if rpcURL == "" {
			return nil, nil, fmt.Errorf("must define an RPC URL for backend %s", name)
		}
		if wsURL == "" {
			return nil, nil, fmt.Errorf("must define a WS URL for backend %s", name)
		}

		if c.BackendOptions.ResponseTimeoutSeconds != 0 {
			timeout := secondsToDuration(c.BackendOptions.ResponseTimeoutSeconds)
			opts = append(opts, WithTimeout(timeout))
		}
		if c.BackendOptions.MaxRetries != 0 {
			opts = append(opts, WithMaxRetries(c.BackendOptions.MaxRetries))
		}
		if c.BackendOptions.MaxResponseSizeBytes != 0 {
			opts = append(opts, WithMaxResponseSize(c.BackendOptions.MaxResponseSizeBytes))
		}
		if c.BackendOptions.OutOfServiceSeconds != 0 {
			opts = append(opts, WithOutOfServiceDuration(secondsToDuration(c.BackendOptions.OutOfServiceSeconds)))
		}
		if cfg.MaxRPS != 0 {
			opts = append(opts, WithMaxRPS(cfg.MaxRPS))
		}
		if cfg.MaxWSConns != 0 {
			opts = append(opts, WithMaxWSConns(cfg.MaxWSConns))
		}
		if cfg.Password != "" {
			passwordVal, err := ReadFromEnvOrConfig(cfg.Password)
			if err != nil {
				return nil, nil, err
			}
			opts = append(opts, WithBasicAuth(cfg.Username, passwordVal))
		}
		tlsConfig, err := configureBackendTLS(cfg)
		if err != nil {
			return nil, nil, err
		}
		if tlsConfig != nil {
			log.Info("using custom TLS config for backend", "name", name)
			opts = append(opts, WithTLSConfig(tlsConfig))
		}
		if cfg.StripTrailingXFF {
			opts = append(opts, WithStrippedTrailingXFF())
		}
		opts = append(opts, WithProxydIP(os.Getenv("PROXYD_IP")))
		back := NewBackend(name, rpcURL, wsURL, lim, opts...)
		backendNames = append(backendNames, name)
		backendsByName[name] = back
		log.Info("configured backend", "name", name, "rpc_url", rpcURL, "ws_url", wsURL)
	}

	return backendNames, backendsByName, nil
}

func (c *Config) ValidateDaisyChainBackends() error {
	valid := false
	for name := range c.Backends {
		switch name {
		case "epoch1", "epoch2", "epoch3", "epoch4", "epoch5", "epoch6":
			valid = true
		default:
			continue
		}
	}
	if valid {
		return nil
	}
	return errors.New("invalid backend name configuration")
}

func ReadFromEnvOrConfig(value string) (string, error) {
	if strings.HasPrefix(value, "$") {
		envValue := os.Getenv(strings.TrimPrefix(value, "$"))
		if envValue == "" {
			return "", fmt.Errorf("config env var %s not found", value)
		}
		return envValue, nil
	}

	if strings.HasPrefix(value, "\\") {
		return strings.TrimPrefix(value, "\\"), nil
	}

	return value, nil
}
