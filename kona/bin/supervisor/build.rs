//! Used for generating build information for the supervisor service.

use std::{env, error::Error};
use vergen::{BuildBuilder, CargoBuilder, Emitter};
use vergen_git2::Git2Builder;

fn main() -> Result<(), Box<dyn Error>> {
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

    // Need to print in order to set the environment variables.
    let sha = env::var("VERGEN_GIT_SHA")?;
    println!("cargo:rustc-env=VERGEN_GIT_SHA_SHORT={}", &sha[..8]);

    let out_dir = env::var("OUT_DIR").unwrap();
    let profile = out_dir.rsplit(std::path::MAIN_SEPARATOR).nth(3).unwrap();
    println!("cargo:rustc-env=KONA_SUPERVISOR_BUILD_PROFILE={profile}");

    Ok(())
}
