//! Build script for the `op-reth` binary crate.
//!
//! Adapted from [reth](https://github.com/paradigmxyz/reth/blob/main/crates/node/core/build.rs)
//! and [op-rbuilder](https://github.com/flashbots/op-rbuilder/blob/main/crates/op-rbuilder/build.rs).
//!
//! This script performs three jobs at compile time:
//!
//! ## 1. Version override (`OP_RETH_VERSION`)
//!
//! If the `OP_RETH_VERSION` environment variable is set (e.g. injected as a
//! Docker `--build-arg` during a release build), its value is used as
//! `CARGO_PKG_VERSION` for the entire build unit.  This ensures that the
//! version string reported by `op-reth --version` matches the published git
//! tag (e.g. `v1.14.0`) rather than whatever semver is written in
//! `Cargo.toml`.
//!
//! ## 2. Build-time metadata via `vergen`
//!
//! [`vergen`] is used to emit a set of `cargo:rustc-env` variables that are
//! available to the binary at runtime through `env!(...)`:
//!
//! | Variable | Content |
//! |---|---|
//! | `VERGEN_BUILD_TIMESTAMP` | RFC 3339 timestamp of the build |
//! | `VERGEN_CARGO_FEATURES` | Comma-separated list of active Cargo features |
//! | `VERGEN_CARGO_TARGET_TRIPLE` | Target triple (e.g. `x86_64-unknown-linux-gnu`) |
//! | `VERGEN_GIT_SHA` | Full 40-character commit SHA |
//! | `VERGEN_GIT_SHA_SHORT` | First 8 characters of the commit SHA |
//! | `VERGEN_GIT_DIRTY` | `true` when the working tree has uncommitted changes |
//! | `VERGEN_GIT_DESCRIBE` | Output of `git describe --always --tags` |
//!
//! ## 3. Short version string (`RETH_SHORT_VERSION`)
//!
//! A human-readable short version string is assembled and exposed as
//! `RETH_SHORT_VERSION`, for example:
//!
//! ```text
//! v1.14.0 (1939939)
//! v1.14.0-dev (1939939)
//! ```
//!
//! A `-dev` suffix is appended when the working tree is dirty **or** when the
//! current commit is not directly on a tag, unless `OP_RETH_VERSION` was
//! explicitly provided (release builds are always treated as "on tag").

use std::{env, error::Error};
use vergen::{BuildBuilder, CargoBuilder, Emitter};
use vergen_git2::Git2Builder;

fn main() -> Result<(), Box<dyn Error>> {
    // If OP_RETH_VERSION is set at Docker build time (injected via --build-arg),
    // use it as the canonical version so the reported version matches the git tag
    // (e.g. "op-reth/v1.14.0" → "v1.14.0"). Otherwise fall back to the semver
    // string in Cargo.toml.
    println!("cargo:rerun-if-env-changed=OP_RETH_VERSION");
    let pkg_version = env::var("OP_RETH_VERSION")
        .ok()
        .filter(|v| !v.is_empty())
        .unwrap_or_else(|| env!("CARGO_PKG_VERSION").to_owned());

    // Override CARGO_PKG_VERSION for all crates that call env!("CARGO_PKG_VERSION")
    // within this build unit so that reth's version machinery picks up the tag.
    println!("cargo:rustc-env=CARGO_PKG_VERSION={pkg_version}");

    let mut emitter = Emitter::default();

    let build_builder = BuildBuilder::default().build_timestamp(true).build()?;
    emitter.add_instructions(&build_builder)?;

    let cargo_builder = CargoBuilder::default().features(true).target_triple(true).build()?;
    emitter.add_instructions(&cargo_builder)?;

    let git_builder =
        Git2Builder::default().describe(false, true, None).dirty(true).sha(false).build()?;
    emitter.add_instructions(&git_builder)?;

    emitter.emit_and_set()?;

    let sha = env::var("VERGEN_GIT_SHA")?;
    let sha_short = &sha[0..7];

    // Set short SHA
    println!("cargo:rustc-env=VERGEN_GIT_SHA_SHORT={}", &sha[..8]);

    let is_dirty = env::var("VERGEN_GIT_DIRTY")? == "true";
    // > git describe --always --tags
    // if not on a tag: v0.2.0-beta.3-82-g1939939b
    // if on a tag:     v0.2.0-beta.3
    let not_on_tag = env::var("VERGEN_GIT_DESCRIBE")?.ends_with(&format!("-g{sha_short}"));

    // When OP_RETH_VERSION is explicitly provided (i.e. a release build), treat the
    // binary as "on tag" regardless of what git describe says about the monorepo.
    let version_suffix = if env::var("OP_RETH_VERSION").ok().filter(|v| !v.is_empty()).is_some() {
        ""
    } else if is_dirty || not_on_tag {
        "-dev"
    } else {
        ""
    };

    println!("cargo:rustc-env=RETH_SHORT_VERSION={pkg_version}{version_suffix} ({sha_short})");

    Ok(())
}
