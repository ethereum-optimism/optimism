//! Logging Configuration Types

use std::path::PathBuf;

use crate::{LogArgs, LogFilter, LogFormat};
use clap::ValueEnum;
use serde::{Deserialize, Serialize};

/// The rotation of the log files.
#[derive(Debug, Clone, Serialize, Deserialize, ValueEnum, Default)]
#[serde(rename_all = "lowercase")]
pub enum LogRotation {
    /// Rotate the log files every minute.
    Minutely,
    /// Rotate the log files hourly.
    Hourly,
    /// Rotate the log files daily.
    Daily,
    /// Do not rotate the log files.
    #[default]
    Never,
}

/// Configuration for file logging.
#[derive(Debug, Clone)]
pub struct FileLogConfig {
    /// The path to the directory where the log files are stored.
    pub directory_path: PathBuf,
    /// The format of the logs printed to the log file.
    pub format: LogFormat,
    /// The rotation of the log files.
    pub rotation: LogRotation,
}

/// Configuration for stdout logging.
#[derive(Debug, Clone)]
pub struct StdoutLogConfig {
    /// The format of the logs printed to stdout.
    pub format: LogFormat,
}

/// Global configuration for logging.
/// Default is to only print logs to stdout in full format.
#[derive(Debug, Clone)]
pub struct LogConfig {
    /// Which logs to keep, for both stdout and file logging.
    pub filter: LogFilter,
    /// The configuration for stdout logging.
    pub stdout_logs: Option<StdoutLogConfig>,
    /// The configuration for file logging.
    pub file_logs: Option<FileLogConfig>,
}

impl Default for LogConfig {
    fn default() -> Self {
        Self {
            filter: LogFilter::default(),
            stdout_logs: Some(StdoutLogConfig { format: LogFormat::Full }),
            file_logs: None,
        }
    }
}

impl From<LogArgs> for LogConfig {
    fn from(args: LogArgs) -> Self {
        Self::new(args)
    }
}

impl LogConfig {
    /// Creates a new `LogConfig` from `LogArgs`.
    pub fn new(args: LogArgs) -> Self {
        let stdout_logs = if args.stdout_quiet {
            None
        } else {
            Some(StdoutLogConfig { format: args.stdout_format })
        };

        let file_logs = args.file_directory.as_ref().map(|path| FileLogConfig {
            directory_path: path.clone(),
            format: args.file_format,
            rotation: args.file_rotation,
        });

        Self { filter: args.filter, stdout_logs, file_logs }
    }
}
