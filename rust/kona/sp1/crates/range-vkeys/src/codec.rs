//! Dependency-free helpers shared by `build.rs` and the unit tests.

/// Decode a `0x`-prefixed 64-hex-character string into 32 big-endian bytes.
pub(crate) fn parse_hex32(s: &str) -> [u8; 32] {
    let s = s.strip_prefix("0x").unwrap_or(s);
    assert_eq!(s.len(), 64, "vkey hex must be 64 chars, got {}", s.len());

    let mut out = [0u8; 32];
    for (i, byte) in out.iter_mut().enumerate() {
        *byte = u8::from_str_radix(&s[i * 2..i * 2 + 2], 16)
            .unwrap_or_else(|e| panic!("invalid hex in vkey: {e}"));
    }
    out
}

/// Unpack SP1's `bytes32()` (a tight 31-bit packing of eight `SP1Field` elements) into the
/// `[u32; 8]` digest `verify_sp1_proof` expects. `bytes` is a big-endian integer below 2^248, so
/// word `i` occupies bits `[8 + 31*i, 8 + 31*i + 31)` (bit zero is the MSB).
pub(crate) fn unpack_vkey_bytes32(bytes: &[u8; 32]) -> [u32; 8] {
    let bit = |p: usize| ((bytes[p / 8] >> (7 - (p % 8))) & 1) as u32;
    let mut words = [0u32; 8];
    for (i, word) in words.iter_mut().enumerate() {
        let start = 8 + 31 * i;
        for b in 0..31 {
            *word = (*word << 1) | bit(start + b);
        }
    }
    words
}

/// Extract one key's `0x...` value from `vkeys.toml` by exact key.
pub(crate) fn vkey_hex<'a>(toml: &'a str, key: &str) -> &'a str {
    for line in toml.lines() {
        let Some((lhs, rhs)) = line.split_once('=') else {
            continue;
        };
        if lhs.trim() == key {
            return rhs.trim().trim_matches('"');
        }
    }
    panic!("vkeys.toml missing key {key:?}");
}
