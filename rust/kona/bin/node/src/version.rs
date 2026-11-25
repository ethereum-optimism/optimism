//! Version information for kona-node.

op_version::version_accessors!(pub(crate));

/// The resolved release version (no leading `v`), e.g. `1.2.3` or `0.0.0-dev`.
pub(crate) fn version() -> &'static str {
    BUILD_INFO.version()
}

/// The short commit SHA.
pub(crate) fn git_sha() -> &'static str {
    BUILD_INFO.short_sha()
}

/// The build timestamp.
pub(crate) fn build_timestamp() -> &'static str {
    BUILD_INFO.build_timestamp()
}

/// The enabled cargo features (empty string if none were injected).
pub(crate) fn cargo_features() -> &'static str {
    BUILD_INFO.cargo_features()
}

/// The target triple, approximated from the compiled-in arch/OS.
pub(crate) fn target_triple() -> &'static str {
    BUILD_INFO.target_triple()
}

/// The build profile name.
pub(crate) fn build_profile() -> &'static str {
    BUILD_INFO.build_profile()
}
