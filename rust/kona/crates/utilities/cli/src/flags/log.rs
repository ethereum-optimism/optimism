//! Arguments for logging.

use std::path::PathBuf;

use clap::{ArgAction, Args};
use serde::{Deserialize, Serialize};

use crate::{LogFormat, LogRotation};

/// Global configuration arguments.
#[derive(Args, Debug, Default, Serialize, Deserialize, Clone)]
pub struct LogArgs {
    /// Verbosity level (1-5).
    /// By default, the verbosity level is set to 3 (info level).
    ///
    /// This verbosity level is shared by both stdout and file logging (if enabled).
    #[arg(
        short = 'v',
        global = true,
        default_value = "3",
        env = "KONA_LOG_LEVEL",
        action = ArgAction::Count,
    )]
    pub level: u8,
    /// If set, no logs are printed to stdout.
    #[arg(
        long = "logs.stdout.quiet",
        short = 'q',
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

#[cfg(test)]
mod tests {
    use super::*;
    use clap::Parser;

    // Helper struct to parse GlobalArgs within a test CLI structure
    #[derive(Parser, Debug)]
    struct TestCli {
        #[command(flatten)]
        global: LogArgs,
    }

    #[test]
    fn test_default_verbosity_level() {
        let cli = TestCli::parse_from(["test_app"]);
        assert_eq!(
            cli.global.level, 3,
            "Default verbosity should be 3 when no -v flag is present."
        );
    }

    #[test]
    fn test_verbosity_count() {
        let cli_v1 = TestCli::parse_from(["test_app", "-v"]);
        assert_eq!(cli_v1.global.level, 1, "Verbosity with a single -v should be 1.");

        let cli_v3 = TestCli::parse_from(["test_app", "-vvv"]);
        assert_eq!(cli_v3.global.level, 3, "Verbosity with -vvv should be 3.");

        let cli_v5 = TestCli::parse_from(["test_app", "-vvvvv"]);
        assert_eq!(cli_v5.global.level, 5, "Verbosity with -vvvvv should be 5.");
    }
}
