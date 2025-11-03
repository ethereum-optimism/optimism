// Package config provides configuration validation utilities.
package config

import (
	"fmt"
	"net/url"
	"time"

	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Message string
	Value   interface{}
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s (value: %v)", e.Field, e.Message, e.Value)
}

// ValidateConfig validates the op-node configuration with descriptive error messages.
func ValidateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}
	
	var errors []error
	
	// Validate L1 endpoint
	if l1Config, ok := cfg.L1.(*L1EndpointConfig); ok {
		if err := validateL1Endpoint(l1Config); err != nil {
			errors = append(errors, err)
		}
	}
	
	// Validate L2 endpoint
	if l2Config, ok := cfg.L2.(*L2EndpointConfig); ok {
		if err := validateL2Endpoint(l2Config); err != nil {
			errors = append(errors, err)
		}
	}
	
	// Validate RPC configuration
	if err := validateRPCConfig(cfg.RPC); err != nil {
		errors = append(errors, err)
	}
	
	// Validate metrics configuration
	if err := validateMetricsConfig(cfg.Metrics); err != nil {
		errors = append(errors, err)
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed with %d error(s): %v", len(errors), errors)
	}
	
	return nil
}

func validateL1Endpoint(cfg *L1EndpointConfig) error {
	if cfg.L1NodeAddr == "" {
		return &ValidationError{
			Field:   "l1.node_addr",
			Message: "L1 node address cannot be empty",
			Value:   cfg.L1NodeAddr,
		}
	}
	
	if _, err := url.Parse(cfg.L1NodeAddr); err != nil {
		return &ValidationError{
			Field:   "l1.node_addr",
			Message: fmt.Sprintf("L1 node address must be a valid URL: %v", err),
			Value:   cfg.L1NodeAddr,
		}
	}
	
	if cfg.RateLimit < 0 {
		return &ValidationError{
			Field:   "l1.rate_limit",
			Message: "Rate limit cannot be negative",
			Value:   cfg.RateLimit,
		}
	}
	
	if cfg.BatchSize < 1 {
		return &ValidationError{
			Field:   "l1.batch_size",
			Message: "Batch size must be at least 1",
			Value:   cfg.BatchSize,
		}
	}
	
	if cfg.MaxConcurrency < 1 {
		return &ValidationError{
			Field:   "l1.max_concurrency",
			Message: "Max concurrency must be at least 1",
			Value:   cfg.MaxConcurrency,
		}
	}
	
	return nil
}

func validateL2Endpoint(cfg *L2EndpointConfig) error {
	if cfg.L2EngineAddr == "" {
		return &ValidationError{
			Field:   "l2.engine_addr",
			Message: "L2 engine address cannot be empty",
			Value:   cfg.L2EngineAddr,
		}
	}
	
	if _, err := url.Parse(cfg.L2EngineAddr); err != nil {
		return &ValidationError{
			Field:   "l2.engine_addr",
			Message: fmt.Sprintf("L2 engine address must be a valid URL: %v", err),
			Value:   cfg.L2EngineAddr,
		}
	}
	
	if cfg.L2EngineJWTSecret == nil || len(cfg.L2EngineJWTSecret) == 0 {
		return &ValidationError{
			Field:   "l2.jwt_secret",
			Message: "L2 JWT secret cannot be empty",
			Value:   "***",
		}
	}
	
	if cfg.L2EngineCallTimeout <= 0 {
		return &ValidationError{
			Field:   "l2.call_timeout",
			Message: "L2 call timeout must be positive",
			Value:   cfg.L2EngineCallTimeout,
		}
	}
	
	if cfg.L2EngineCallTimeout < time.Second {
		return &ValidationError{
			Field:   "l2.call_timeout",
			Message: "L2 call timeout must be at least 1 second",
			Value:   cfg.L2EngineCallTimeout,
		}
	}
	
	return nil
}

func validateRPCConfig(cfg oprpc.CLIConfig) error {
	if cfg.Addr == "" {
		return &ValidationError{
			Field:   "rpc.addr",
			Message: "RPC address cannot be empty",
			Value:   cfg.Addr,
		}
	}
	
	if cfg.Port < 1 || cfg.Port > 65535 {
		return &ValidationError{
			Field:   "rpc.port",
			Message: "RPC port must be between 1 and 65535",
			Value:   cfg.Port,
		}
	}
	
	return nil
}

func validateMetricsConfig(cfg opmetrics.CLIConfig) error {
	if cfg.Enabled {
		if cfg.ListenAddr == "" {
			return &ValidationError{
				Field:   "metrics.addr",
				Message: "Metrics address cannot be empty when metrics are enabled",
				Value:   cfg.ListenAddr,
			}
		}
		
		if cfg.ListenPort < 1 || cfg.ListenPort > 65535 {
			return &ValidationError{
				Field:   "metrics.port",
				Message: "Metrics port must be between 1 and 65535",
				Value:   cfg.ListenPort,
			}
		}
	}
	
	return nil
}

