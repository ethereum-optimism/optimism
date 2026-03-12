//! Blob store utilities for the mock L1 chain.
//!
//! Provides helpers for creating and storing EIP-4844 blobs.

use alloy_eips::eip4844::Blob;
use alloy_primitives::{B256, Keccak256};

/// Computes a mock versioned hash from blob data.
///
/// In production, this would use KZG commitment. For testing, we use
/// a simple keccak hash with the EIP-4844 version byte prefix.
pub fn mock_versioned_hash(blob: &Blob) -> B256 {
    let mut hasher = Keccak256::new();
    hasher.update(AsRef::<[u8]>::as_ref(blob));
    let mut hash = hasher.finalize();
    // Set the version byte (0x01 for EIP-4844)
    hash[0] = 0x01;
    hash
}

/// Creates a blob containing the given data, zero-padded to the full blob size.
pub fn data_to_blob(data: &[u8]) -> Box<Blob> {
    let mut blob = Box::new(Blob::ZERO);
    // EIP-4844 blob encoding: each 31-byte field element has a leading zero byte.
    // For simplicity in tests, we just copy data directly into the blob bytes.
    let len = data.len().min(blob.len());
    blob[..len].copy_from_slice(&data[..len]);
    blob
}
