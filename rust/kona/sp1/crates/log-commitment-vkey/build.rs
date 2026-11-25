use std::{env, fs, io, path::PathBuf};

#[path = "../range-vkeys/src/codec.rs"]
mod codec;

use codec::{parse_hex32, unpack_vkey_bytes32, vkey_hex};

fn main() {
    let manifest_dir =
        PathBuf::from(env::var("CARGO_MANIFEST_DIR").expect("CARGO_MANIFEST_DIR is set"));
    let vkeys_path = manifest_dir.join("../../elf/vkeys.toml");
    // Cargo does not track the manifest or the shared codec: both are outside this crate.
    println!("cargo:rerun-if-changed={}", vkeys_path.display());
    println!("cargo:rerun-if-changed=../range-vkeys/src/codec.rs");

    let manifest = match fs::read_to_string(&vkeys_path) {
        Ok(manifest) => manifest,
        Err(err) if err.kind() == io::ErrorKind::NotFound => panic!(
            "elf/vkeys.toml not found — build the log-commitment ELF first (cd rust/kona/sp1 && \
             just build-elfs); required to build the super-range guest."
        ),
        Err(err) => panic!("failed to read {}: {err}", vkeys_path.display()),
    };

    let vkey = unpack_vkey_bytes32(&parse_hex32(vkey_hex(&manifest, "log-commitment")));
    let generated = format!(
        "/// Verification key of the `log-commitment` guest, embedded in the super-range guest.\n\
         pub const LOG_COMMITMENT_VKEY: [u32; 8] = {vkey:?};\n"
    );

    let out_dir = PathBuf::from(env::var("OUT_DIR").expect("OUT_DIR is set"));
    let output = out_dir.join("vkeys.rs");
    fs::write(&output, generated)
        .unwrap_or_else(|err| panic!("failed to write {}: {err}", output.display()));
}
