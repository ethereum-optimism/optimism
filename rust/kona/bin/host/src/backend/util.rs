//! Utilities for the preimage server backend.

use crate::{KeyValueStore, Result};
use alloy_consensus::EMPTY_ROOT_HASH;
use alloy_primitives::keccak256;
use alloy_rlp::EMPTY_STRING_CODE;
use kona_preimage::{PreimageKey, PreimageKeyType};
use tokio::sync::RwLock;

/// Constructs a merkle patricia trie from the ordered list passed and stores all encoded
/// intermediate nodes of the trie in the [`KeyValueStore`].
///
/// # Errors
///
/// Returns an error if any of the [`KeyValueStore::set`] writes fail.
#[allow(clippy::future_not_send, clippy::significant_drop_tightening, clippy::redundant_pub_crate)]
// SAFETY: the write guard must be held across the loop to keep node insertions atomic; the
// future is `!Send` because callers drive it from a single-threaded async runtime;
// `pub(crate)` is required to satisfy the workspace-level `unreachable_pub` lint.
pub(crate) async fn store_ordered_trie<KV: KeyValueStore + ?Sized, T: AsRef<[u8]>>(
    kv: &RwLock<KV>,
    values: &[T],
) -> Result<()> {
    let mut kv_write_lock = kv.write().await;

    // If the list of nodes is empty, store the empty root hash and exit early.
    // The `HashBuilder` will not push the preimage of the empty root hash to the
    // `ProofRetainer` in the event that there are no leaves inserted.
    if values.is_empty() {
        let empty_key = PreimageKey::new(*EMPTY_ROOT_HASH, PreimageKeyType::Keccak256);
        return kv_write_lock.set(empty_key.into(), [EMPTY_STRING_CODE].into());
    }

    let mut hb = kona_mpt::ordered_trie_with_encoder(values, |node, buf| {
        buf.put_slice(node.as_ref());
    });
    hb.root();
    let intermediates = hb.take_proof_nodes().into_inner();

    for (_, value) in intermediates {
        let value_hash = keccak256(value.as_ref());
        let key = PreimageKey::new(*value_hash, PreimageKeyType::Keccak256);

        kv_write_lock.set(key.into(), value.into())?;
    }

    Ok(())
}
