//! Utility module to house implementation and declaration of MetricsArgs since it's being used in
//! multiple places, it's just being referenced from this module.

use crate::{CliResult, init_prometheus_server};
use clap::{Parser, arg};
use std::net::IpAddr;

/// Configuration for Prometheus metrics.
#[derive(Debug, Clone, Parser)]
#[command(next_help_heading = "Metrics")]
pub struct MetricsArgs {
    /// Controls whether Prometheus metrics are enabled. Disabled by default.
    #[arg(
        long = "metrics.enabled",
        global = true,
        default_value_t = false,
        env = "KONA_METRICS_ENABLED"
    )]
    pub enabled: bool,

    /// The port to serve Prometheus metrics on.
    #[arg(long = "metrics.port", global = true, default_value = "9090", env = "KONA_METRICS_PORT")]
    pub port: u16,

    /// The IP address to use for Prometheus metrics.
    #[arg(
        long = "metrics.addr",
        global = true,
        default_value = "0.0.0.0",
        env = "KONA_METRICS_ADDR"
    )]
    pub addr: IpAddr,
}

impl Default for MetricsArgs {
    fn default() -> Self {
        Self::parse_from::<[_; 0], &str>([])
    }
}

impl MetricsArgs {
    /// Initialize the tracing stack and Prometheus metrics recorder.
    ///
    /// This function should be called at the beginning of the program.
    pub fn init_metrics(&self) -> CliResult<()> {
        if self.enabled {
            init_prometheus_server(self.addr, self.port)?;
        }

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use clap::Parser;
    use std::net::{IpAddr, Ipv4Addr};

    /// Helper struct to parse MetricsArgs within a test CLI structure.
    #[derive(Parser, Debug)]
    struct TestCli {
        #[command(flatten)]
        metrics: MetricsArgs,
    }

    #[test]
    fn test_default_metrics_args() {
        let cli = TestCli::parse_from(["test_app"]);
        assert!(!cli.metrics.enabled, "Default for metrics.enabled should be false.");
        assert_eq!(cli.metrics.port, 9090, "Default for metrics.port should be 9090.");
        assert_eq!(
            cli.metrics.addr,
            IpAddr::V4(Ipv4Addr::new(0, 0, 0, 0)),
            "Default for metrics.addr should be 0.0.0.0."
        );
    }

    #[test]
    fn test_metrics_args_from_cli() {
        let cli = TestCli::parse_from([
            "test_app",
            "--metrics.enabled",
            "--metrics.port",
            "9999",
            "--metrics.addr",
            "127.0.0.1",
        ]);
        assert!(cli.metrics.enabled, "metrics.enabled should be true.");
        assert_eq!(cli.metrics.port, 9999, "metrics.port should be parsed from CLI.");
        assert_eq!(
            cli.metrics.addr,
            IpAddr::V4(Ipv4Addr::new(127, 0, 0, 1)),
            "metrics.addr should be parsed from CLI."
        );
    }
}
