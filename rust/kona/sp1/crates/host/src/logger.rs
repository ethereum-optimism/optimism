//! Logger setup with optional `OpenTelemetry` export.

use std::{env, sync::OnceLock};

use anyhow::{Context, Result};
use opentelemetry::{KeyValue, global};
use opentelemetry_appender_tracing::layer::OpenTelemetryTracingBridge;
use opentelemetry_otlp::{ExportConfig, LogExporter, Protocol, WithExportConfig};
use opentelemetry_sdk::{Resource, logs::SdkLoggerProvider, propagation::TraceContextPropagator};
use tracing_subscriber::{
    EnvFilter, Layer, Registry, fmt::format::JsonFields, layer::SubscriberExt,
    util::SubscriberInitExt,
};

static INIT: OnceLock<Result<()>> = OnceLock::new();

fn build_env_filter() -> EnvFilter {
    let mut filter = EnvFilter::new("info")
        .add_directive("single_hint_handler=error".parse().unwrap())
        .add_directive("execute=error".parse().unwrap())
        .add_directive("sp1_prover=error".parse().unwrap())
        .add_directive("boot_loader=error".parse().unwrap())
        .add_directive("client_executor=error".parse().unwrap())
        .add_directive("client=error".parse().unwrap())
        .add_directive("channel_assembler=error".parse().unwrap())
        .add_directive("attributes_queue=error".parse().unwrap())
        .add_directive("batch_validator=error".parse().unwrap())
        .add_directive("batch_queue=error".parse().unwrap())
        .add_directive("client_derivation_driver=error".parse().unwrap())
        .add_directive("block_builder=error".parse().unwrap())
        .add_directive("host_server=error".parse().unwrap())
        .add_directive("kona_protocol=error".parse().unwrap())
        .add_directive("sp1_core_executor=off".parse().unwrap())
        .add_directive("sp1_core_machine=error".parse().unwrap());

    if let Ok(var) = env::var(EnvFilter::DEFAULT_ENV) {
        for directive in var.split(',') {
            match directive.trim().parse() {
                Ok(d) => filter = filter.add_directive(d),
                Err(e) => eprintln!("ignoring invalid RUST_LOG directive {directive:?}: {e}"),
            }
        }
    }

    filter
}

/// Set up the logger with optional `OpenTelemetry` export.
///
/// # Environment Variables
/// - `LOGGER_NAME`: Service name for opentelemetry logs (defaults to `kona-sp1`)
/// - `OTLP_ENDPOINT`: `OpenTelemetry` endpoint (defaults to <http://localhost:4317>)
/// - `OTLP_ENABLED`: Whether to enable `OpenTelemetry` export (defaults to false)
/// - `RUST_LOG`: Standard Rust log level configuration
/// - `LOG_FORMAT`: Output format (pretty or json, defaults to pretty)
pub fn setup_logger() {
    INIT.get_or_init(|| {
        let logger_name = std::env::var("LOGGER_NAME").ok();
        let otlp_endpoint =
            std::env::var("OTLP_ENDPOINT").unwrap_or_else(|_| "http://localhost:4317".to_string());

        let otlp_enabled = std::env::var("OTLP_ENABLED")
            .unwrap_or_else(|_| "false".to_string())
            .parse::<bool>()
            .unwrap_or(false);

        let service_name = logger_name.unwrap_or_else(|| "kona-sp1".to_string());

        let params = vec![
            KeyValue::new("service.name", service_name),
            KeyValue::new("service.version", env!("CARGO_PKG_VERSION").to_string()),
        ];

        let resource = Resource::builder_empty().with_attributes(params).build();
        global::set_text_map_propagator(TraceContextPropagator::new());

        let log_format = std::env::var("LOG_FORMAT").unwrap_or_else(|_| "pretty".to_string());
        let fmt_layer: Option<Box<dyn Layer<_> + Send + Sync>> =
            match log_format.to_lowercase().as_str() {
                "json" => {
                    // Initialize with JSON formatting
                    Some(Box::new(
                        tracing_subscriber::fmt::layer()
                            .event_format(tracing_subscriber::fmt::format().json())
                            .fmt_fields(JsonFields::new())
                            .with_filter(build_env_filter()),
                    ))
                }
                _ => {
                    // Default to pretty formatting with ANSI colors
                    let ansi = cfg!(feature = "ansi") &&
                        env::var("NO_COLOR").map_or(true, |v| v.is_empty());

                    Some(Box::new(
                        tracing_subscriber::fmt::layer()
                            .with_level(true)
                            .with_target(false)
                            .with_thread_ids(false)
                            .with_thread_names(false)
                            .with_file(false)
                            .with_line_number(false)
                            .with_ansi(ansi)
                            .with_filter(build_env_filter()),
                    ))
                }
            };

        let log_export_layer: Option<Box<dyn Layer<_> + Send + Sync>> = if otlp_enabled {
            let export_layer = LogExporter::builder()
                .with_tonic()
                .with_export_config(ExportConfig {
                    endpoint: Some(otlp_endpoint.clone()),
                    protocol: Protocol::Grpc,
                    ..Default::default()
                })
                .build()
                .context("Failed to create OpenTelemetry log exporter")?;
            let logger_provider = SdkLoggerProvider::builder()
                .with_resource(resource)
                .with_batch_exporter(export_layer)
                .build();
            Some(Box::new(
                OpenTelemetryTracingBridge::new(&logger_provider).with_filter(build_env_filter()),
            ))
        } else {
            None
        };

        Registry::default().with(log_export_layer).with(fmt_layer).init();
        if otlp_enabled {
            tracing::info!("OTLP endpoint configured: {}", otlp_endpoint);
        }
        Ok(())
    });
}
