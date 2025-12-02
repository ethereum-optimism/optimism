//! Prometheus metrics CLI args
//!
//! Specifies the available flags for prometheus metric configuration inside CLI

use crate::metrics::VersionInfo;
use kona_cli::MetricsArgs;

/// Initializes metrics for a Kona application, including Prometheus and node-specific metrics.
/// Initialize the tracing stack and Prometheus metrics recorder.
///
/// This function should be called at the beginning of the program.
pub fn init_unified_metrics(args: &MetricsArgs) -> anyhow::Result<()> {
    args.init_metrics()?;
    if args.enabled {
        kona_gossip::Metrics::init();
        kona_disc::Metrics::init();
        kona_engine::Metrics::init();
        kona_node_service::Metrics::init();
        kona_derive::Metrics::init();
        kona_providers_alloy::Metrics::init();
        VersionInfo::from_build().register_version_metrics();
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use clap::Parser;
    use std::net::IpAddr;

    /// A mock command that uses the MetricsArgs.
    #[derive(Parser, Debug, Clone)]
    #[command(about = "Mock command")]
    struct MockCommand {
        /// Metrics CLI Flags
        #[clap(flatten)]
        pub metrics: MetricsArgs,
    }

    #[test]
    fn test_metrics_args_listen_enabled() {
        let args = MockCommand::parse_from(["test", "--metrics.enabled"]);
        assert!(args.metrics.enabled);

        let args = MockCommand::parse_from(["test"]);
        assert!(!args.metrics.enabled);
    }

    #[test]
    fn test_metrics_args_listen_ip() {
        let args = MockCommand::parse_from(["test", "--metrics.addr", "127.0.0.1"]);
        let expected: IpAddr = "127.0.0.1".parse().unwrap();
        assert_eq!(args.metrics.addr, expected);
    }

    #[test]
    fn test_metrics_args_listen_port() {
        let args = MockCommand::parse_from(["test", "--metrics.port", "1234"]);
        assert_eq!(args.metrics.port, 1234);
    }
}
