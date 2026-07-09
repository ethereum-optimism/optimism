//! Shared build/version metadata resolution for OP Stack binaries.

/// Version reported when no release tag was injected (non-release / local build).
const FALLBACK_VERSION: &str = "0.0.0-dev";

/// Placeholder emitted when a value was not injected at build time.
const UNKNOWN: &str = "unknown";

/// Resolve this binary's [`BuildInfo`] from the compile-time build environment.
///
/// Expands to a [`BuildInfo::resolve`] call over the standard build-time
/// environment variables (`GIT_VERSION`, `GIT_COMMIT`, `GIT_DATE`, `FEATURES`,
/// `BUILD_PROFILE`). Because a macro expands at its call site, the `option_env!`
/// reads happen in the calling binary crate — so each binary bakes its own
/// values.
///
/// ```
/// let info = op_version::build_info!();
/// println!("{}", info.short_version());
/// ```
#[macro_export]
macro_rules! build_info {
    () => {
        $crate::BuildInfo::resolve(
            option_env!("GIT_VERSION"),
            option_env!("GIT_COMMIT"),
            option_env!("GIT_DATE"),
            option_env!("FEATURES"),
            option_env!("BUILD_PROFILE"),
        )
    };
}

/// Resolved, normalized build metadata for a binary.
///
/// Construct it with the [`build_info!`] macro (the normal entry point), or with
/// [`BuildInfo::resolve`] directly.
#[derive(Debug, Clone)]
pub struct BuildInfo {
    version: String,
    commit_sha: String,
    build_timestamp: String,
    cargo_features: String,
    build_profile: String,
    target_triple: String,
}

impl BuildInfo {
    /// Resolve build metadata from the raw compile-time environment values.
    ///
    /// The [`build_info!`] macro is the normal way to call this.
    pub fn resolve(
        git_version: Option<&str>,
        git_commit: Option<&str>,
        git_date: Option<&str>,
        features: Option<&str>,
        build_profile: Option<&str>,
    ) -> Self {
        Self {
            version: resolve_version(git_version),
            commit_sha: non_placeholder(git_commit).unwrap_or(UNKNOWN).to_string(),
            build_timestamp: non_placeholder(git_date).unwrap_or(UNKNOWN).to_string(),
            cargo_features: features.unwrap_or("none").to_string(),
            build_profile: non_placeholder(build_profile).unwrap_or(UNKNOWN).to_string(),
            // Approximate the target triple from the compiled-in arch/OS.
            target_triple: format!("{}-{}", std::env::consts::ARCH, std::env::consts::OS),
        }
    }

    /// The resolved release version (no leading `v`), e.g. `1.2.3` or `0.0.0-dev`.
    pub fn version(&self) -> &str {
        &self.version
    }

    /// The full commit SHA.
    pub fn commit_sha(&self) -> &str {
        &self.commit_sha
    }

    /// The build timestamp.
    pub fn build_timestamp(&self) -> &str {
        &self.build_timestamp
    }

    /// The enabled cargo features (empty string if none were injected).
    pub fn cargo_features(&self) -> &str {
        &self.cargo_features
    }

    /// The build profile.
    pub fn build_profile(&self) -> &str {
        &self.build_profile
    }

    /// The target triple, approximated from the compiled-in arch/OS
    /// (e.g. `x86_64-linux`).
    pub fn target_triple(&self) -> &str {
        &self.target_triple
    }

    /// The first few characters of the commit SHA, or the whole SHA if it is
    /// shorter.
    pub fn short_sha(&self) -> &str {
        // `str::get` keeps this panic-safe when the SHA is the shorter
        // [`UNKNOWN`] placeholder.
        self.commit_sha.get(..8).unwrap_or(&self.commit_sha)
    }

    /// The short version string, e.g. `1.2.3 (abc12345)`.
    pub fn short_version(&self) -> String {
        format!("{} ({})", self.version, self.short_sha())
    }

    /// The long, multi-line version block.
    pub fn long_version(&self) -> String {
        format!(
            "Version: {}\n\
             Commit SHA: {}\n\
             Build Timestamp: {}\n\
             Build Features: {}\n\
             Build Profile: {}",
            self.version,
            self.commit_sha,
            self.build_timestamp,
            self.cargo_features,
            self.build_profile,
        )
    }
}

/// Resolve the reported version from the injected `GIT_VERSION`, stripping a
/// leading `v` and falling back to [`FALLBACK_VERSION`] when absent or empty.
fn resolve_version(version: Option<&str>) -> String {
    if let Some(raw) = version {
        let bare = raw.strip_prefix('v').unwrap_or(raw);
        if !bare.is_empty() {
            return bare.to_string();
        }
    }
    FALLBACK_VERSION.to_string()
}

/// Treat empty strings and build-system sentinels (`dev`, `0`) as "not injected".
fn non_placeholder(value: Option<&str>) -> Option<&str> {
    value.filter(|s| !s.is_empty() && *s != "dev" && *s != "0")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn release_tag_wins_and_v_prefix_is_stripped() {
        assert_eq!(resolve_version(Some("v1.2.3")), "1.2.3");
        assert_eq!(resolve_version(Some("1.2.3")), "1.2.3");
        assert_eq!(resolve_version(Some("v1.2.3-rc.1")), "1.2.3-rc.1");
    }

    #[test]
    fn falls_back_when_absent_or_empty() {
        for input in [Some(""), None] {
            assert_eq!(resolve_version(input), FALLBACK_VERSION, "input: {input:?}");
        }
    }

    #[test]
    fn non_placeholder_filters_sentinels() {
        assert_eq!(non_placeholder(Some("abc")), Some("abc"));
        assert_eq!(non_placeholder(Some("")), None);
        assert_eq!(non_placeholder(Some("dev")), None);
        assert_eq!(non_placeholder(Some("0")), None);
        assert_eq!(non_placeholder(None), None);
    }

    #[test]
    fn short_sha_is_panic_safe_and_width_selectable() {
        let bi = BuildInfo::resolve(
            Some("v1.2.3"),
            Some("abcdef1234567890"),
            Some("1"),
            Some("f"),
            Some("p"),
        );
        assert_eq!(bi.short_sha(), "abcdef12");
        assert_eq!(bi.short_version(), "1.2.3 (abcdef12)");
        let missing = BuildInfo::resolve(Some("v1"), None, None, None, None);
        assert_eq!(missing.short_sha(), UNKNOWN);
    }

    #[test]
    fn long_version_has_five_lines() {
        let bi =
            BuildInfo::resolve(Some("v1.2.3"), Some("abc"), Some("123"), Some(""), Some("release"));
        assert_eq!(bi.long_version().lines().count(), 5);
        assert!(bi.long_version().contains("Version: 1.2.3"));
    }
}
