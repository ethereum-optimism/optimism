//! Utility module to house implementation and declaration of `MetricsArgs` since it's being used in
//! multiple places, it's just being referenced from this module.

use crate::{CliResult, init_prometheus_server, parse_enabled};
use clap::{ArgAction, Parser};
use std::net::IpAddr;

/// The id of `--metrics.enabled`, which the other metrics flags require.
const METRICS_ENABLED: &str = "metrics_enabled";

/// Configuration for Prometheus metrics.
#[derive(Debug, Clone, Parser)]
#[command(next_help_heading = "Metrics")]
pub struct MetricsArgs {
    /// Serve Prometheus metrics. Disabled by default. The environment variable must be `true` or
    /// unset.
    // Read it with `MetricsArgs::is_enabled`.
    #[arg(
        id = METRICS_ENABLED,
        long = "metrics.enabled",
        env = "KONA_METRICS_ENABLED",
        action = ArgAction::Set,
        num_args = 0,
        default_missing_value = "true",
        value_parser = parse_enabled
    )]
    pub enabled: Option<bool>,

    /// The port to serve Prometheus metrics on. Requires `--metrics.enabled`.
    #[arg(
        long = "metrics.port",
        default_value = "9090",
        env = "KONA_METRICS_PORT",
        requires = METRICS_ENABLED
    )]
    pub port: u16,

    /// The IP address to serve Prometheus metrics on. Requires `--metrics.enabled`.
    #[arg(
        long = "metrics.addr",
        default_value = "0.0.0.0",
        env = "KONA_METRICS_ADDR",
        requires = METRICS_ENABLED
    )]
    pub addr: IpAddr,
}

impl Default for MetricsArgs {
    fn default() -> Self {
        Self::parse_from::<[_; 0], &str>([])
    }
}

impl MetricsArgs {
    /// Returns whether metrics are enabled.
    pub const fn is_enabled(&self) -> bool {
        matches!(self.enabled, Some(true))
    }

    /// Initialize the tracing stack and Prometheus metrics recorder.
    ///
    /// This function should be called at the beginning of the program.
    pub fn init_metrics(&self) -> CliResult<()> {
        if self.is_enabled() {
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

    /// Helper struct to parse `MetricsArgs` within a test CLI structure.
    #[derive(Parser, Debug)]
    struct TestCli {
        #[command(flatten)]
        metrics: MetricsArgs,
    }

    #[test]
    fn test_default_metrics_args() {
        let cli = TestCli::parse_from(["test_app"]);
        assert!(!cli.metrics.is_enabled(), "Default for metrics.enabled should be false.");
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
        assert!(cli.metrics.is_enabled(), "metrics.enabled should be true.");
        assert_eq!(cli.metrics.port, 9999, "metrics.port should be parsed from CLI.");
        assert_eq!(
            cli.metrics.addr,
            IpAddr::V4(Ipv4Addr::new(127, 0, 0, 1)),
            "metrics.addr should be parsed from CLI."
        );
    }

    #[test]
    fn metrics_flags_require_metrics_enabled() {
        for args in [["--metrics.port", "9999"], ["--metrics.addr", "127.0.0.1"]] {
            let error =
                TestCli::try_parse_from(std::iter::once("test_app").chain(args)).unwrap_err();
            assert_eq!(error.kind(), clap::error::ErrorKind::MissingRequiredArgument, "{args:?}");
        }
    }

    #[test]
    fn metrics_enabled_takes_no_value() {
        let error = TestCli::try_parse_from(["test_app", "--metrics.enabled=false"]).unwrap_err();
        assert_eq!(error.kind(), clap::error::ErrorKind::TooManyValues);
    }

    #[test]
    fn metrics_enabled_environment_variable_must_be_true_or_unset() {
        // SAFETY: nextest runs each test in its own process.
        unsafe { std::env::set_var("KONA_METRICS_ENABLED", "true") };
        let enabled = TestCli::try_parse_from(["test_app", "--metrics.port", "9999"]);
        // SAFETY: as above.
        unsafe { std::env::set_var("KONA_METRICS_ENABLED", "false") };
        let disabled = TestCli::try_parse_from(["test_app"]);
        // SAFETY: as above.
        unsafe { std::env::remove_var("KONA_METRICS_ENABLED") };

        let enabled = enabled.unwrap().metrics;
        assert!(enabled.is_enabled());
        assert_eq!(enabled.port, 9999);
        assert_eq!(disabled.unwrap_err().kind(), clap::error::ErrorKind::ValueValidation);
    }
}
