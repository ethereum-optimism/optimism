package telemetry

import (
	"context"
	"os"

	"github.com/honeycombio/otel-config-go/otelconfig"
)

const (
	serviceNameEnvVar     = "OTEL_SERVICE_NAME"
	serviceVersionEnvVar  = "OTEL_SERVICE_VERSION"
	tracesEndpointEnvVar  = "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"
	metricsEndpointEnvVar = "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"

	defaultServiceName    = "devstack"
	defaultServiceVersion = "0.0.0"
)

func envOrDefault(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func SetupOpenTelemetry(ctx context.Context, opts ...otelconfig.Option) (context.Context, func(), error) {
	defaultOpts := []otelconfig.Option{
		otelconfig.WithServiceName(envOrDefault(serviceNameEnvVar, defaultServiceName)),
		otelconfig.WithServiceVersion(envOrDefault(serviceVersionEnvVar, defaultServiceVersion)),
		otelconfig.WithPropagators(defaultPropagators),
	}

	// do not use localhost:4317 by default, we want telemetry to be opt-in and
	// explicit.
	// The caller is still able to override this by passing in their own opts.
	if os.Getenv(tracesEndpointEnvVar) == "" {
		defaultOpts = append(defaultOpts, otelconfig.WithTracesEnabled(false))
	}
	if os.Getenv(metricsEndpointEnvVar) == "" {
		defaultOpts = append(defaultOpts, otelconfig.WithMetricsEnabled(false))
	}

	opts = append(defaultOpts, opts...)
	otelShutdown, err := otelconfig.ConfigureOpenTelemetry(opts...)
	if err != nil {
		return ctx, nil, err
	}

	// If the environment contains carrier information, extract it.
	// This is useful for test runner / test communication for example.
	ctx, err = ExtractEnvironment(ctx, os.Environ())
	if err != nil {
		return ctx, nil, err
	}

	return ctx, otelShutdown, nil
}
