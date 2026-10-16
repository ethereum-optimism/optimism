//! Arguments for logging.

use std::{fmt, path::PathBuf, str::FromStr};

use clap::Args;
use serde::{Deserialize, Serialize};
use tracing_subscriber::{EnvFilter, filter::ParseError};

use crate::{LogFormat, LogRotation};

/// Global configuration arguments.
#[derive(Args, Debug, Default, Serialize, Deserialize, Clone)]
#[command(next_help_heading = "Logging")]
pub struct LogArgs {
    /// Which logs to keep, for both stdout and file logging, in `RUST_LOG` syntax: a level such
    /// as `debug` for every target, a level for one target such as `engine=debug`, or several,
    /// comma-separated, such as `info,engine=debug`. A target without a level of its own logs at
    /// the bare level, and with no bare level it logs nothing. An invalid directive is an error.
    ///
    /// Unlike the other flags, which read a `KONA_` variable, this reads `RUST_LOG`.
    #[arg(long = "logs.filter", global = true, default_value = "info", env = "RUST_LOG")]
    pub filter: LogFilter,
    /// If set, no logs are printed to stdout.
    #[arg(
        long = "logs.stdout.quiet",
        global = true,
        default_value = "false",
        env = "KONA_LOGS_STDOUT_QUIET"
    )]
    pub stdout_quiet: bool,
    /// The format of the logs printed to stdout. One of: full, json, pretty, compact, logfmt.
    ///
    /// full: The default rust log format.
    /// json: The logs are printed in JSON structured format.
    /// pretty: The logs are printed in a pretty, human readable format.
    /// compact: The logs are printed in a compact format.
    /// logfmt: The logs are printed in logfmt key=value format.
    #[arg(long = "logs.stdout.format", default_value = "full", env = "KONA_LOGS_STDOUT_FORMAT")]
    pub stdout_format: LogFormat,
    /// The directory to store the log files.
    /// If not set, no logs are printed to files.
    #[arg(long = "logs.file.directory", env = "KONA_LOGS_FILE_DIRECTORY")]
    pub file_directory: Option<PathBuf>,
    /// The format of the logs printed to log files. One of: full, json, pretty, compact, logfmt.
    ///
    /// full: The default rust log format.
    /// json: The logs are printed in JSON structured format.
    /// pretty: The logs are printed in a pretty, human readable format.
    /// compact: The logs are printed in a compact format.
    /// logfmt: The logs are printed in logfmt key=value format.
    #[arg(long = "logs.file.format", default_value = "full", env = "KONA_LOGS_FILE_FORMAT")]
    pub file_format: LogFormat,
    /// The rotation of the log files. One of: hourly, daily, weekly, monthly, never.
    /// If set, new log files will be created every interval.
    #[arg(long = "logs.file.rotation", default_value = "never", env = "KONA_LOGS_FILE_ROTATION")]
    pub file_rotation: LogRotation,
}

/// A log filter in `RUST_LOG` syntax, whose directives are checked when it is parsed.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(try_from = "String", into = "String")]
pub struct LogFilter(String);

impl LogFilter {
    /// Returns this filter with the directives in `defaults` before its own, so that its own
    /// directive for a target overrides a default for the same target.
    pub fn with_defaults(&self, defaults: &str) -> Result<Self, ParseError> {
        format!("{defaults},{}", self.0).parse()
    }

    /// Returns the filter for a tracing subscriber.
    pub fn env_filter(&self) -> EnvFilter {
        // Every directive was checked when the filter was parsed, so none is dropped.
        EnvFilter::builder().parse_lossy(&self.0)
    }
}

impl Default for LogFilter {
    /// Logs at the `info` level.
    fn default() -> Self {
        Self("info".to_string())
    }
}

impl FromStr for LogFilter {
    type Err = ParseError;

    fn from_str(filter: &str) -> Result<Self, Self::Err> {
        EnvFilter::builder().parse(filter)?;
        Ok(Self(filter.to_string()))
    }
}

impl TryFrom<String> for LogFilter {
    type Error = ParseError;

    fn try_from(filter: String) -> Result<Self, Self::Error> {
        filter.parse()
    }
}

impl From<LogFilter> for String {
    fn from(filter: LogFilter) -> Self {
        filter.0
    }
}

impl fmt::Display for LogFilter {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(&self.0)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use clap::{CommandFactory, Parser};
    use tracing_subscriber::layer::SubscriberExt;

    // Helper struct to parse LogArgs within a test CLI structure
    #[derive(Parser, Debug)]
    struct TestCli {
        #[command(flatten)]
        global: LogArgs,
    }

    #[test]
    fn logs_at_info_unless_set_on_the_command_line_or_in_rust_log() {
        // Parsing would read `RUST_LOG` from the environment the tests run in, so check how the
        // flag is declared instead.
        let command = TestCli::command();
        let filter = command.get_arguments().find(|arg| arg.get_long() == Some("logs.filter"));
        let filter = filter.unwrap();
        assert_eq!(filter.get_env(), Some(std::ffi::OsStr::new("RUST_LOG")));
        assert_eq!(filter.get_default_values(), ["info"]);
        assert_eq!(LogFilter::default().to_string(), "info");
    }

    #[test]
    fn parses_levels_and_target_directives() {
        for filter in
            ["off", "error", "trace", "engine=debug", "info,engine=debug,rpc_client=trace"]
        {
            let cli = TestCli::parse_from(["test_app", "--logs.filter", filter]);
            assert_eq!(cli.global.filter.to_string(), filter);
            let json = serde_json::to_string(&cli.global.filter).unwrap();
            assert_eq!(serde_json::from_str::<LogFilter>(&json).unwrap(), cli.global.filter);
        }
    }

    #[test]
    fn a_filter_overrides_its_defaults_for_the_same_target() {
        let enabled = |filter: &str, target: &str| {
            let filter = filter.parse::<LogFilter>().unwrap().with_defaults("discv5=off").unwrap();
            let subscriber = tracing_subscriber::registry().with(filter.env_filter());
            tracing::subscriber::with_default(subscriber, || match target {
                "discv5" => tracing::enabled!(target: "discv5", tracing::Level::INFO),
                _ => tracing::enabled!(target: "engine", tracing::Level::INFO),
            })
        };
        assert!(!enabled("info", "discv5"));
        assert!(enabled("info", "engine"));
        assert!(enabled("info,discv5=info", "discv5"));
    }

    #[test]
    fn rejects_an_invalid_directive() {
        // A bare word names a target, so only a bad level or malformed directive is invalid.
        for filter in ["engine=loud", "info,engine=loud", "engine[=debug", "engine=debug=trace"] {
            assert!(
                TestCli::try_parse_from(["test_app", "--logs.filter", filter]).is_err(),
                "{filter}"
            );
            assert!(serde_json::from_str::<LogFilter>(&format!("{filter:?}")).is_err(), "{filter}");
        }
        assert!(TestCli::try_parse_from(["test_app", "--logs.level", "debug"]).is_err());
    }
}
