//! Compile-time SP1 log-commitment guest verification key, embedded from `elf/vkeys.toml`.
//!
//! The super-range guest verifies log-commitment batch proofs against this key only. The build
//! makes the log-commitment ELF first, so the key is that of the ELF built from the same tree.

#![cfg_attr(not(test), no_std)]

include!(concat!(env!("OUT_DIR"), "/vkeys.rs"));

#[cfg(test)]
#[path = "../../range-vkeys/src/codec.rs"]
mod codec;

#[cfg(test)]
mod tests {
    use super::LOG_COMMITMENT_VKEY;
    use crate::codec::{parse_hex32, repack_vkey_words, unpack_vkey_bytes32, vkey_hex};

    const SP1_FIELD_MODULUS: u32 = 2_130_706_433;
    const VKEYS_TOML: &str = include_str!("../../../elf/vkeys.toml");

    #[test]
    fn log_commitment_vkey_packs_back_to_committed_bytes() {
        let committed = parse_hex32(vkey_hex(VKEYS_TOML, "log-commitment"));
        assert_eq!(unpack_vkey_bytes32(&committed), LOG_COMMITMENT_VKEY);
        assert_eq!(repack_vkey_words(LOG_COMMITMENT_VKEY), committed);
        assert!(LOG_COMMITMENT_VKEY.into_iter().all(|word| word < SP1_FIELD_MODULUS));
    }
}
