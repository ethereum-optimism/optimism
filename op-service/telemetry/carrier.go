package telemetry

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/propagation"
)

const CarrierEnvVarPrefix = "OTEL_DEVSTACK_PROPAGATOR_CARRIER_"

// keep in sync with textPropagator() below
var defaultPropagators = []string{
	"tracecontext",
	"baggage",
}

func textPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		// keep in sync with propagators above
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func InstrumentEnvironment(ctx context.Context, env []string) []string {
	propagator := textPropagator()
	carrier := propagation.MapCarrier{}
	propagator.Inject(ctx, carrier)

	for k, v := range carrier {
		env = append(env, fmt.Sprintf("%s%s=%s", CarrierEnvVarPrefix, k, v))
	}

	return env
}

func ExtractEnvironment(ctx context.Context, env []string) (context.Context, error) {
	carrier := propagation.MapCarrier{}
	// Reconstruct the carrier from the environment variables
	for _, e := range env {
		if strings.HasPrefix(e, CarrierEnvVarPrefix) {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimPrefix(parts[0], CarrierEnvVarPrefix)
				value := parts[1]
				carrier.Set(key, value)
			}
		}
	}

	ctx = textPropagator().Extract(ctx, carrier)
	return ctx, nil
}
