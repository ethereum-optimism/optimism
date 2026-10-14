//! Logging CLI Flags

use clap::{ArgAction, Args};
use kona_cli::{LogConfig, LogFormat, LogLevel, StdoutLogConfig};

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
    /// The most detailed level to log.
    #[arg(long = "logs.level", default_value = "info", env = "KONA_NODE_LOGS_LEVEL")]
    pub level: LogLevel,
    /// The format of the logs.
    #[arg(long = "logs.format", default_value = "full", env = "KONA_NODE_LOGS_FORMAT")]
    pub format: LogFormat,
}

impl LogArgs {
    /// Returns the logging configuration: stdout logs at the given level and format if enabled,
    /// and no logs otherwise.
    pub fn config(&self) -> LogConfig {
        LogConfig {
            global_level: self.level.into(),
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
    fn logs_to_stdout_at_the_level_and_format_given() {
        let logs = parse(&["--logs.enabled", "--logs.level", "debug", "--logs.format", "json"])
            .unwrap()
            .config();
        assert_eq!(logs.global_level, tracing::level_filters::LevelFilter::DEBUG);
        assert_eq!(logs.stdout_logs.unwrap().format, LogFormat::Json);
        assert!(logs.file_logs.is_none());
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
