package proxyd

type ServerConfig struct {
	Host             string `toml:"host"`
	Port             int    `toml:"port"`
	MaxBodySizeBytes int64  `toml:"max_body_size_bytes"`
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
	MaxRetries             int   `toml:"backend_retries"`
	OutOfServiceSeconds    int   `toml:"out_of_service_seconds"`
}

type BackendConfig struct {
	Username   string `toml:"username"`
	Password   string `toml:"password"`
	RPCURL     string `toml:"rpc_url"`
	WSURL      string `toml:"ws_url"`
	MaxRPS     int    `toml:"max_rps"`
	MaxWSConns int    `toml:"max_ws_conns"`
}

type BackendsConfig map[string]*BackendConfig

type BackendGroupConfig struct {
	Backends  []string `toml:"backends"`
	WSEnabled bool     `toml:"ws_enabled"`
}

type BackendGroupsConfig map[string]*BackendGroupConfig

type MethodMappingsConfig map[string]string

type Config struct {
	Server            *ServerConfig       `toml:"server"`
	Redis             *RedisConfig        `toml:"redis"`
	Metrics           *MetricsConfig      `toml:"metrics"`
	BackendOptions    *BackendOptions     `toml:"backend"`
	Backends          BackendsConfig      `toml:"backends"`
	Authentication    map[string]string   `toml:"authentication"`
	BackendGroups     BackendGroupsConfig `toml:"backend_groups"`
	RPCMethodMappings map[string]string   `toml:"rpc_method_mappings"`
	WSMethodWhitelist []string            `toml:"ws_method_whitelist"`
}
