//! Compile-time SP1 super-range guest verification key, embedded from `elf/vkeys.toml`.

#![cfg_attr(not(test), no_std)]

include!(concat!(env!("OUT_DIR"), "/vkeys.rs"));

#[cfg(test)]
mod codec;

#[cfg(test)]
mod tests {
    use super::SUPER_RANGE_VKEY;
    use crate::codec::{parse_hex32, unpack_vkey_bytes32, vkey_hex};

    const SP1_FIELD_MODULUS: u32 = 2_130_706_433;
    const VKEYS_TOML: &str = include_str!("../../../elf/vkeys.toml");

    fn repack(words: [u32; 8]) -> [u8; 32] {
        let mut bytes = [0u8; 32];
        for (i, word) in words.iter().copied().enumerate() {
            assert!(word <= 0x7fff_ffff);
            for b in 0..31 {
                let position = 8 + 31 * i + b;
                let bit = ((word >> (30 - b)) & 1) as u8;
                bytes[position / 8] |= bit << (7 - (position % 8));
            }
        }
        bytes
    }

    fn assert_canonical(words: [u32; 8]) {
        assert!(words.into_iter().all(|word| word < SP1_FIELD_MODULUS));
    }

    #[test]
    fn super_range_vkey_packs_back_to_committed_bytes() {
        let committed = parse_hex32(vkey_hex(VKEYS_TOML, "super-range"));
        assert_eq!(repack(SUPER_RANGE_VKEY), committed);
        assert_canonical(SUPER_RANGE_VKEY);
    }

    #[test]
    fn parse_is_format_invariant() {
        const VALUE: &str = "0x0000000000000000000000000000000000000000000000000000000000000001";
        let first = format!(
            "super-range = \"{VALUE}\"\nsuper-aggregation = \
             \"0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\"\n\
             unrelated = \"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff\"\n"
        );
        let second = format!(
            "# reordered with unrelated aggregate data\nsuper-aggregation = \
             \"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\"\n\n\
             unrelated = \
             \"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff\"\n\
             # exact-key parsing ignores comments and spacing\n\
             super-range=\"{VALUE}\"\n"
        );

        let parse =
            |manifest: &str| unpack_vkey_bytes32(&parse_hex32(vkey_hex(manifest, "super-range")));
        assert_eq!(parse(&first), parse(&second));
    }
}
