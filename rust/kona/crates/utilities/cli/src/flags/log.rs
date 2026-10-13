//! Arguments for logging.

use std::path::PathBuf;

use clap::{Args, ValueEnum};
use serde::{Deserialize, Serialize};
use tracing::level_filters::LevelFilter;

use crate::{LogFormat, LogRotation};

/// Global configuration arguments.
#[derive(Args, Debug, Default, Serialize, Deserialize, Clone)]
#[command(next_help_heading = "Logging")]
pub struct LogArgs {
    /// The most detailed level to log, for both stdout and file logging.
    #[arg(long = "logs.level", global = true, default_value = "info", env = "KONA_LOG_LEVEL")]
    pub level: LogLevel,
    /// If set, no logs are printed to stdout.
    #[arg(
        long = "logs.stdout.quiet",
        global = true,
        default_value = "false",
        env = "KONA_STDOUT_LOG_QUIET"
    )]
    pub stdout_quiet: bool,
    /// The format of the logs printed to stdout. One of: full, json, pretty, compact, logfmt.
    ///
    /// full: The default rust log format.
    /// json: The logs are printed in JSON structured format.
    /// pretty: The logs are printed in a pretty, human readable format.
    /// compact: The logs are printed in a compact format.
    /// logfmt: The logs are printed in logfmt key=value format.
    #[arg(long = "logs.stdout.format", default_value = "full", env = "KONA_LOG_STDOUT_FORMAT")]
    pub stdout_format: LogFormat,
    /// The directory to store the log files.
    /// If not set, no logs are printed to files.
    #[arg(long = "logs.file.directory", env = "KONA_LOG_FILE_DIRECTORY")]
    pub file_directory: Option<PathBuf>,
    /// The format of the logs printed to log files. One of: full, json, pretty, compact, logfmt.
    ///
    /// full: The default rust log format.
    /// json: The logs are printed in JSON structured format.
    /// pretty: The logs are printed in a pretty, human readable format.
    /// compact: The logs are printed in a compact format.
    /// logfmt: The logs are printed in logfmt key=value format.
    #[arg(long = "logs.file.format", default_value = "full", env = "KONA_LOG_FILE_FORMAT")]
    pub file_format: LogFormat,
    /// The rotation of the log files. One of: hourly, daily, weekly, monthly, never.
    /// If set, new log files will be created every interval.
    #[arg(long = "logs.file.rotation", default_value = "never", env = "KONA_LOG_FILE_ROTATION")]
    pub file_rotation: LogRotation,
}

/// A level to log at, from the least detailed to the most.
#[derive(Default, Debug, Clone, Copy, PartialEq, Eq, Hash, ValueEnum, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
#[clap(rename_all = "lowercase")]
pub enum LogLevel {
    /// Errors only.
    Error,
    /// Warnings and errors.
    Warn,
    /// Informational messages, warnings and errors.
    #[default]
    Info,
    /// Debugging messages and everything above.
    Debug,
    /// Every message.
    Trace,
}

impl From<LogLevel> for LevelFilter {
    fn from(level: LogLevel) -> Self {
        match level {
            LogLevel::Error => Self::ERROR,
            LogLevel::Warn => Self::WARN,
            LogLevel::Info => Self::INFO,
            LogLevel::Debug => Self::DEBUG,
            LogLevel::Trace => Self::TRACE,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use clap::Parser;

    // Helper struct to parse LogArgs within a test CLI structure
    #[derive(Parser, Debug)]
    struct TestCli {
        #[command(flatten)]
        global: LogArgs,
    }

    #[test]
    fn logs_at_info_by_default() {
        let cli = TestCli::parse_from(["test_app"]);
        assert_eq!(cli.global.level, LogLevel::Info);
    }

    #[test]
    fn parses_each_level_by_name() {
        for (name, level) in [
            ("error", LogLevel::Error),
            ("warn", LogLevel::Warn),
            ("info", LogLevel::Info),
            ("debug", LogLevel::Debug),
            ("trace", LogLevel::Trace),
        ] {
            let cli = TestCli::parse_from(["test_app", "--logs.level", name]);
            assert_eq!(cli.global.level, level, "{name}");
        }
        assert!(TestCli::try_parse_from(["test_app", "--logs.level", "3"]).is_err());
        assert!(TestCli::try_parse_from(["test_app", "-v"]).is_err());
    }
}
