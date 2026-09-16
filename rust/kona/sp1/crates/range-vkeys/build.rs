use std::{env, fs, io, path::PathBuf};

#[path = "src/codec.rs"]
mod codec;

use codec::{parse_hex32, unpack_vkey_bytes32, vkey_hex};

fn main() {
    let manifest_dir =
        PathBuf::from(env::var("CARGO_MANIFEST_DIR").expect("CARGO_MANIFEST_DIR is set"));
    let vkeys_path = manifest_dir.join("../../elf/vkeys.toml");
    // The manifest is outside this crate, so Cargo's default package tracking does not cover it.
    println!("cargo:rerun-if-changed={}", vkeys_path.display());

    let manifest = match fs::read_to_string(&vkeys_path) {
        Ok(manifest) => manifest,
        Err(err) if err.kind() == io::ErrorKind::NotFound => panic!(
            "elf/vkeys.toml not found — build the super-range ELF first (cd rust/kona/sp1 && just \
             build-elfs); required to build the super-aggregation guest."
        ),
        Err(err) => panic!("failed to read {}: {err}", vkeys_path.display()),
    };

    let super_range_vkey = decode_vkey(&manifest, "super-range");
    let generated = format!(
        "/// Verification key of the `super-range` guest, embedded in the super-aggregation \
         guest.\n\
         pub const SUPER_RANGE_VKEY: [u32; 8] = {super_range_vkey:?};\n"
    );

    let out_dir = PathBuf::from(env::var("OUT_DIR").expect("OUT_DIR is set"));
    let output = out_dir.join("vkeys.rs");
    fs::write(&output, generated)
        .unwrap_or_else(|err| panic!("failed to write {}: {err}", output.display()));
}

fn decode_vkey(manifest: &str, key: &str) -> [u32; 8] {
    unpack_vkey_bytes32(&parse_hex32(vkey_hex(manifest, key)))
}
