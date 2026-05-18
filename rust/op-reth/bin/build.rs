// Adapted from reth [https://github.com/paradigmxyz/reth/blob/main/crates/node/core/build.rs]
// and op-rbuilder [https://github.com/flashbots/op-rbuilder/blob/main/crates/op-rbuilder/build.rs]

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
