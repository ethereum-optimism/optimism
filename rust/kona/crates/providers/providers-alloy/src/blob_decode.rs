//! Standalone EIP-4844 blob payload decoder.
//!
//! The decoder reverses the blob-encoding scheme used by op-batcher to pack
//! arbitrary derivation bytes into a sequence of BLS12-381 field elements. It
//! is lifted out of `kona_derive::sources::blob_data` so the pure-derivation
//! driver in this crate can consume blob bytes without depending on the
//! soon-to-be-removed async derivation sources. The pure deriver itself stays
//! IO-free; KZG verification + version-byte stripping happen here, in the
//! caller of [`kona_derive::extract_l1_input`].

use alloy_eips::eip4844::{BYTES_PER_BLOB, VERSIONED_HASH_VERSION_KZG};
use alloy_primitives::Bytes;
use kona_derive::BlobDecodingError;

/// Blob encoding version byte. Matches the constant the batcher writes.
const BLOB_ENCODING_VERSION: u8 = 0;

/// Maximum number of payload bytes a single blob can carry.
const BLOB_MAX_DATA_SIZE: usize = (4 * 31 + 3) * 1024 - 4; // 130044

/// How many full encoding rounds (4 field elements each) make up a blob.
const BLOB_ENCODING_ROUNDS: usize = 1024;

/// Decode a single blob's `BYTES_PER_BLOB` bytes into the original derivation
/// payload.
///
/// `blob` must be exactly [`BYTES_PER_BLOB`] bytes long. KZG verification
/// against the versioned hash is the caller's responsibility.
pub fn decode_blob(blob: &[u8]) -> Result<Bytes, BlobDecodingError> {
    if blob.len() != BYTES_PER_BLOB {
        return Err(BlobDecodingError::MissingData);
    }

    if blob[VERSIONED_HASH_VERSION_KZG as usize] != BLOB_ENCODING_VERSION {
        return Err(BlobDecodingError::InvalidEncodingVersion);
    }

    let length = u32::from_be_bytes([0, blob[2], blob[3], blob[4]]) as usize;
    if length > BLOB_MAX_DATA_SIZE {
        return Err(BlobDecodingError::InvalidLength);
    }

    // Output buffer holds the maximum payload size; we truncate to `length` at the end.
    let mut output = alloc::vec![0u8; BLOB_MAX_DATA_SIZE];
    output[0..27].copy_from_slice(&blob[5..32]);

    let mut output_pos = 28usize;
    let mut input_pos = 32usize;

    let mut encoded_byte = [0u8; 4];
    encoded_byte[0] = blob[0];

    // Round 0 — finish decoding the first 4 field elements (one is already partially
    // copied above).
    for b in encoded_byte.iter_mut().skip(1) {
        let (enc, opos, ipos) = decode_field_element(blob, output_pos, input_pos, &mut output)?;
        *b = enc;
        output_pos = opos;
        input_pos = ipos;
    }
    output_pos = reassemble_bytes(output_pos, &encoded_byte, &mut output);

    for _ in 1..BLOB_ENCODING_ROUNDS {
        if output_pos >= length {
            break;
        }
        for d in &mut encoded_byte {
            let (enc, opos, ipos) = decode_field_element(blob, output_pos, input_pos, &mut output)?;
            *d = enc;
            output_pos = opos;
            input_pos = ipos;
        }
        output_pos = reassemble_bytes(output_pos, &encoded_byte, &mut output);
    }

    // Bytes past `length` in the decoded output must be zero — they came from padding.
    for o in output.iter().skip(length) {
        if *o != 0u8 {
            return Err(BlobDecodingError::InvalidFieldElement);
        }
    }
    output.truncate(length);

    // Bytes past `input_pos` in the encoded blob must also be zero.
    for tail_byte in &blob[input_pos..BYTES_PER_BLOB] {
        if *tail_byte != 0 {
            return Err(BlobDecodingError::InvalidFieldElement);
        }
    }

    Ok(Bytes::from(output))
}

/// Read one field element starting at `input_pos`. Returns the high-order
/// byte (which carries 6 bits of the next 3-byte payload triple), the updated
/// output position, and the updated input position.
fn decode_field_element(
    blob: &[u8],
    output_pos: usize,
    input_pos: usize,
    output: &mut [u8],
) -> Result<(u8, usize, usize), BlobDecodingError> {
    // The two highest order bits of the first byte of each field element must be zero
    // for the BLS12-381 scalar to be canonical.
    if blob[input_pos] & 0b1100_0000 != 0 {
        return Err(BlobDecodingError::InvalidFieldElement);
    }
    output[output_pos..output_pos + 31].copy_from_slice(&blob[input_pos + 1..input_pos + 32]);
    Ok((blob[input_pos], output_pos + 32, input_pos + 32))
}

/// Reassemble four 6-bit chunks (held in the top of each preceding field
/// element) into three payload bytes interleaved into `output`.
fn reassemble_bytes(mut output_pos: usize, encoded_byte: &[u8], output: &mut [u8]) -> usize {
    output_pos -= 1;
    let x = (encoded_byte[0] & 0b0011_1111) | ((encoded_byte[1] & 0b0011_0000) << 2);
    let y = (encoded_byte[1] & 0b0000_1111) | ((encoded_byte[3] & 0b0000_1111) << 4);
    let z = (encoded_byte[2] & 0b0011_1111) | ((encoded_byte[3] & 0b0011_0000) << 2);
    output[output_pos - 32] = z;
    output[output_pos - (32 * 2)] = y;
    output[output_pos - (32 * 3)] = x;
    output_pos
}

extern crate alloc;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn wrong_length_input_errors() {
        let blob = vec![0u8; BYTES_PER_BLOB - 1];
        assert!(matches!(decode_blob(&blob), Err(BlobDecodingError::MissingData)));
    }

    #[test]
    fn wrong_encoding_version_errors() {
        let mut blob = vec![0u8; BYTES_PER_BLOB];
        blob[VERSIONED_HASH_VERSION_KZG as usize] = 0xFF;
        assert!(matches!(decode_blob(&blob), Err(BlobDecodingError::InvalidEncodingVersion)));
    }

    #[test]
    fn length_too_large_errors() {
        let mut blob = vec![0u8; BYTES_PER_BLOB];
        blob[VERSIONED_HASH_VERSION_KZG as usize] = BLOB_ENCODING_VERSION;
        // Encode a huge length in bytes [2..5].
        blob[2] = 0xFF;
        blob[3] = 0xFF;
        blob[4] = 0xFF;
        assert!(matches!(decode_blob(&blob), Err(BlobDecodingError::InvalidLength)));
    }

    #[test]
    fn empty_payload_roundtrips_to_empty_bytes() {
        // An all-zero blob with the canonical version byte set decodes to an empty payload.
        let mut blob = vec![0u8; BYTES_PER_BLOB];
        blob[VERSIONED_HASH_VERSION_KZG as usize] = BLOB_ENCODING_VERSION;
        // Length bytes are all zero — length 0.
        let out = decode_blob(&blob).expect("empty blob decodes");
        assert!(out.is_empty());
    }
}
