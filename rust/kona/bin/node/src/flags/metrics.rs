//! Prometheus metrics CLI args
//!
//! Specifies the available flags for prometheus metric configuration inside CLI

use clap::{ArgAction, Args};
use kona_cli::init_prometheus_server;
use metrics::gauge;
use std::net::IpAddr;

/// Prometheus metrics CLI Flags
///
/// The address and port apply only with `--metrics.enabled`, and are ignored otherwise.
#[derive(Args, Debug, Clone, PartialEq, Eq)]
#[command(next_help_heading = "Metrics")]
pub struct MetricsArgs {
    /// Serve Prometheus metrics. Disabled by default.
    #[arg(
        id = "metrics_enabled",
        long = "metrics.enabled",
        env = "KONA_NODE_METRICS_ENABLED",
        action = ArgAction::Set,
        num_args = 0..=1,
        require_equals = true,
        default_value_t = false,
        default_missing_value = "true"
    )]
    pub enabled: bool,
    /// The IP address to serve Prometheus metrics on.
    #[arg(long = "metrics.addr", default_value = "0.0.0.0", env = "KONA_NODE_METRICS_ADDR")]
    pub addr: IpAddr,
    /// The port to serve Prometheus metrics on.
    #[arg(long = "metrics.port", default_value = "9090", env = "KONA_NODE_METRICS_PORT")]
    pub port: u16,
}

/// Initializes metrics for a Kona application, including Prometheus and node-specific metrics.
/// Initialize the tracing stack and Prometheus metrics recorder.
///
/// This function should be called at the beginning of the program.
pub fn init_unified_metrics(args: &MetricsArgs) -> anyhow::Result<()> {
    if !args.enabled {
        return Ok(());
    }
    init_prometheus_server(args.addr, args.port)?;
    kona_gossip::Metrics::init();
    kona_disc::Metrics::init();
    kona_sources::Metrics::init();
    kona_node_service::Metrics::init();
    kona_derive::Metrics::init();
    kona_providers_alloy::Metrics::init();
    kona_providers_local::Metrics::init();
    gauge!(
        "kona_node_info",
        &[
            ("version", crate::version::version()),
            ("build_timestamp", crate::version::build_timestamp()),
            ("cargo_features", crate::version::cargo_features()),
            ("git_sha", crate::version::git_sha()),
            ("target_triple", crate::version::target_triple()),
            ("build_profile", crate::version::build_profile()),
        ]
    )
    .set(1);
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use clap::Parser;
    use std::net::Ipv4Addr;

    #[derive(Parser, Debug)]
    struct TestCli {
        #[command(flatten)]
        metrics: MetricsArgs,
    }

    fn parse(args: &[&str]) -> Result<MetricsArgs, clap::Error> {
        TestCli::try_parse_from(std::iter::once("test").chain(args.iter().copied()))
            .map(|cli| cli.metrics)
    }

    #[test]
    fn metrics_are_off_by_default() {
        let metrics = parse(&[]).unwrap();
        assert!(!metrics.enabled);
        assert_eq!(metrics.addr, IpAddr::V4(Ipv4Addr::UNSPECIFIED));
        assert_eq!(metrics.port, 9090);
    }

    #[test]
    fn serves_on_the_address_and_port_given() {
        let metrics =
            parse(&["--metrics.enabled", "--metrics.addr", "127.0.0.1", "--metrics.port", "1234"])
                .unwrap();
        assert!(metrics.enabled);
        assert_eq!(metrics.addr, IpAddr::V4(Ipv4Addr::LOCALHOST));
        assert_eq!(metrics.port, 1234);
    }

    /// A script can turn metrics off without unsetting their address and port.
    #[test]
    fn the_address_and_port_are_ignored_while_metrics_are_off() {
        let metrics = parse(&["--metrics.enabled=false", "--metrics.port", "1234"]).unwrap();
        assert!(!metrics.enabled);
    }

    #[test]
    fn the_environment_variable_takes_true_or_false() {
        // SAFETY: nextest runs each test in its own process.
        unsafe { std::env::set_var("KONA_NODE_METRICS_ENABLED", "true") };
        let enabled = parse(&[]);
        // SAFETY: as above.
        unsafe { std::env::set_var("KONA_NODE_METRICS_ENABLED", "false") };
        let disabled = parse(&["--metrics.port", "1234"]);
        // SAFETY: as above.
        unsafe { std::env::remove_var("KONA_NODE_METRICS_ENABLED") };

        assert!(enabled.unwrap().enabled);
        assert!(!disabled.unwrap().enabled);
    }
}
