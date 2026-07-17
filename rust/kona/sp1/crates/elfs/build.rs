use std::{
    env, fs,
    path::{Path, PathBuf},
};

const ELF_NAMES: [&str; 4] =
    ["aggregation-elf", "range-elf", "super-aggregation-elf", "super-range-elf"];

fn main() {
    let manifest_dir =
        PathBuf::from(env::var("CARGO_MANIFEST_DIR").expect("CARGO_MANIFEST_DIR is set"));
    let source_dir = manifest_dir.join("../../elf");
    let out_dir = PathBuf::from(env::var("OUT_DIR").expect("OUT_DIR is set"));

    println!("cargo:rerun-if-changed={}", source_dir.display());

    for elf_name in ELF_NAMES {
        let source = source_dir.join(elf_name);
        let dest = out_dir.join(elf_name);

        println!("cargo:rerun-if-changed={}", source.display());
        write_elf_input(&source, &dest);
    }
}

fn write_elf_input(source: &Path, dest: &Path) {
    if source.is_file() {
        fs::copy(source, dest).unwrap_or_else(|err| {
            panic!("failed to copy {} to {}: {err}", source.display(), dest.display())
        });
    } else {
        // Host-toolchain jobs run without generated, gitignored ELFs.
        // Runtime proof paths reject these empty placeholders.
        fs::write(dest, b"")
            .unwrap_or_else(|err| panic!("failed to write {}: {err}", dest.display()));
    }
}
