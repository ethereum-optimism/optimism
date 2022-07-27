package proxyd

import (
	"fmt"
	"os"
	"strings"
)

type ServerConfig struct {
	RPCHost           string `toml:"rpc_host"`
	RPCPort           int    `toml:"rpc_port"`
	WSHost            string `toml:"ws_host"`
	WSPort            int    `toml:"ws_port"`
	MaxBodySizeBytes  int64  `toml:"max_body_size_bytes"`
	MaxConcurrentRPCs int64  `toml:"max_concurrent_rpcs"`

	// TimeoutSeconds specifies the maximum time spent serving an HTTP request. Note that isn't used for websocket connections
	TimeoutSeconds int `toml:"timeout_seconds"`

	MaxUpstreamBatchSize int `toml:"max_upstream_batch_size"`

	EnableRequestLog     bool `toml:"enable_request_log"`
	MaxRequestBodyLogLen int  `toml:"max_request_body_log_len"`
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
