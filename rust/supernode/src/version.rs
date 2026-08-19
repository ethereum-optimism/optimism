//! Version information for `rust-supernode`.

use op_version::BuildInfo;
use std::sync::LazyLock;

/// Build metadata resolved from the compile-time environment.
static BUILD_INFO: LazyLock<BuildInfo> = LazyLock::new(|| op_version::build_info!());

/// The single-line version string.
static SHORT_VERSION: LazyLock<String> = LazyLock::new(|| BUILD_INFO.short_version());

/// The multi-line build metadata block.
static LONG_VERSION: LazyLock<String> = LazyLock::new(|| BUILD_INFO.long_version());

/// The single-line version string, e.g. `0.1.0 (abc12345)`.
pub(crate) fn short_version() -> &'static str {
    &SHORT_VERSION
}

/// The multi-line build metadata block.
pub(crate) fn long_version() -> &'static str {
    &LONG_VERSION
}
