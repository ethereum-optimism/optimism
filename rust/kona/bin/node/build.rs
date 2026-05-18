//! Derived from [`reth-node-core`][reth-build-script]
//!
//! [reth-build-script]: https://github.com/paradigmxyz/reth/blob/805fb1012cd1601c3b4fe9e8ca2d97c96f61355b/crates/node/core/build.rs

#![allow(missing_docs)]

use std::{env, error::Error};
use vergen::{BuildBuilder, CargoBuilder, Emitter};
use vergen_git2::Git2Builder;

fn main() -> Result<(), Box<dyn Error>> {
    // If KONA_NODE_VERSION is set at Docker build time (injected via --build-arg),
    // use it as the canonical version so the reported version matches the git tag
    // (e.g. "kona-node/v1.2.3" -> "v1.2.3"). Otherwise fall back to the semver
    // string in Cargo.toml.
    println!("cargo:rerun-if-env-changed=KONA_NODE_VERSION");
    let pkg_version = env::var("KONA_NODE_VERSION")
        .ok()
        .filter(|v| !v.is_empty())
        .unwrap_or_else(|| env!("CARGO_PKG_VERSION").to_owned());

    // Override CARGO_PKG_VERSION so any downstream crate calling env!("CARGO_PKG_VERSION")
    // in this build unit also picks up the injected tag.
    println!("cargo:rustc-env=CARGO_PKG_VERSION={pkg_version}");

    let mut emitter = Emitter::default();

    let build_builder = BuildBuilder::default().build_timestamp(true).build()?;

    // Add build timestamp information.
    emitter.add_instructions(&build_builder)?;

    let cargo_builder = CargoBuilder::default().features(true).target_triple(true).build()?;

    // Add cargo features and target information.
    emitter.add_instructions(&cargo_builder)?;

    let git_builder =
        Git2Builder::default().describe(false, true, None).dirty(true).sha(false).build()?;

    // Add commit information.
    emitter.add_instructions(&git_builder)?;

    emitter.emit_and_set()?;
    let sha = env::var("VERGEN_GIT_SHA")?;
    let sha_short = &sha[0..8];

    let is_dirty = env::var("VERGEN_GIT_DIRTY")? == "true";
    // > git describe --always --tags
    // if not on a tag: v0.2.0-beta.3-82-g1939939b
    // if on a tag: v0.2.0-beta.3
    let not_on_tag = env::var("VERGEN_GIT_DESCRIBE")?.ends_with(&format!("-g{sha_short}"));
    // When KONA_NODE_VERSION is explicitly provided (i.e. a release build), treat the
    // binary as "on tag" regardless of what git describe says about the monorepo.
    let version_suffix = if env::var("KONA_NODE_VERSION").ok().filter(|v| !v.is_empty()).is_some() {
        ""
    } else if is_dirty || not_on_tag {
        "-dev"
    } else {
        ""
    };
    println!("cargo:rustc-env=KONA_NODE_VERSION_SUFFIX={version_suffix}");

    // Set short SHA
    println!("cargo:rustc-env=VERGEN_GIT_SHA_SHORT={sha_short}");

    // Set the build profile
    let out_dir = env::var("OUT_DIR").unwrap();
    let profile = out_dir.rsplit(std::path::MAIN_SEPARATOR).nth(3).unwrap();
    println!("cargo:rustc-env=KONA_NODE_BUILD_PROFILE={profile}");

    // Set formatted version strings — use the version resolved at the top of main().

    // The short version information for kona-node.
    // - The latest version from Cargo.toml
    // - The short SHA of the latest commit.
    // Example: 0.1.0 (defa64b2)
    println!("cargo:rustc-env=KONA_NODE_SHORT_VERSION={pkg_version}{version_suffix} ({sha_short})");

    let features = env::var("VERGEN_CARGO_FEATURES")?;

    // LONG_VERSION
    // The long version information for kona-node.
    //
    // - The latest version from Cargo.toml + version suffix (if any)
    // - The full SHA of the latest commit
    // - The build datetime
    // - The build features
    // - The build profile
    //
    // Example:
    //
    // ```text
    // Version: 0.1.0
    // Commit SHA: defa64b2
    // Build Timestamp: 2023-05-19T01:47:19.815651705Z
    // Build Features: jemalloc
    // Build Profile: maxperf
    // ```
    println!("cargo:rustc-env=KONA_NODE_LONG_VERSION_0=Version: {pkg_version}{version_suffix}");
    println!("cargo:rustc-env=KONA_NODE_LONG_VERSION_1=Commit SHA: {sha}");
    println!(
        "cargo:rustc-env=KONA_NODE_LONG_VERSION_2=Build Timestamp: {}",
        env::var("VERGEN_BUILD_TIMESTAMP")?
    );
    println!(
        "cargo:rustc-env=KONA_NODE_LONG_VERSION_3=Build Features: {}",
        if features.is_empty() { "no features enabled".to_string() } else { features }
    );
    println!("cargo:rustc-env=KONA_NODE_LONG_VERSION_4=Build Profile: {profile}");

    Ok(())
}
