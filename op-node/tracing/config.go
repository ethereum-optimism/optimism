package tracing

// TracingConfig defines the configuration for distributed tracing
type TracingConfig struct {
	Enabled     bool
	ServiceName string
	TracerName  string
}