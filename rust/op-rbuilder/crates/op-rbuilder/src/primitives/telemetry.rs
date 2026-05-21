use crate::args::TelemetryArgs;
use tracing_subscriber::{Layer, filter::Targets};
use url::Url;

/// Setup telemetry layer with sampling and custom endpoint configuration
pub fn setup_telemetry_layer(
    args: &TelemetryArgs,
) -> eyre::Result<impl Layer<tracing_subscriber::Registry>> {
    use tracing::level_filters::LevelFilter;

    let Some(otlp_endpoint) = &args.otlp_endpoint else {
        return Err(eyre::eyre!("OTLP endpoint is not set"));
    };

    // OTLP uses env vars internally.
    if let Some(headers) = &args.otlp_headers {
        unsafe { std::env::set_var("OTEL_EXPORTER_OTLP_HEADERS", headers) };
    }

    // Create OTLP layer with custom configuration
    let protocol = reth_tracing_otlp::OtlpProtocol::Http;
    let mut endpoint = Url::parse(otlp_endpoint)?;
    protocol.validate_endpoint(&mut endpoint)?;
    let otlp_config = reth_tracing_otlp::OtlpConfig::new("op-rbuilder", endpoint, protocol, None)?;
    let otlp_layer = reth_tracing_otlp::span_layer(otlp_config)?;

    // Create a trace filter that sends more data to OTLP but less to stdout
    let trace_filter = Targets::new()
        .with_default(LevelFilter::WARN)
        .with_target("op_rbuilder", LevelFilter::INFO)
        .with_target("payload_builder", LevelFilter::DEBUG);

    let filtered_layer = otlp_layer.with_filter(trace_filter);

    Ok(filtered_layer)
}
