package telemetry

import (
	"os"

	"github.com/honeycombio/otel-config-go/otelconfig"
)

func SetupOpenTelemetry(svc string) (func(), error) {
	// do not use localhost:4317 by default, we want telemetry to be opt-in and
	// explicit.
	if os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") == "" {
		return func() {}, nil
	}

	otelShutdown, err := otelconfig.ConfigureOpenTelemetry(
		otelconfig.WithServiceName(svc),
	)
	if err != nil {
		return nil, err
	}
	return otelShutdown, nil
}
