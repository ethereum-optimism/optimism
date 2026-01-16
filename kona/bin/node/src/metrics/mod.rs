//! Global metrics for `kona-node`

mod cli_opts;
pub use cli_opts::{CliMetrics, init_rollup_config_metrics};

mod version;
pub use version::VersionInfo;
