// Taken from reth [https://github.com/paradigmxyz/reth/blob/main/crates/node/core/build.rs]
// The MIT License (MIT)
//
// Copyright (c) 2022-2025 Reth Contributors
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

use std::{env, error::Error};
use vergen::{BuildBuilder, CargoBuilder, Emitter};
use vergen_git2::Git2Builder;

fn main() -> Result<(), Box<dyn Error>> {
    let mut emitter = Emitter::default();

    let build_builder = BuildBuilder::default().build_timestamp(true).build()?;

    emitter.add_instructions(&build_builder)?;

    let cargo_builder = CargoBuilder::default()
        .features(true)
        .target_triple(true)
        .build()?;

    emitter.add_instructions(&cargo_builder)?;

    let git_builder = Git2Builder::default()
        .describe(false, true, None)
        .dirty(true)
        .sha(false)
        .commit_author_name(true)
        .commit_author_email(true)
        .commit_message(true)
        .build()?;

    emitter.add_instructions(&git_builder)?;

    emitter.emit_and_set()?;
    let sha = env::var("VERGEN_GIT_SHA")?;
    let sha_short = &sha[0..7];

    // Set short SHA
    println!("cargo:rustc-env=VERGEN_GIT_SHA_SHORT={}", &sha[..8]);

    let author_name = env::var("VERGEN_GIT_COMMIT_AUTHOR_NAME")?;
    let author_email = env::var("VERGEN_GIT_COMMIT_AUTHOR_EMAIL")?;
    let author_full = format!("{author_name} <{author_email}>");

    // Set author full name
    println!("cargo:rustc-env=VERGEN_GIT_COMMIT_AUTHOR={author_full}");

    let is_dirty = env::var("VERGEN_GIT_DIRTY")? == "true";
    // > git describe --always --tags
    // if not on a tag: v0.2.0-beta.3-82-g1939939b
    // if on a tag: v0.2.0-beta.3
    let not_on_tag = env::var("VERGEN_GIT_DESCRIBE")?.ends_with(&format!("-g{sha_short}"));
    let version_suffix = if is_dirty || not_on_tag { "-dev" } else { "" };
    println!("cargo:rustc-env=OP_RBUILDER_VERSION_SUFFIX={version_suffix}");

    // Set the build profile
    let out_dir = env::var("OUT_DIR").unwrap();
    let profile = out_dir.rsplit(std::path::MAIN_SEPARATOR).nth(3).unwrap();
    println!("cargo:rustc-env=OP_RBUILDER_BUILD_PROFILE={profile}");

    // Set formatted version strings
    let pkg_version = env!("CARGO_PKG_VERSION");

    // The short version information for op-rbuilder.
    // - The latest version from Cargo.toml
    // - The short SHA of the latest commit.
    // Example: 0.1.0 (defa64b2)
    println!(
        "cargo:rustc-env=OP_RBUILDER_SHORT_VERSION={pkg_version}{version_suffix} ({sha_short})"
    );

    // LONG_VERSION
    // The long version information for op-rbuilder.
    //
    // - The latest version from Cargo.toml + version suffix (if any)
    // - The full SHA of the latest commit
    // - The build datetime
    // - The build features
    // - The build profile
    // - The latest commit message and author
    //
    // Example:
    //
    // ```text
    // Version: 0.1.0
    // Commit SHA: defa64b2
    // Build Timestamp: 2023-05-19T01:47:19.815651705Z
    // Build Features: jemalloc
    // Build Profile: maxperf
    // Latest Commit: 'message' by John Doe <john.doe@example.com>
    // ```
    println!("cargo:rustc-env=OP_RBUILDER_LONG_VERSION_0=Version: {pkg_version}{version_suffix}");
    println!("cargo:rustc-env=OP_RBUILDER_LONG_VERSION_1=Commit SHA: {sha}");
    println!(
        "cargo:rustc-env=OP_RBUILDER_LONG_VERSION_2=Build Timestamp: {}",
        env::var("VERGEN_BUILD_TIMESTAMP")?
    );
    println!(
        "cargo:rustc-env=OP_RBUILDER_LONG_VERSION_3=Build Features: {}",
        env::var("VERGEN_CARGO_FEATURES")?
    );
    println!("cargo:rustc-env=OP_RBUILDER_LONG_VERSION_4=Build Profile: {profile}");
    println!(
        "cargo:rustc-env=OP_RBUILDER_LONG_VERSION_5=Latest Commit: '{}' by {}",
        env::var("VERGEN_GIT_COMMIT_MESSAGE")?.trim_end(),
        author_full
    );

    // The version information for op-rbuilder formatted for P2P (devp2p).
    // - The latest version from Cargo.toml
    // - The target triple
    //
    // Example: op-rbuilder/v0.1.0-alpha.1-428a6dc2f/aarch64-apple-darwin
    println!(
        "cargo:rustc-env=OP_RBUILDER_P2P_CLIENT_VERSION={}",
        format_args!(
            "op-rbuilder/v{pkg_version}-{sha_short}/{}",
            env::var("VERGEN_CARGO_TARGET_TRIPLE")?
        )
    );

    Ok(())
}
