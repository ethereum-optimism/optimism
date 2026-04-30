use crate::args::TelemetryArgs;
use tracing_subscriber::{Layer, filter::Targets};
use url::Url;

/// Setup telemetry layer with sampling and custom endpoint configuration
pub fn setup_telemetry_layer(
    args: &TelemetryArgs,
) -> eyre::Result<impl Layer<tracing_subscriber::Registry>> {
    use tracing::level_filters::LevelFilter;

    if args.otlp_endpoint.is_none() {
        return Err(eyre::eyre!("OTLP endpoint is not set"));
    }

    // Otlp uses evn vars inside

    if let Some(headers) = &args.otlp_headers {
        unsafe { std::env::set_var("OTEL_EXPORTER_OTLP_HEADERS", headers) };
    }

    // Create OTLP layer with custom configuration
    let otlp_layer = reth_tracing_otlp::span_layer(
        "op-rbuilder",
        &Url::parse(args.otlp_endpoint.as_ref().unwrap()).expect("Invalid OTLP endpoint"),
        reth_tracing_otlp::OtlpProtocol::Http,
    )?;

    // Create a trace filter that sends more data to OTLP but less to stdout
    let trace_filter = Targets::new()
        .with_default(LevelFilter::WARN)
        .with_target("op_rbuilder", LevelFilter::INFO)
        .with_target("payload_builder", LevelFilter::DEBUG);

    let filtered_layer = otlp_layer.with_filter(trace_filter);

    Ok(filtered_layer)
}
