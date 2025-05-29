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
use tracing_subscriber::fmt::writer::BoxMakeWriter;
use tracing_subscriber::layer::SubscriberExt;

use crate::cli::{Args, LogFormat};

/// Span attribute keys that should be recorded as metric labels.
///
/// Use caution when adding new attributes here and keep
/// label cardinality in mind. Not all span attributes make
/// appropriate labels.
pub const SPAN_ATTRIBUTE_LABELS: [&str; 4] = ["code", "payload_source", "method", "builder_has_payload"];

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
            .chain([
                ("span_kind".to_string(), format!("{:?}", span.span_kind)),
                ("status".to_string(), status.into()),
            ])
            .collect::<Vec<_>>();

        // 0 = no difference in gas build via builder vs l2
        // > 0 = gas used by builder block is greater than l2 block
        // < 0 = gas used by l2 block is greater than builder block
        let gas_delta = span
            .attributes
            .iter()
            .find(|attr| attr.key.as_str() == "gas_delta")
            .map(|attr| attr.value.as_str().to_string());

        if let Some(gas_delta) = gas_delta {
            histogram!("block_building_gas_delta", &labels)
                .record(gas_delta.parse::<u64>().unwrap_or_default() as f64);
        }

        // 0 = no difference in tx count build via builder vs l2
        // > 0 = num txs in builder block is greater than l2 block
        // < 0 = num txs in l2 block is greater than builder block
        let tx_count_delta = span
            .attributes
            .iter()
            .find(|attr| attr.key.as_str() == "tx_count_delta")
            .map(|attr| attr.value.as_str().to_string());

        if let Some(tx_count_delta) = tx_count_delta {
            histogram!("block_building_tx_count_delta", &labels)
                .record(tx_count_delta.parse::<u64>().unwrap_or_default() as f64);
        }

        histogram!(format!("{}_duration", span.name), &labels).record(duration);
    }

    fn force_flush(&self) -> opentelemetry_sdk::error::OTelSdkResult {
        Ok(())
    }

    fn shutdown(&self) -> opentelemetry_sdk::error::OTelSdkResult {
        Ok(())
    }
}

pub fn init_tracing(args: &Args) -> eyre::Result<()> {
    // Be cautious with snake_case and kebab-case here
    let filter_name = "rollup_boost".to_string();

    let global_filter = Targets::new()
        .with_default(LevelFilter::INFO)
        .with_target(&filter_name, LevelFilter::TRACE);

    let registry = tracing_subscriber::registry().with(global_filter);

    let log_filter = Targets::new()
        .with_default(LevelFilter::INFO)
        .with_target(&filter_name, args.log_level);

    let writer = if let Some(path) = &args.log_file {
        let file = std::fs::OpenOptions::new()
            .create(true)
            .append(true)
            .open(path)
            .context("Failed to open log file")?;
        BoxMakeWriter::new(file)
    } else {
        BoxMakeWriter::new(std::io::stdout)
    };

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
                            .with_writer(writer)
                            .with_filter(log_filter.clone()),
                    ),
                )?;
            }
            LogFormat::Text => {
                tracing::subscriber::set_global_default(
                    registry.with(
                        tracing_subscriber::fmt::layer()
                            .with_ansi(false)
                            .with_writer(writer)
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
                            .with_writer(writer)
                            .with_filter(log_filter.clone()),
                    ),
                )?;
            }
            LogFormat::Text => {
                tracing::subscriber::set_global_default(
                    registry.with(
                        tracing_subscriber::fmt::layer()
                            .with_ansi(false)
                            .with_writer(writer)
                            .with_filter(log_filter.clone()),
                    ),
                )?;
            }
        }
    }

    Ok(())
}
