//! Build script for `kona-hardforks`.
//!
//! Reads NUT bundle JSON files from `op-core/nuts/bundles/` and generates Rust source
//! that constructs [`op_alloy_consensus::NutBundle`] values at runtime without serde.
//!
//! The parsing and codegen logic lives in [`build_helpers`] so it can be shared
//! with integration tests under `tests/`.

use std::{env, fs, path::PathBuf};

use anyhow::{Context, Result, anyhow};

#[path = "build_helpers.rs"]
mod build_helpers;

use build_helpers::{capitalize, format_bundle, parse_bundle};

/// Read the bundle JSON, generate Rust source, and write it to `out_dir`.
fn generate(name: &str, json_path: &PathBuf, out_dir: &str) -> Result<()> {
    let json =
        fs::read_to_string(json_path).with_context(|| format!("read {}", json_path.display()))?;
    let bundle = parse_bundle(&json).with_context(|| format!("parse {}", json_path.display()))?;
    let code = format_bundle(name, &capitalize(name), &bundle);
    let out_path = PathBuf::from(out_dir).join(format!("{name}_nut_bundle.rs"));
    fs::write(&out_path, code).with_context(|| format!("write {}", out_path.display()))?;
    Ok(())
}

fn run() -> Result<()> {
    let out_dir = env::var("OUT_DIR").context("OUT_DIR not set")?;
    let manifest_dir =
        PathBuf::from(env::var("CARGO_MANIFEST_DIR").context("CARGO_MANIFEST_DIR not set")?);

    // Probe for the karst bundle file rather than the `op-core/` directory: Docker
    // bind-mounts auto-create empty `op-core/` ancestor stubs on the host, and an
    // `is_dir()` check picks those up before reaching the real op-core at the
    // monorepo root. Probing for a known file inside the bundle path skips the stubs.
    let monorepo_root = manifest_dir
        .ancestors()
        .find(|p| p.join("op-core/nuts/bundles/karst_nut_bundle.json").is_file())
        .ok_or_else(|| {
            anyhow!(
                "could not find op-core/nuts/bundles/karst_nut_bundle.json in any ancestor of {}",
                manifest_dir.display()
            )
        })?
        .to_path_buf();

    let karst_bundle = monorepo_root.join("op-core/nuts/bundles/karst_nut_bundle.json");
    println!("cargo::rerun-if-changed={}", karst_bundle.display());

    generate("karst", &karst_bundle, &out_dir).context("generate karst bundle")
}

fn main() {
    if let Err(e) = run() {
        panic!("{e:?}");
    }
}
