#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/favicon.ico"
)]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]

mod error;
pub use error::{CliError, CliResult};

mod flags;
pub use flags::{GlobalArgs, LogArgs, MetricsArgs, OverrideArgs};

mod logs;
pub use logs::{FileLogConfig, LogConfig, LogRotation, StdoutLogConfig};

mod clap;
pub use clap::cli_styles;

#[cfg(feature = "secrets")]
mod secrets;
#[cfg(feature = "secrets")]
pub use secrets::{KeypairError, ParseKeyError, SecretKeyLoader};

pub mod backtrace;

mod tracing;
pub use tracing::{LogFormat, init_test_tracing};

mod prometheus;
pub use prometheus::init_prometheus_server;

pub mod sigsegv_handler;
