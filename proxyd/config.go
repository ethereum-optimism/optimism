package proxyd

import (
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"
)

type ServerConfig struct {
	RPCHost           string `toml:"rpc_host"`
	RPCPort           int    `toml:"rpc_port"`
	WSHost            string `toml:"ws_host"`
	WSPort            int    `toml:"ws_port"`
	MaxBodySizeBytes  int64  `toml:"max_body_size_bytes"`
	MaxConcurrentRPCs int64  `toml:"max_concurrent_rpcs"`
	LogLevel          string `toml:"log_level"`

	// TimeoutSeconds specifies the maximum time spent serving an HTTP request. Note that isn't used for websocket connections
	TimeoutSeconds int `toml:"timeout_seconds"`

	MaxUpstreamBatchSize int `toml:"max_upstream_batch_size"`

	EnableRequestLog      bool `toml:"enable_request_log"`
	MaxRequestBodyLogLen  int  `toml:"max_request_body_log_len"`
	EnablePprof           bool `toml:"enable_pprof"`
	EnableXServedByHeader bool `toml:"enable_served_by_header"`
}

type CacheConfig struct {
	Enabled bool         `toml:"enabled"`
	TTL     TOMLDuration `toml:"ttl"`
}

type RedisConfig struct {
	URL       string `toml:"url"`
	Namespace string `toml:"namespace"`
}

type MetricsConfig struct {
	Enabled bool   `toml:"enabled"`
	Host    string `toml:"host"`
	Port    int    `toml:"port"`
}

type RateLimitConfig struct {
	UseRedis         bool                                `toml:"use_redis"`
	BaseRate         int                                 `toml:"base_rate"`
	BaseInterval     TOMLDuration                        `toml:"base_interval"`
	ExemptOrigins    []string                            `toml:"exempt_origins"`
	ExemptUserAgents []string                            `toml:"exempt_user_agents"`
	ErrorMessage     string                              `toml:"error_message"`
	MethodOverrides  map[string]*RateLimitMethodOverride `toml:"method_overrides"`
	IPHeaderOverride string                              `toml:"ip_header_override"`
}

type RateLimitMethodOverride struct {
	Limit    int          `toml:"limit"`
	Interval TOMLDuration `toml:"interval"`
	Global   bool         `toml:"global"`
}

type TOMLDuration time.Duration

func (t *TOMLDuration) UnmarshalText(b []byte) error {
	d, err := time.ParseDuration(string(b))
	if err != nil {
		return err
	}

	*t = TOMLDuration(d)
	return nil
}

type BackendOptions struct {
	ResponseTimeoutSeconds      int          `toml:"response_timeout_seconds"`
	MaxResponseSizeBytes        int64        `toml:"max_response_size_bytes"`
	MaxRetries                  int          `toml:"max_retries"`
	OutOfServiceSeconds         int          `toml:"out_of_service_seconds"`
	MaxDegradedLatencyThreshold TOMLDuration `toml:"max_degraded_latency_threshold"`
	MaxLatencyThreshold         TOMLDuration `toml:"max_latency_threshold"`
	MaxErrorRateThreshold       float64      `toml:"max_error_rate_threshold"`
}

type BackendConfig struct {
	Username         string            `toml:"username"`
	Password         string            `toml:"password"`
	RPCURL           string            `toml:"rpc_url"`
	WSURL            string            `toml:"ws_url"`
	WSPort           int               `toml:"ws_port"`
	MaxRPS           int               `toml:"max_rps"`
	MaxWSConns       int               `toml:"max_ws_conns"`
	CAFile           string            `toml:"ca_file"`
	ClientCertFile   string            `toml:"client_cert_file"`
	ClientKeyFile    string            `toml:"client_key_file"`
	StripTrailingXFF bool              `toml:"strip_trailing_xff"`
	Headers          map[string]string `toml:"headers"`

	Weight int `toml:"weight"`

	ConsensusSkipPeerCountCheck bool   `toml:"consensus_skip_peer_count"`
	ConsensusForcedCandidate    bool   `toml:"consensus_forced_candidate"`
	ConsensusReceiptsTarget     string `toml:"consensus_receipts_target"`
}

type BackendsConfig map[string]*BackendConfig

type BackendGroupConfig struct {
	Backends []string `toml:"backends"`

	WeightedRouting bool `toml:"weighted_routing"`

	ConsensusAware        bool   `toml:"consensus_aware"`
	ConsensusAsyncHandler string `toml:"consensus_handler"`

	ConsensusBanPeriod          TOMLDuration `toml:"consensus_ban_period"`
	ConsensusMaxUpdateThreshold TOMLDuration `toml:"consensus_max_update_threshold"`
	ConsensusMaxBlockLag        uint64       `toml:"consensus_max_block_lag"`
	ConsensusMaxBlockRange      uint64       `toml:"consensus_max_block_range"`
	ConsensusMinPeerCount       int          `toml:"consensus_min_peer_count"`

	ConsensusHA                  bool         `toml:"consensus_ha"`
	ConsensusHAHeartbeatInterval TOMLDuration `toml:"consensus_ha_heartbeat_interval"`
	ConsensusHALockPeriod        TOMLDuration `toml:"consensus_ha_lock_period"`
}

type BackendGroupsConfig map[string]*BackendGroupConfig

type MethodMappingsConfig map[string]string

type BatchConfig struct {
	MaxSize      int    `toml:"max_size"`
	ErrorMessage string `toml:"error_message"`
}

// SenderRateLimitConfig configures the sender-based rate limiter
// for eth_sendRawTransaction requests.
// To enable pre-eip155 transactions, add '0' to allowed_chain_ids.
type SenderRateLimitConfig struct {
	Enabled         bool
	Interval        TOMLDuration
	Limit           int
	AllowedChainIds []*big.Int `toml:"allowed_chain_ids"`
}

type Config struct {
	WSBackendGroup        string                `toml:"ws_backend_group"`
	Server                ServerConfig          `toml:"server"`
	Cache                 CacheConfig           `toml:"cache"`
	Redis                 RedisConfig           `toml:"redis"`
	Metrics               MetricsConfig         `toml:"metrics"`
	RateLimit             RateLimitConfig       `toml:"rate_limit"`
	BackendOptions        BackendOptions        `toml:"backend"`
	Backends              BackendsConfig        `toml:"backends"`
	BatchConfig           BatchConfig           `toml:"batch"`
	Authentication        map[string]string     `toml:"authentication"`
	BackendGroups         BackendGroupsConfig   `toml:"backend_groups"`
	RPCMethodMappings     map[string]string     `toml:"rpc_method_mappings"`
	WSMethodWhitelist     []string              `toml:"ws_method_whitelist"`
	WhitelistErrorMessage string                `toml:"whitelist_error_message"`
	SenderRateLimit       SenderRateLimitConfig `toml:"sender_rate_limit"`
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
