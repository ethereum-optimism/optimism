package proxyd

type ServerConfig struct {
	Host             string `toml:"host"`
	Port             int    `toml:"port"`
	MaxBodySizeBytes int64  `toml:"max_body_size_bytes"`
}

type MetricsConfig struct {
	Enabled bool   `toml:"enabled"`
	Host    string `toml:"host"`
	Port    int    `toml:"port"`
}

type BackendOptions struct {
	ResponseTimeoutSeconds               int   `toml:"response_timeout_seconds"`
	MaxResponseSizeBytes                 int64 `toml:"max_response_size_bytes"`
	MaxRetries                           int   `toml:"backend_retries"`
	UnhealthyBackendRetryIntervalSeconds int64 `toml:"unhealthy_backend_retry_interval_seconds"`
}

type BackendConfig struct {
	Username string `toml:"username"`
	Password string `toml:"password"`
	BaseURL  string `toml:"base_url"`
}

type BackendsConfig map[string]*BackendConfig

type BackendGroupConfig struct {
	Backends []string
}

type BackendGroupsConfig map[string]*BackendGroupConfig

type MethodMappingsConfig map[string]string

type Config struct {
	Server         *ServerConfig        `toml:"server"`
	Metrics        *MetricsConfig       `toml:"metrics"`
	BackendOptions *BackendOptions      `toml:"backend"`
	Backends       BackendsConfig       `toml:"backends"`
	BackendGroups  BackendGroupsConfig  `toml:"backend_groups"`
	MethodMappings MethodMappingsConfig `toml:"method_mappings"`
}
