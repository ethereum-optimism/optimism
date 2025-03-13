use eyre::Context as _;
use metrics::histogram;
use opentelemetry::trace::TracerProvider as _;
use opentelemetry::{global, KeyValue};
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::trace::SpanProcessor;
use opentelemetry_sdk::{propagation::TraceContextPropagator, Resource};
use tracing::level_filters::LevelFilter;
use tracing_opentelemetry::OpenTelemetryLayer;
use tracing_subscriber::filter::Targets;
use tracing_subscriber::layer::SubscriberExt;
use tracing_subscriber::Layer;

use crate::{Args, LogFormat};

#[derive(Debug)]
struct MetricsSpanProcessor;

impl SpanProcessor for MetricsSpanProcessor {
    fn on_start(&self, _span: &mut opentelemetry_sdk::trace::Span, _cx: &opentelemetry::Context) {}

    fn on_end(&self, span: opentelemetry_sdk::trace::SpanData) {
        let duration = span
            .end_time
            .duration_since(span.start_time)
            .unwrap_or_default();

        let labels = span
            .attributes
            .iter()
            .map(|kv| (kv.key.to_string(), kv.value.to_string()))
            .chain(std::iter::once((
                "span_kind".to_string(),
                format!("{:?}", span.span_kind),
            )))
            .collect::<Vec<_>>();

        histogram!(format!("{}_duration", span.name), &labels).record(duration);
    }

    fn force_flush(&self) -> opentelemetry_sdk::error::OTelSdkResult {
        Ok(())
    }

    fn shutdown(&self) -> opentelemetry_sdk::error::OTelSdkResult {
        Ok(())
    }
}

pub(crate) fn init_tracing(args: &Args) -> eyre::Result<()> {
    let log_filter = Targets::new()
        .with_default(LevelFilter::INFO)
        .with_target("rollup-boost", args.log_level);

    let registry = tracing_subscriber::registry();

    // Weird control flow here is required because of type system
    if args.tracing {
        global::set_text_map_propagator(TraceContextPropagator::new());
        // Define a custom sampler that checks if the span's name starts with your crate's name.
        let otlp_exporter = opentelemetry_otlp::SpanExporter::builder()
            .with_tonic()
            .with_endpoint(&args.otlp_endpoint)
            .build()
            .context("Failed to create OTLP exporter")?;
        let mut provider_builder = opentelemetry_sdk::trace::SdkTracerProvider::builder()
            .with_batch_exporter(otlp_exporter)
            .with_resource(
                Resource::builder_empty()
                    .with_attributes([
                        KeyValue::new("service.name", env!("CARGO_PKG_NAME")),
                        KeyValue::new("service.version", env!("CARGO_PKG_VERSION")),
                    ])
                    .build(),
            );
        if args.metrics {
            provider_builder = provider_builder.with_span_processor(MetricsSpanProcessor);
        }
        let provider = provider_builder.build();
        let tracer = provider.tracer(env!("CARGO_PKG_NAME"));

        // Set global tracer provider for use with opentelemetry api
        global::set_tracer_provider(provider);

        let trace_filter = Targets::new()
            .with_default(LevelFilter::OFF)
            .with_target(env!("CARGO_PKG_NAME"), LevelFilter::TRACE);

        let registry = registry.with(OpenTelemetryLayer::new(tracer).with_filter(trace_filter));

        match args.log_format {
            LogFormat::Json => {
                tracing::subscriber::set_global_default(
                    registry.with(
                        tracing_subscriber::fmt::layer()
                            .json()
                            .with_ansi(false)
                            .with_filter(log_filter.clone()),
                    ),
                )?;
            }
            LogFormat::Text => {
                tracing::subscriber::set_global_default(
                    registry.with(
                        tracing_subscriber::fmt::layer()
                            .with_ansi(false)
                            .with_filter(log_filter.clone()),
                    ),
                )?;
            }
        }
    } else {
        match args.log_format {
            LogFormat::Json => {
                tracing::subscriber::set_global_default(
                    registry.with(
                        tracing_subscriber::fmt::layer()
                            .json()
                            .with_ansi(false)
                            .with_filter(log_filter.clone()),
                    ),
                )?;
            }
            LogFormat::Text => {
                tracing::subscriber::set_global_default(
                    registry.with(
                        tracing_subscriber::fmt::layer()
                            .with_ansi(false)
                            .with_filter(log_filter.clone()),
                    ),
                )?;
            }
        }
    }

    Ok(())
}
