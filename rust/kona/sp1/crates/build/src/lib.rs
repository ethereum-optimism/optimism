//! Contains utilities for building ELFs for SP1 programs.

use sp1_build::{BuildArgs, build_program_with_args};

/// Builds a specific program in the programs directory.
#[allow(unused)]
fn build_program(program_name: &str, elf_name: &str, features: Option<Vec<String>>) {
    let metadata =
        cargo_metadata::MetadataCommand::new().exec().expect("Failed to get cargo metadata");

    let mut build_args = BuildArgs {
        elf_name: Some(elf_name.to_string()),
        output_directory: Some("kona/sp1/elf".to_string()),
        docker: true,
        tag: "v6.2.4".to_string(),
        workspace_directory: Some(".".to_string()),
        ignore_rust_version: true,
        ..Default::default()
    };

    if let Some(features) = features {
        build_args.features = features;
    }

    build_program_with_args(
        &format!("{}/{}", metadata.workspace_root.join("kona/sp1/programs"), program_name),
        build_args,
    );
}

/// Build all the native programs and the native host runner. Optional flag to build the zkVM
/// programs.
#[allow(unused)]
pub fn build_all() {
    build_program("aggregation", "aggregation-elf", None);
    build_program("range", "range-elf", None);
}
