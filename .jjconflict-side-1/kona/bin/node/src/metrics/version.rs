//! [`VersionInfo`] metrics
//!
//! Derived from [`reth-node-core`'s type][reth-version-info]
//!
//! [reth-version-info]: https://github.com/paradigmxyz/reth/blob/805fb1012cd1601c3b4fe9e8ca2d97c96f61355b/crates/node/metrics/src/version.rs#L6

use metrics::gauge;

/// Contains version information for the application and allows for exposing the contained
/// information as a prometheus metric.
#[derive(Debug, Clone)]
pub struct VersionInfo {
    /// The version of the application.
    pub version: &'static str,
    /// The build timestamp of the application.
    pub build_timestamp: &'static str,
    /// The cargo features enabled for the build.
    pub cargo_features: &'static str,
    /// The Git SHA of the build.
    pub git_sha: &'static str,
    /// The target triple for the build.
    pub target_triple: &'static str,
    /// The build profile (e.g., debug or release).
    pub build_profile: &'static str,
}

impl VersionInfo {
    /// Creates a new instance of [`VersionInfo`] from the constants defined in [`crate::version`]
    /// at compile time.
    pub const fn from_build() -> Self {
        Self {
            version: crate::version::CARGO_PKG_VERSION,
            build_timestamp: crate::version::VERGEN_BUILD_TIMESTAMP,
            cargo_features: crate::version::VERGEN_CARGO_FEATURES,
            git_sha: crate::version::VERGEN_GIT_SHA,
            target_triple: crate::version::VERGEN_CARGO_TARGET_TRIPLE,
            build_profile: crate::version::BUILD_PROFILE_NAME,
        }
    }

    /// Exposes kona-node's version information over prometheus.
    pub fn register_version_metrics(&self) {
        // If no features are enabled, the string will be empty, and the metric will not be
        // reported. Report "none" if the string is empty.
        let features = if self.cargo_features.is_empty() { "none" } else { self.cargo_features };

        let labels: [(&str, &str); 6] = [
            ("version", self.version),
            ("build_timestamp", self.build_timestamp),
            ("cargo_features", features),
            ("git_sha", self.git_sha),
            ("target_triple", self.target_triple),
            ("build_profile", self.build_profile),
        ];

        let gauge = gauge!("kona_node_info", &labels);
        gauge.set(1);
    }
}
