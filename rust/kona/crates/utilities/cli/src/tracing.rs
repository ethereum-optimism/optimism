//! [`tracing_subscriber`] utilities.

use tracing_subscriber::{
    Layer,
    fmt::{
        format::{FormatEvent, FormatFields, Writer},
        time::{FormatTime, SystemTime},
    },
    prelude::__tracing_subscriber_SubscriberExt,
    registry::LookupSpan,
    util::{SubscriberInitExt, TryInitError},
};

use serde::{Deserialize, Serialize};
use std::fmt;

use crate::{LogConfig, LogRotation};

/// The format of the logs.
#[derive(
    Default, Debug, Clone, Copy, PartialEq, Eq, Hash, clap::ValueEnum, Serialize, Deserialize,
)]
#[serde(rename_all = "lowercase")]
#[clap(rename_all = "lowercase")]
pub enum LogFormat {
    /// Full format (default).
    #[default]
    Full,
    /// JSON format.
    Json,
    /// Pretty format.
    Pretty,
    /// Compact format.
    Compact,
    /// Logfmt format.
    Logfmt,
}

/// Custom logfmt formatter for tracing events.
struct LogfmtFormatter;

impl<S, N> FormatEvent<S, N> for LogfmtFormatter
where
    S: tracing::Subscriber + for<'a> LookupSpan<'a>,
    N: for<'a> FormatFields<'a> + 'static,
{
    fn format_event(
        &self,
        ctx: &tracing_subscriber::fmt::FmtContext<'_, S, N>,
        mut writer: Writer<'_>,
        event: &tracing::Event<'_>,
    ) -> fmt::Result {
        let meta = event.metadata();

        // Write timestamp
        let time_format = SystemTime;
        write!(writer, "time=\"")?;
        time_format.format_time(&mut writer)?;
        write!(writer, "\" ")?;

        // Write level
        write!(writer, "level={} ", meta.level())?;

        // Write target
        write!(writer, "target={} ", meta.target())?;

        // Write the message and fields
        ctx.field_format().format_fields(writer.by_ref(), event)?;

        writeln!(writer)
    }
}

impl LogConfig {
    /// Initializes the tracing subscriber, keeping the logs that [`Self::filter`] keeps.
    pub fn init_tracing_subscriber(&self) -> Result<(), TryInitError> {
        let file_layer = self.file_logs.as_ref().map(|file_logs| {
            let directory_path = file_logs.directory_path.clone();

            let appender = match file_logs.rotation {
                LogRotation::Minutely => {
                    tracing_appender::rolling::minutely(directory_path, "kona.log")
                }
                LogRotation::Hourly => {
                    tracing_appender::rolling::hourly(directory_path, "kona.log")
                }
                LogRotation::Daily => tracing_appender::rolling::daily(directory_path, "kona.log"),
                LogRotation::Never => tracing_appender::rolling::never(directory_path, "kona.log"),
            };

            match file_logs.format {
                LogFormat::Full => tracing_subscriber::fmt::layer().with_writer(appender).boxed(),
                LogFormat::Json => {
                    tracing_subscriber::fmt::layer().json().with_writer(appender).boxed()
                }
                LogFormat::Pretty => {
                    tracing_subscriber::fmt::layer().pretty().with_writer(appender).boxed()
                }
                LogFormat::Compact => {
                    tracing_subscriber::fmt::layer().compact().with_writer(appender).boxed()
                }
                LogFormat::Logfmt => tracing_subscriber::fmt::layer()
                    .event_format(LogfmtFormatter)
                    .with_writer(appender)
                    .boxed(),
            }
        });

        let stdout_layer = self.stdout_logs.as_ref().map(|stdout_logs| match stdout_logs.format {
            LogFormat::Full => tracing_subscriber::fmt::layer().boxed(),
            LogFormat::Json => tracing_subscriber::fmt::layer().json().boxed(),
            LogFormat::Pretty => tracing_subscriber::fmt::layer().pretty().boxed(),
            LogFormat::Compact => tracing_subscriber::fmt::layer().compact().boxed(),
            LogFormat::Logfmt => {
                tracing_subscriber::fmt::layer().event_format(LogfmtFormatter).boxed()
            }
        });

        tracing_subscriber::registry()
            .with(self.filter.env_filter())
            .with(file_layer)
            .with(stdout_layer)
            .try_init()?;

        Ok(())
    }
}

/// Initializes tracing for a test, filtered by `RUST_LOG` if it holds a valid filter and at the
/// `info` level otherwise. It does nothing if tracing is already initialized.
pub fn init_test_tracing() {
    let filter = std::env::var("RUST_LOG").ok().and_then(|filter| filter.parse().ok());
    let _ = LogConfig { filter: filter.unwrap_or_default(), ..Default::default() }
        .init_tracing_subscriber();
}
