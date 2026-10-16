//! Logging CLI Flags

use clap::{ArgAction, Args};
use kona_cli::{LogConfig, LogFilter, LogFormat, StdoutLogConfig};

/// Logging CLI Flags
///
/// The node logs to stdout only: redirect stdout to keep its logs in a file.
#[derive(Args, Debug, Clone, PartialEq, Eq)]
#[command(next_help_heading = "Logging")]
pub struct LogArgs {
    /// Print logs to stdout. Disabled by default.
    #[arg(
        id = "logs_enabled",
        long = "logs.enabled",
        env = "KONA_NODE_LOGS_ENABLED",
        action = ArgAction::Set,
        num_args = 0..=1,
        require_equals = true,
        default_value_t = false,
        default_missing_value = "true",
        value_name = "BOOL"
    )]
    pub enabled: bool,
    /// Which logs to keep, in `RUST_LOG` syntax: a level such as `debug` for every target, a
    /// level for one target such as `engine=debug`, or several, comma-separated, such as
    /// `info,engine=debug`. A target without a level of its own logs at the bare level, and with
    /// no bare level it logs nothing. An invalid directive is an error.
    ///
    /// Unlike the other flags, which read a `KONA_NODE_` variable, this reads `RUST_LOG`.
    #[arg(long = "logs.filter", default_value = "info", env = "RUST_LOG")]
    pub filter: LogFilter,
    /// The format of the logs.
    #[arg(long = "logs.format", default_value = "full", env = "KONA_NODE_LOGS_FORMAT")]
    pub format: LogFormat,
}

impl LogArgs {
    /// Returns the logging configuration: stdout logs kept by the given filter, in the given
    /// format, if enabled, and no logs otherwise.
    pub fn config(&self) -> LogConfig {
        LogConfig {
            filter: self.filter.clone(),
            stdout_logs: self.enabled.then_some(StdoutLogConfig { format: self.format }),
            file_logs: None,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use clap::Parser;

    #[derive(Parser, Debug)]
    struct TestCli {
        #[command(flatten)]
        logs: LogArgs,
    }

    fn parse(args: &[&str]) -> Result<LogArgs, clap::Error> {
        TestCli::try_parse_from(std::iter::once("test").chain(args.iter().copied()))
            .map(|cli| cli.logs)
    }

    #[test]
    fn logs_nothing_by_default() {
        let logs = parse(&[]).unwrap();
        assert!(!logs.enabled);
        assert!(logs.config().stdout_logs.is_none());
        assert!(logs.config().file_logs.is_none());
    }

    #[test]
    fn logs_to_stdout_with_the_filter_and_format_given() {
        let logs = parse(&[
            "--logs.enabled",
            "--logs.filter",
            "info,engine=debug",
            "--logs.format",
            "json",
        ])
        .unwrap()
        .config();
        assert_eq!(logs.filter.to_string(), "info,engine=debug");
        assert_eq!(logs.stdout_logs.unwrap().format, LogFormat::Json);
        assert!(logs.file_logs.is_none());
    }

    #[test]
    fn the_level_flag_is_gone() {
        assert!(parse(&["--logs.level", "debug"]).is_err());
    }

    #[test]
    fn the_switch_takes_true_or_false() {
        assert!(parse(&["--logs.enabled=true"]).unwrap().enabled);
        assert!(!parse(&["--logs.enabled=false"]).unwrap().enabled);
        assert!(parse(&["--logs.enabled=yes"]).is_err());
    }

    #[test]
    fn the_environment_variable_takes_true_or_false() {
        // SAFETY: nextest runs each test in its own process.
        unsafe { std::env::set_var("KONA_NODE_LOGS_ENABLED", "true") };
        let enabled = parse(&[]);
        // SAFETY: as above.
        unsafe { std::env::set_var("KONA_NODE_LOGS_ENABLED", "false") };
        let disabled = parse(&[]);
        // SAFETY: as above.
        unsafe { std::env::remove_var("KONA_NODE_LOGS_ENABLED") };

        assert!(enabled.unwrap().enabled);
        assert!(!disabled.unwrap().enabled);
    }
}
