use metrics::gauge;

/// The latest version from Cargo.toml.
pub const CARGO_PKG_VERSION: &str = env!("CARGO_PKG_VERSION");

/// The 8 character short SHA of the latest commit.
pub const VERGEN_GIT_SHA: &str = env!("VERGEN_GIT_SHA_SHORT");

/// The build timestamp.
pub const VERGEN_BUILD_TIMESTAMP: &str = env!("VERGEN_BUILD_TIMESTAMP");

pub const VERSION: VersionInfo = VersionInfo {
    version: CARGO_PKG_VERSION,
    build_timestamp: VERGEN_BUILD_TIMESTAMP,
    git_sha: VERGEN_GIT_SHA,
};

/// Contains version information for the application.
#[derive(Debug, Clone)]
pub struct VersionInfo {
    /// The version of the application.
    pub version: &'static str,
    /// The build timestamp of the application.
    pub build_timestamp: &'static str,
    /// The Git SHA of the build.
    pub git_sha: &'static str,
}

impl VersionInfo {
    /// This exposes rollup-boost's version information over prometheus.
    pub fn register_version_metrics(&self) {
        let labels: [(&str, &str); 3] = [
            ("version", self.version),
            ("build_timestamp", self.build_timestamp),
            ("git_sha", self.git_sha),
        ];

        let gauge = gauge!("builder_info", &labels);
        gauge.set(1);
    }
}

pub const fn get_version() -> &'static str {
    env!("CARGO_PKG_VERSION")
}
