use std::{env, error::Error};
use vergen::{BuildBuilder, Emitter};
use vergen_git2::Git2Builder;

fn main() -> Result<(), Box<dyn Error>> {
    let mut emitter = Emitter::default();

    let build_builder = BuildBuilder::default().build_timestamp(true).build()?;
    emitter.add_instructions(&build_builder)?;

    let git_builder = Git2Builder::default()
        .describe(false, true, None)
        .dirty(true)
        .sha(false)
        .build()?;

    emitter.add_instructions(&git_builder)?;

    emitter.emit_and_set()?;
    let sha = env::var("VERGEN_GIT_SHA")?;
    let sha_short = &sha[0..8];

    // Set short SHA
    println!("cargo:rustc-env=VERGEN_GIT_SHA_SHORT={}", &sha_short);

    Ok(())
}
