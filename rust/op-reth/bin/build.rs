//! Emits `OP_RETH_*` build-time env vars consumed by `src/version.rs`, so that op-reth's
//! reported version/commit reflects its own release rather than the pinned upstream `reth`
//! crate's. Adapted from reth's `crates/node/core/build.rs`.
//!
//! op-reth's release version is the monorepo's `op-reth/v*` git tag (e.g. `op-reth/v2.3.3`), not
//! this crate's Cargo.toml `version`: op-reth isn't published to crates.io, and its Cargo version
//! tracks the vendored `reth-optimism-*` library crates (1.11.x), not op-reth's own releases. We
//! read it from `git describe` when `.git` is available, and from the `OP_RETH_VERSION` build arg
//! otherwise — op-reth's Docker build context (`rust/`) deliberately excludes `.git`, so
//! docker-bake.hcl threads in the commit (`GIT_COMMIT`) and the tag (`OP_RETH_VERSION`, sourced
//! from the per-image `GIT_VERSION`) as build args.

use std::{env, error::Error};
use vergen::{BuildBuilder, CargoBuilder, Emitter};
use vergen_git2::Git2Builder;

/// vergen's placeholder for a git instruction it couldn't resolve — emitted (instead of failing
/// the build) when there's no `.git`, as in op-reth's Docker build context.
const VERGEN_IDEMPOTENT_OUTPUT: &str = "VERGEN_IDEMPOTENT_OUTPUT";

/// Release version reported when neither a reachable `op-reth/v*` tag nor an `OP_RETH_VERSION`
/// build arg is available (a shallow/tagless checkout, or a non-release Docker build). Always
/// paired with a `-dev` suffix so it's never mistaken for a real `0.0.0` release.
const DEV_VERSION: &str = "0.0.0";

/// Parses a `git describe --tags --match 'op-reth/v*'` string into `(base_version, on_tag)`:
/// - `op-reth/v2.3.3`             -> `Some(("2.3.3", true))`
/// - `op-reth/v2.3.3-12-gaf90f02` -> `Some(("2.3.3", false))` (12 commits past the tag)
/// - `op-reth/v2.3.3-rc.1`        -> `Some(("2.3.3-rc.1", true))`
/// - anything else (e.g. vergen's `VERGEN_IDEMPOTENT_OUTPUT` when `.git` is absent) -> `None`
fn parse_describe(describe: &str) -> Option<(&str, bool)> {
    let rest = describe.strip_prefix("op-reth/v")?;
    // `git describe` appends `-<commits>-g<sha>` once HEAD moves past the tag. Split from the
    // right so multi-segment prerelease tags (`2.3.3-rc.1`) survive, then strip that suffix.
    let mut it = rest.rsplitn(3, '-');
    match (it.next(), it.next(), it.next()) {
        (Some(sha), Some(commits), Some(base))
            if commits.bytes().all(|b| b.is_ascii_digit())
                && sha
                    .strip_prefix('g')
                    .is_some_and(|s| !s.is_empty() && s.bytes().all(|b| b.is_ascii_hexdigit())) =>
        {
            Some((base, false))
        }
        _ => Some((rest, true)),
    }
}

fn main() -> Result<(), Box<dyn Error>> {
    let mut emitter = Emitter::default();

    let build_builder = BuildBuilder::default().build_timestamp(true).build()?;
    emitter.add_instructions(&build_builder)?;

    let cargo_builder = CargoBuilder::default().features(true).target_triple(true).build()?;
    emitter.add_instructions(&cargo_builder)?;

    // `describe(tags, dirty, matches)`: match `op-reth/v*` so `VERGEN_GIT_DESCRIBE` reflects
    // op-reth's own release tag rather than whatever tag (op-node, cannon, ...) is nearest in this
    // shared monorepo. `tags = true` to also see lightweight tags; `dirty = false` keeps the
    // describe string clean for parsing — the separate `.dirty(true)` still emits
    // `VERGEN_GIT_DIRTY`, which we fold into the `-dev` suffix below.
    let git_builder = Git2Builder::default()
        .describe(true, false, Some("op-reth/v*"))
        .dirty(true)
        .sha(false)
        .build()?;
    emitter.add_instructions(&git_builder)?;

    emitter.emit_and_set()?;

    let discovered_sha = env::var("VERGEN_GIT_SHA")?;
    let describe = env::var("VERGEN_GIT_DESCRIBE").unwrap_or_default();

    // With no `.git` in the Docker context, vergen can't discover the commit or the tag;
    // docker-bake.hcl threads both in as build args. Prefer discovered values, fall back to args.
    let git_commit_arg = env::var("GIT_COMMIT").unwrap_or_default();
    let is_real_sha =
        git_commit_arg.len() >= 8 && git_commit_arg.chars().all(|c| c.is_ascii_hexdigit());
    // `OP_RETH_VERSION` (from bake's per-image `GIT_VERSION`) is the tag with its `op-reth/`
    // prefix stripped, e.g. `v2.3.3`; a non-release build passes an untagged marker instead.
    let version_arg = env::var("OP_RETH_VERSION").unwrap_or_default();
    let arg_version = version_arg.strip_prefix('v').unwrap_or(&version_arg);
    let arg_is_release = arg_version.starts_with(|c: char| c.is_ascii_digit());
    println!("cargo:rerun-if-env-changed=GIT_COMMIT");
    println!("cargo:rerun-if-env-changed=OP_RETH_VERSION");

    let (sha, pkg_version, version_suffix) = if discovered_sha != VERGEN_IDEMPOTENT_OUTPUT {
        // `.git` was discoverable: take both the commit and the release version from it.
        let is_dirty = env::var("VERGEN_GIT_DIRTY")? == "true";
        let (base, on_tag) = parse_describe(&describe).unwrap_or((DEV_VERSION, false));
        (discovered_sha, base.to_string(), if is_dirty || !on_tag { "-dev" } else { "" })
    } else if is_real_sha {
        // Docker: no `.git`, but the build args carry the real commit and (for release builds) the
        // tag. Images are built from a clean checkout of that exact tagged commit, so there's no
        // dirty/off-tag state to detect.
        if arg_is_release {
            (git_commit_arg, arg_version.to_string(), "")
        } else {
            (git_commit_arg, DEV_VERSION.to_string(), "-dev")
        }
    } else {
        // No `.git` to discover and no usable `GIT_COMMIT` arg — don't ship a truncated vergen
        // placeholder as though it were a real commit/version.
        ("unknown".to_string(), DEV_VERSION.to_string(), "-dev")
    };
    // Match reth's convention: 7-char SHA in the human/P2P version strings.
    let sha_short = &sha[..sha.len().min(7)];
    println!("cargo:rustc-env=OP_RETH_VERSION_SUFFIX={version_suffix}");

    // Re-emit the (possibly arg-sourced) SHA under vergen's own env var names, overriding what
    // `emit_and_set` printed, so `src/version.rs`'s `env!("VERGEN_GIT_SHA*")` pick up corrections.
    println!("cargo:rustc-env=VERGEN_GIT_SHA={sha}");
    println!("cargo:rustc-env=VERGEN_GIT_SHA_SHORT={}", &sha[..sha.len().min(8)]);

    // Re-export the resolved op-reth release version for `src/version.rs`, whose
    // `cargo_pkg_version` field feeds `engine_getClientVersionV1`. Suffix-free, mirroring reth's
    // own `cargo_pkg_version` (the `-dev` marker only rides along in the formatted strings below).
    println!("cargo:rustc-env=OP_RETH_PKG_VERSION={pkg_version}");

    // Set the build profile
    let out_dir = env::var("OUT_DIR").unwrap();
    let profile = out_dir.rsplit(std::path::MAIN_SEPARATOR).nth(3).unwrap();
    println!("cargo:rustc-env=OP_RETH_BUILD_PROFILE={profile}");

    // The short version information for op-reth.
    // - op-reth's release version (from the `op-reth/v*` git tag)
    // - The short SHA of the latest commit.
    // Example: 2.3.3 (af90f02)
    println!("cargo:rustc-env=OP_RETH_SHORT_VERSION={pkg_version}{version_suffix} ({sha_short})");

    // LONG_VERSION
    // The long version information for op-reth.
    //
    // - op-reth's release version (from the git tag) + version suffix (if any)
    // - The full SHA of the latest commit
    // - The build datetime
    // - The build features
    // - The build profile
    //
    // Example:
    //
    // ```text
    // Version: 2.3.3
    // Commit SHA: af90f026ce05f8...
    // Build Timestamp: 2023-05-19T01:47:19.815651705Z
    // Build Features: jemalloc
    // Build Profile: maxperf
    // ```
    println!("cargo:rustc-env=OP_RETH_LONG_VERSION_0=Version: {pkg_version}{version_suffix}");
    println!("cargo:rustc-env=OP_RETH_LONG_VERSION_1=Commit SHA: {sha}");
    println!(
        "cargo:rustc-env=OP_RETH_LONG_VERSION_2=Build Timestamp: {}",
        env::var("VERGEN_BUILD_TIMESTAMP")?
    );
    println!(
        "cargo:rustc-env=OP_RETH_LONG_VERSION_3=Build Features: {}",
        env::var("VERGEN_CARGO_FEATURES")?
    );
    println!("cargo:rustc-env=OP_RETH_LONG_VERSION_4=Build Profile: {profile}");

    // The version information for op-reth formatted for P2P (devp2p).
    // - op-reth's release version (from the `op-reth/v*` git tag)
    // - The target triple
    //
    // Example: op-reth/v2.3.3-af90f02/aarch64-apple-darwin
    println!(
        "cargo:rustc-env=OP_RETH_P2P_CLIENT_VERSION={}",
        format_args!(
            "op-reth/v{pkg_version}-{sha_short}/{}",
            env::var("VERGEN_CARGO_TARGET_TRIPLE")?
        )
    );

    Ok(())
}
