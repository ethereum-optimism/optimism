//! Prometheus metrics CLI args
//!
//! Specifies the available flags for prometheus metric configuration inside CLI

use kona_cli::MetricsArgs;
use metrics::gauge;

/// Initializes metrics for a Kona application, including Prometheus and node-specific metrics.
/// Initialize the tracing stack and Prometheus metrics recorder.
///
/// This function should be called at the beginning of the program.
pub fn init_unified_metrics(args: &MetricsArgs, l2_chain_id: u64) -> anyhow::Result<()> {
    args.init_metrics()?;
    if args.enabled {
        kona_gossip::Metrics::init(l2_chain_id);
        kona_disc::Metrics::init(l2_chain_id);
        kona_engine::Metrics::init(l2_chain_id);
        kona_node_service::Metrics::init(l2_chain_id);
        kona_derive::Metrics::init(l2_chain_id);
        kona_providers_alloy::Metrics::init(l2_chain_id);
        kona_providers_local::Metrics::init(l2_chain_id);
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
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use clap::Parser;
    use std::net::IpAddr;

    /// A mock command that uses the `MetricsArgs`.
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
