use eyre::Context as _;
use metrics::histogram;
use opentelemetry::trace::{Status, TracerProvider as _};
use opentelemetry::{KeyValue, global};
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::trace::SpanProcessor;
use opentelemetry_sdk::{Resource, propagation::TraceContextPropagator};
use tracing::level_filters::LevelFilter;
use tracing_opentelemetry::OpenTelemetryLayer;
use tracing_subscriber::Layer;
use tracing_subscriber::filter::Targets;
use tracing_subscriber::layer::SubscriberExt;

use crate::{Args, LogFormat};

/// Span attribute keys that should be recorded as metric labels.
///
/// Use caution when adding new attributes here and keep
/// label cardinality in mind. Not all span attributes make
/// appropriate labels.
pub const SPAN_ATTRIBUTE_LABELS: [&str; 3] = ["code", "has_attributes", "payload_source"];

/// Custom span processor that records span durations as histograms
#[derive(Debug)]
struct MetricsSpanProcessor;

impl SpanProcessor for MetricsSpanProcessor {
    fn on_start(&self, _span: &mut opentelemetry_sdk::trace::Span, _cx: &opentelemetry::Context) {}

    fn on_end(&self, span: opentelemetry_sdk::trace::SpanData) {
        let duration = span
            .end_time
            .duration_since(span.start_time)
            .unwrap_or_default();

        // Remove status description to avoid cardinality explosion
        let status = match span.status {
            Status::Ok => "ok",
            Status::Error { .. } => "error",
            Status::Unset => "unset",
        };

        // Add custom labels
        let labels = span
            .attributes
            .iter()
            .filter(|attr| SPAN_ATTRIBUTE_LABELS.contains(&attr.key.as_str()))
            .map(|attr| {
                (
                    attr.key.as_str().to_string(),
                    attr.value.as_str().to_string(),
                )
            })
            .chain(
                [
                    ("span_kind".to_string(), format!("{:?}", span.span_kind)),
                    ("status".to_string(), status.into()),
                ]
                .into_iter(),
            )
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
    // Be cautious with snake_case and kebab-case here
    let filter_name = "rollup_boost".to_string();

    let global_filter = Targets::new()
        .with_default(LevelFilter::INFO)
        .with_target(&filter_name, LevelFilter::TRACE);

    let registry = tracing_subscriber::registry().with(global_filter);

    let log_filter = Targets::new()
        .with_default(LevelFilter::INFO)
        .with_target(&filter_name, args.log_level);

    // Weird control flow here is required because of type system
    if args.tracing {
        global::set_text_map_propagator(TraceContextPropagator::new());
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

        let trace_filter = Targets::new()
            .with_default(LevelFilter::OFF)
            .with_target(&filter_name, LevelFilter::TRACE);

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
