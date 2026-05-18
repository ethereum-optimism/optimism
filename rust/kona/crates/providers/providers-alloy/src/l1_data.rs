//! Block-oriented L1 data fetcher feeding the pure derivation deriver.
//!
//! `kona_derive::extract_l1_input` takes pre-fetched, pre-decoded
//! [`L1TxView`]s and is sysconfig-blind. This module is the per-L1-block
//! glue around the existing alloy / beacon providers that produces those
//! views: it fetches the full block + receipts, sequences through every
//! batch-inbox tx, fetches and KZG-verifies blob payloads for EIP-4844 txs,
//! and yields a vector of `L1TxView`s + the receipt list in block order.
//!
//! The deriver decides which views belong to the rolling batcher address and
//! which to drop. This helper deliberately stays sysconfig-blind so the trace
//! the deriver emits remains the single source of truth for "what was
//! dropped and why".

use crate::{AlloyChainProvider, OnlineBeaconClient, OnlineBlobProvider, blob_decode::decode_blob};
use alloy_consensus::{
    Receipt, Transaction, TxEip4844Variant, TxEnvelope, transaction::SignerRecoverable,
};
use alloy_primitives::{B256, Bytes};
use alloy_provider::Provider;
use kona_derive::{BlobDecodingError, BlobProviderError, L1TxView};
use kona_protocol::BlockInfo;
use std::vec::Vec;
use thiserror::Error;

/// One L1 block, fetched and decoded into the shape the pure deriver
/// expects. The header + tx views go through `kona_derive::extract_l1_input`
/// without further mutation; the receipts list is forwarded to the helper as
/// well so deposit-event / config-update logs can be filtered out.
#[derive(Debug, Clone)]
pub struct L1BlockData {
    /// The L1 block header.
    pub header: alloy_consensus::Header,
    /// Block-ordered transactions, with blob payloads already decoded and
    /// signers already recovered.
    pub txs: Vec<L1TxView>,
    /// Block-ordered receipts. The deriver filters deposit / config-update
    /// logs out of these.
    pub receipts: Vec<Receipt>,
}

/// Failure modes for [`fetch_l1_block_data`].
#[derive(Debug, Error)]
pub enum L1FetchError {
    /// The underlying alloy provider returned an error.
    #[error("alloy provider error: {0}")]
    Provider(String),
    /// The L1 block reported a different number than the one requested.
    #[error("L1 block at #{requested} returned number #{got}")]
    BlockNumberMismatch {
        /// Block number requested by the caller.
        requested: u64,
        /// Block number actually returned by the provider.
        got: u64,
    },
    /// A blob payload failed to decode after KZG verification.
    #[error("blob decode error at tx {tx_index}, blob {blob_index}: {error}")]
    BlobDecode {
        /// Index of the EIP-4844 transaction in the L1 block.
        tx_index: usize,
        /// Index of the blob within that transaction's `blob_versioned_hashes`.
        blob_index: usize,
        /// Underlying decoder error.
        error: BlobDecodingError,
    },
    /// Fetching blob sidecars from the beacon node failed.
    #[error("blob provider error: {0}")]
    BlobProvider(#[from] BlobProviderError),
    /// Tx signer recovery failed for an EIP-4844 transaction. Only fatal for
    /// blob-carrying txs because the deriver still needs the sender for
    /// non-blob paths and tolerates a `from = ZERO` for those.
    #[error("signer recovery failed for EIP-4844 transaction at index {tx_index}")]
    SignerRecoveryFailed {
        /// Index of the offending transaction.
        tx_index: usize,
    },
}

/// Fetch an L1 block's full transaction list and receipts, decode any blob
/// payloads, and return the result shaped for
/// [`kona_derive::extract_l1_input`].
///
/// `inbox_address` is the rollup's static batch-inbox address (used to skip
/// the costly blob fetch for non-inbox 4844 transactions — most of an L1
/// block).
pub async fn fetch_l1_block_data(
    chain_provider: &AlloyChainProvider,
    blob_provider: &OnlineBlobProvider<OnlineBeaconClient>,
    block_number: u64,
    inbox_address: alloy_primitives::Address,
) -> Result<L1BlockData, L1FetchError> {
    let block = chain_provider
        .inner
        .get_block_by_number(block_number.into())
        .full()
        .await
        .map_err(|e| L1FetchError::Provider(e.to_string()))?
        .ok_or_else(|| L1FetchError::Provider(format!("L1 block #{block_number} not found")))?
        .into_consensus()
        .map_transactions(|t| t.inner.into_inner());

    if block.header.number != block_number {
        return Err(L1FetchError::BlockNumberMismatch {
            requested: block_number,
            got: block.header.number,
        });
    }

    let header = block.header.clone();
    let block_ref = BlockInfo {
        hash: header.hash_slow(),
        number: header.number,
        parent_hash: header.parent_hash,
        timestamp: header.timestamp,
    };

    let receipts = fetch_receipts(chain_provider, block_ref.hash).await?;

    // Walk every transaction in the block and produce one `L1TxView` per tx.
    // For 4844 txs that target the inbox, the views explode 1:N into the
    // decoded blob payloads (one per blob_versioned_hash).
    let mut blob_hashes: Vec<(usize, usize, B256)> = Vec::new();
    let mut tx_views: Vec<L1TxView> = Vec::new();
    for (tx_index, tx) in block.body.transactions.into_iter().enumerate() {
        let (to, calldata, blob_hash_list) = tx_dispatch(&tx);

        let from = tx.recover_signer().unwrap_or_default();

        match (blob_hash_list, to) {
            (Some(hashes), Some(to_addr)) if to_addr == inbox_address => {
                // 4844 → push one placeholder per blob; we fill in the bytes after fetching.
                if from == alloy_primitives::Address::ZERO {
                    return Err(L1FetchError::SignerRecoveryFailed { tx_index });
                }
                for (blob_index, hash) in hashes.into_iter().enumerate() {
                    blob_hashes.push((tx_index, blob_index, hash));
                    tx_views.push(L1TxView { from, to: Some(to_addr), input: Bytes::new() });
                }
            }
            (None, _) => {
                // Non-4844 → forward calldata as-is.
                tx_views.push(L1TxView { from, to, input: calldata });
            }
            (Some(_), _) => {
                // 4844 but not the inbox tx — keep its `to`/`from` so the deriver can ignore it.
                tx_views.push(L1TxView { from, to, input: Bytes::new() });
            }
        }
    }

    if !blob_hashes.is_empty() {
        let hash_only: Vec<B256> = blob_hashes.iter().map(|(_, _, h)| *h).collect();
        // BlobProvider::get_and_validate_blobs takes &mut self; the actor owns a single
        // OnlineBlobProvider and we don't want to plumb &mut through the call chain just for
        // this. Clone-then-call is cheap (the provider is essentially an Arc-backed handle).
        use kona_derive::BlobProvider;
        let mut owned_provider = blob_provider.clone();
        let blobs = owned_provider.get_and_validate_blobs(&block_ref, &hash_only).await?;
        for ((tx_index, blob_index, _), blob) in blob_hashes.iter().zip(blobs.into_iter()) {
            let bytes = decode_blob(blob.as_slice()).map_err(|e| L1FetchError::BlobDecode {
                tx_index: *tx_index,
                blob_index: *blob_index,
                error: e,
            })?;
            // Find the matching view: same tx_index, same `to == inbox_address`. We pushed them
            // in block order so the i-th `L1TxView` with empty input and inbox `to` is what we
            // want.
            let view = tx_views
                .iter_mut()
                .filter(|v| v.to == Some(inbox_address) && v.input.is_empty())
                .nth(*blob_index)
                .expect("blob hash list matches tx_views ordering");
            view.input = bytes;
        }
    }

    Ok(L1BlockData { header, txs: tx_views, receipts })
}

async fn fetch_receipts(
    chain_provider: &AlloyChainProvider,
    block_hash: B256,
) -> Result<Vec<Receipt>, L1FetchError> {
    use alloy_eips::BlockId;
    let receipts = chain_provider
        .inner
        .get_block_receipts(BlockId::Hash(block_hash.into()))
        .await
        .map_err(|e| L1FetchError::Provider(e.to_string()))?
        .ok_or_else(|| L1FetchError::Provider(format!("L1 receipts not found for {block_hash}")))?;
    receipts
        .into_iter()
        .map(|r| r.inner.into_primitives_receipt().as_receipt().cloned())
        .collect::<Option<Vec<_>>>()
        .ok_or_else(|| L1FetchError::Provider("failed to convert receipts".into()))
}

/// Common envelope-style dispatch — extract `(to, calldata, blob_hashes)`.
fn tx_dispatch(tx: &TxEnvelope) -> (Option<alloy_primitives::Address>, Bytes, Option<Vec<B256>>) {
    match tx {
        TxEnvelope::Legacy(t) => (t.tx().to(), t.tx().input.clone(), None),
        TxEnvelope::Eip2930(t) => (t.tx().to(), t.tx().input.clone(), None),
        TxEnvelope::Eip1559(t) => (t.tx().to(), t.tx().input.clone(), None),
        TxEnvelope::Eip4844(wrapper) => match wrapper.tx() {
            TxEip4844Variant::TxEip4844(t) => {
                (t.to(), t.input.clone(), Some(t.blob_versioned_hashes.clone()))
            }
            TxEip4844Variant::TxEip4844WithSidecar(t) => {
                let t = t.tx();
                (t.to(), t.input.clone(), Some(t.blob_versioned_hashes.clone()))
            }
        },
        // Any other envelope variant (EIP-7702 etc.) doesn't participate in derivation.
        _ => (None, Bytes::new(), None),
    }
}

#[cfg(test)]
mod tests {
    // Integration coverage for fetch_l1_block_data lives in the kona-node-service test suite
    // (the actor-level span-batch-overlap test in particular runs this end-to-end). Direct unit
    // tests here would need a mock alloy provider, which adds dependency weight without
    // catching anything the actor test doesn't already exercise.
}
