package telemetry

import (
	"context"
	"os"

	"github.com/honeycombio/otel-config-go/otelconfig"
)

const (
	serviceNameEnvVar    = "OTEL_SERVICE_NAME"
	serviceVersionEnvVar = "OTEL_SERVICE_VERSION"
)

var (
	serviceName    = envOrDefault(serviceNameEnvVar, "devstack")
	serviceVersion = envOrDefault(serviceVersionEnvVar, "0.0.0")
)

func envOrDefault(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func SetupOpenTelemetry(ctx context.Context, opts ...otelconfig.Option) (context.Context, func(), error) {
	// do not use localhost:4317 by default, we want telemetry to be opt-in and
	// explicit.
	if os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") == "" {
		return ctx, func() {}, nil
	}

	opts = append([]otelconfig.Option{
		otelconfig.WithServiceName(serviceName),
		otelconfig.WithServiceVersion(serviceVersion),
		otelconfig.WithPropagators(defaultPropagators),
	}, opts...)
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
