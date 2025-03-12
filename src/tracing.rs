use eyre::Context as _;
use opentelemetry::trace::TracerProvider as _;
use opentelemetry::{global, KeyValue};
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::{propagation::TraceContextPropagator, Resource};
use tracing_opentelemetry::OpenTelemetryLayer;
use tracing_subscriber::{layer::SubscriberExt, EnvFilter};

use crate::{Args, LogFormat};

pub(crate) fn init_tracing(args: &Args) -> eyre::Result<()> {
    let log_level = args.log_level.to_string();
    let registry = tracing_subscriber::registry().with(EnvFilter::new(log_level));

    // Weird control flow here is required because of type system
    if args.tracing {
        global::set_text_map_propagator(TraceContextPropagator::new());
        let otlp_exporter = opentelemetry_otlp::SpanExporter::builder()
            .with_tonic()
            .with_endpoint(&args.otlp_endpoint)
            .build()
            .context("Failed to create OTLP exporter")?;
        let provider = opentelemetry_sdk::trace::SdkTracerProvider::builder()
            .with_batch_exporter(otlp_exporter)
            .with_resource(
                Resource::builder_empty()
                    .with_attributes([
                        KeyValue::new("service.name", env!("CARGO_PKG_NAME")),
                        KeyValue::new("service.version", env!("CARGO_PKG_VERSION")),
                    ])
                    .build(),
            )
            .build();

        let tracer = provider.tracer(env!("CARGO_PKG_NAME"));
        let registry = registry.with(OpenTelemetryLayer::new(tracer));

        match args.log_format {
            LogFormat::Json => {
                tracing::subscriber::set_global_default(registry.with(
                    tracing_subscriber::fmt::layer().json().with_ansi(false), // Disable colored logging
                ))?;
            }
            LogFormat::Text => {
                tracing::subscriber::set_global_default(
                    registry.with(tracing_subscriber::fmt::layer()),
                )?;
            }
        }
    } else {
        match args.log_format {
            LogFormat::Json => {
                tracing::subscriber::set_global_default(
                    registry.with(tracing_subscriber::fmt::layer().json()),
                )?;
            }
            LogFormat::Text => {
                tracing::subscriber::set_global_default(
                    registry.with(tracing_subscriber::fmt::layer()),
                )?;
            }
        }
    }

    Ok(())
}
