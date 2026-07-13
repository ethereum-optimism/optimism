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
        // TODO(#18326): Remove this hacky temporary fallback once SP1 ELF
        // builds are properly integrated into monorepo CI. Right now the
        // generated ELF artifacts are ignored by git, and the monorepo Rust CI
        // does not install the SP1 toolchain or run the Dockerized
        // `cargo-prove` ELF build. The empty files written here keep normal
        // host-toolchain compile, clippy, rustdoc, and test jobs working
        // without making SP1 a global CI prerequisite. Runtime bench/proof
        // entrypoints still reject these empty placeholders and require
        // `cd rust/kona/sp1 && just build-elfs` first.
        fs::write(dest, b"")
            .unwrap_or_else(|err| panic!("failed to write {}: {err}", dest.display()));
    }
}
