//! Tests for the output-root preimage validation in the helpers that decode
//! an L2 output root into the safe head block hash.
//!
//! Two parallel helpers are covered:
//!   * `single::fetch_safe_head_hash` (non-interop fault proof path, uses `get_exact` with a `[u8;
//!     128]` buffer).
//!   * `interop::util::fetch_output_block_hash` (interop path, uses `.get()` and validates the
//!     length explicitly).
//!
//! For each helper we assert that both the version-word and the preimage-
//! length guard reject malformed inputs, so refactors to either function are
//! caught independently.

use alloy_primitives::B256;
use kona_client::{interop, single};
use kona_preimage::{PreimageKey, errors::PreimageOracleError};
use kona_proof::errors::OracleProviderError;

mod common;
use common::MockOracle;

fn b256(fill: u8) -> B256 {
    B256::from([fill; 32])
}

#[tokio::test(flavor = "multi_thread")]
async fn fetch_safe_head_hash_rejects_unknown_output_version() {
    let agreed_root = b256(0xAA);
    let mut bad_preimage = [0u8; 128];
    bad_preimage[0] = 0x01; // non-V0 version word; [96..128] stays zero
    let oracle =
        MockOracle::single(PreimageKey::new_keccak256(*agreed_root), bad_preimage.to_vec());

    let err = single::fetch_safe_head_hash(&oracle, agreed_root).await.unwrap_err();
    match err {
        OracleProviderError::UnknownOutputVersion(version) => {
            assert_eq!(version[0], 0x01);
        }
        other => panic!("unexpected error: {other:?}"),
    }
}

#[tokio::test(flavor = "multi_thread")]
async fn fetch_safe_head_hash_rejects_wrong_preimage_length() {
    // `get_exact` with a `[u8; 128]` buffer is what enforces the length here;
    // this test guards against a future refactor swapping to `get`.
    let agreed_root = b256(0xAA);
    let oracle = MockOracle::single(PreimageKey::new_keccak256(*agreed_root), vec![0u8; 127]);

    let err = single::fetch_safe_head_hash(&oracle, agreed_root).await.unwrap_err();
    match err {
        OracleProviderError::Preimage(PreimageOracleError::BufferLengthMismatch(
            expected,
            actual,
        )) => {
            assert_eq!(expected, 128);
            assert_eq!(actual, 127);
        }
        other => panic!("unexpected error: {other:?}"),
    }
}

#[tokio::test(flavor = "multi_thread")]
async fn fetch_output_block_hash_rejects_unknown_output_version() {
    let output_root = b256(0xAA);
    let mut bad_preimage = vec![0u8; 128];
    bad_preimage[0] = 0x01; // non-V0 version word
    let oracle = MockOracle::single(PreimageKey::new_keccak256(*output_root), bad_preimage);

    let err = interop::util::fetch_output_block_hash(&oracle, output_root, 10).await.unwrap_err();
    match err {
        OracleProviderError::UnknownOutputVersion(version) => {
            assert_eq!(version[0], 0x01);
        }
        other => panic!("unexpected error: {other:?}"),
    }
}

#[tokio::test(flavor = "multi_thread")]
async fn fetch_output_block_hash_rejects_wrong_preimage_length() {
    // 127-byte preimage with a V0 version word; without the length guard this
    // slips past the version check and then panics on the [96..128] slice.
    let output_root = b256(0xAA);
    let oracle = MockOracle::single(PreimageKey::new_keccak256(*output_root), vec![0u8; 127]);

    let err = interop::util::fetch_output_block_hash(&oracle, output_root, 10).await.unwrap_err();
    match err {
        OracleProviderError::Preimage(PreimageOracleError::BufferLengthMismatch(
            expected,
            actual,
        )) => {
            assert_eq!(expected, 128);
            assert_eq!(actual, 127);
        }
        other => panic!("unexpected error: {other:?}"),
    }
}
