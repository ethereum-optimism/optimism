//! Resolves the monorepo commit to embed in the SP1 guests.
//!
//! The commit arrives on two channels because `cargo prove build --docker` forwards neither the
//! host environment nor `.git`: an environment variable for native builds, and a `--cfg` rustflag
//! for Docker builds. Same dual channel `kona_custom_configs_dir` uses to reach `kona-registry`.

use std::env;

#[path = "src/sha.rs"]
mod sha;

fn main() {
    println!("cargo:rerun-if-env-changed=KONA_SP1_GIT_SHA");
    println!("cargo:rerun-if-env-changed=CARGO_CFG_KONA_SP1_GIT_SHA");

    let sha = env::var("KONA_SP1_GIT_SHA")
        .ok()
        .filter(|sha| !sha.is_empty())
        .or_else(|| env::var("CARGO_CFG_KONA_SP1_GIT_SHA").ok())
        .unwrap_or_else(|| sha::UNKNOWN.to_string());

    assert!(
        sha::is_valid(&sha),
        "KONA_SP1_GIT_SHA (or --cfg kona_sp1_git_sha=\"...\") must be a 40-character lowercase \
         hex sha, optionally suffixed `-dirty` and `-custom`; got {sha:?}"
    );

    println!("cargo:rustc-env=KONA_SP1_GIT_SHA={sha}");
}
