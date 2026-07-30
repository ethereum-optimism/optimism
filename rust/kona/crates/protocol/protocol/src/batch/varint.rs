//! Base128 varint decoding for span-batch `uvarint` fields.
//!
//! The decoder below is a local port of op-node's `binary.ReadUvarint` rather than a crate call.
//! Crates that decode this format identically do exist — `prost`, `leb128` and `integer-encoding`
//! all match it byte for byte — but `leb128` and `integer-encoding` are `std`-only, and `prost`
//! takes an `unsafe` fast path and allocates a `DecodeError` when it rejects. Both matter because
//! this crate is compiled into the fault proof VM, so the port keeps the decode path safe and
//! allocation-free while naming the implementation it has to agree with.
//!
//! Note that `unsigned-varint` implements a *different* varint family — the minimal-only
//! multiformats encoding — so it rejects the non-minimal encodings this format permits.

/// Maximum number of bytes in the Base128 varint encoding of a `u64`.
const MAX_VARINT_LEN_64: usize = 10;

/// Decodes a Base128 varint from the front of `buf`, returning the value and the remaining input,
/// or [`None`] if `buf` does not begin with a varint representing a `u64`.
///
/// Span-batch `uvarint` fields use the protobuf Base128 encoding, which permits non-minimal
/// encodings: a redundant trailing zero (`[0x81, 0x00]`) decodes to the same value as its minimal
/// form (`[0x01]`). At most `MAX_VARINT_LEN_64` bytes are read, and the tenth byte terminates the
/// varint only if it is `0x00` or `0x01`, since a larger terminator would set bits above 63.
///
/// This accepts exactly the encodings op-node's `binary.ReadUvarint` accepts. The two decoders
/// must agree: a span batch that one client decodes and the other rejects splits derivation on
/// identical L1 data.
pub(super) fn read_uvarint(buf: &[u8]) -> Option<(u64, &[u8])> {
    let mut value = 0u64;
    let mut shift = 0u32;
    // The read bound does double duty: it is op-node's `MaxVarintLen64` limit, and it caps every
    // shift that executes at 63, so none can overflow. (A tenth continuation byte still leaves
    // `shift` at 70, but the loop ends before it is used again.)
    for (i, &byte) in buf.iter().take(MAX_VARINT_LEN_64).enumerate() {
        if byte < 0x80 {
            if i == MAX_VARINT_LEN_64 - 1 && byte > 1 {
                return None;
            }
            return Some((value | ((byte as u64) << shift), &buf[i + 1..]));
        }
        value |= ((byte & 0x7F) as u64) << shift;
        shift += 7;
    }
    // The varint either ran past the maximum width or off the end of the buffer.
    None
}
