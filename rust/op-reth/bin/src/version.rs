//! Version metadata for the `op-reth` binary.
//!
//! `reth-node-core` exposes a global [`RethCliVersionConsts`] singleton (read by the CLI
//! `--version` output, `web3_clientVersion`, `engine_getClientVersionV1`, and the p2p `Hello`
//! handshake identity) that otherwise defaults to the pinned upstream `reth` crate's own
//! version and commit, since `reth-node-core` computes it from its own compile-time env vars.
//! [`try_init_version_metadata`] must be called once, before that singleton is first read, to
//! override it with `op-reth`'s own version and commit info.
//!
//! [`try_init_version_metadata`]: reth_node_core::version::try_init_version_metadata

use reth_node_core::version::RethCliVersionConsts;
use reth_optimism_node::OP_NAME_CLIENT;
use std::borrow::Cow;

/// Builds the `op-reth`-specific version metadata, sourced from env vars emitted by `build.rs`.
pub(crate) fn op_reth_version_metadata() -> RethCliVersionConsts {
    RethCliVersionConsts {
        name_client: Cow::Borrowed(OP_NAME_CLIENT),
        cargo_pkg_version: Cow::Borrowed(env!("CARGO_PKG_VERSION")),
        vergen_git_sha_long: Cow::Borrowed(env!("VERGEN_GIT_SHA")),
        vergen_git_sha: Cow::Borrowed(env!("VERGEN_GIT_SHA_SHORT")),
        vergen_build_timestamp: Cow::Borrowed(env!("VERGEN_BUILD_TIMESTAMP")),
        vergen_cargo_target_triple: Cow::Borrowed(env!("VERGEN_CARGO_TARGET_TRIPLE")),
        vergen_cargo_features: Cow::Borrowed(env!("VERGEN_CARGO_FEATURES")),
        short_version: Cow::Borrowed(env!("OP_RETH_SHORT_VERSION")),
        long_version: Cow::Owned(format!(
            "{}\n{}\n{}\n{}\n{}",
            env!("OP_RETH_LONG_VERSION_0"),
            env!("OP_RETH_LONG_VERSION_1"),
            env!("OP_RETH_LONG_VERSION_2"),
            env!("OP_RETH_LONG_VERSION_3"),
            env!("OP_RETH_LONG_VERSION_4"),
        )),
        build_profile_name: Cow::Borrowed(env!("OP_RETH_BUILD_PROFILE")),
        p2p_client_version: Cow::Borrowed(env!("OP_RETH_P2P_CLIENT_VERSION")),
        // `extra_data` isn't read anywhere in op-reth today; op-reth's payload builder computes
        // its own OP Stack extra-data bytes instead.
        ..Default::default()
    }
}
