package proxyd

type ServerConfig struct {
	RPCHost          string `toml:"rpc_host"`
	RPCPort          int    `toml:"rpc_port"`
	WSHost           string `toml:"ws_host"`
	WSPort           int    `toml:"ws_port"`
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
	MaxWSConns     int    `toml:"max_ws_conns"`
	CAFile         string `toml:"ca_file"`
	ClientCertFile string `toml:"client_cert_file"`
	ClientKeyFile  string `toml:"client_key_file"`
}

type BackendsConfig map[string]*BackendConfig

type BackendGroupConfig struct {
	Backends []string `toml:"backends"`
}

type BackendGroupsConfig map[string]*BackendGroupConfig

type MethodMappingsConfig map[string]string

type Config struct {
	WSBackendGroup    string              `toml:"ws_backend_group"`
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
