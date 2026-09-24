//! L1 authentication from the claim's `l1Head` (spec-sound-profile §D.2.1), and the
//! ordered-list trie helper the host and fixtures use to ship L1 receipts as preimages.

use super::providers::Store;
use alloy_primitives::{B256, Bytes};
use anyhow::{Result, anyhow, ensure};
use kona_mpt::ordered_trie_with_encoder;
use std::collections::BTreeMap;

/// Walk L1 headers from `l1_head` down to `min_epoch` by parent hash, returning the canonical
/// hash of every L1 number in `min_epoch..=head.number`. Every header is read by its hash, so the
/// whole map is a consequence of `l1_head` alone.
pub(crate) fn epoch_hashes(
    store: Store<'_>,
    l1_head: B256,
    min_epoch: u64,
) -> Result<BTreeMap<u64, B256>> {
    let mut out = BTreeMap::new();
    let mut hash = l1_head;
    let mut expected: Option<u64> = None;
    loop {
        let header = store.header(hash).map_err(|e| anyhow!("L1 header chain from l1Head: {e}"))?;
        if let Some(n) = expected {
            ensure!(header.number == n, "L1 header chain from l1Head: parent number");
        }
        ensure!(header.number >= min_epoch, "L1 head precedes the span's epochs");
        out.insert(header.number, hash);
        if header.number == min_epoch {
            return Ok(out);
        }
        expected = Some(header.number - 1);
        hash = header.parent_hash;
    }
}

/// The root and every node of the ordered-list trie over `items` (EIP-2718 receipt or transaction
/// encodings), as `keccak256(node) -> node` preimages accepted by the relation's `Store`.
pub fn ordered_list_preimages(items: &[Bytes]) -> (B256, Vec<Bytes>) {
    let mut hb = ordered_trie_with_encoder(items, |item, out| out.put_slice(item));
    let root = hb.root();
    let nodes = hb.take_proof_nodes().into_nodes_sorted().into_iter().map(|(_, node)| node);
    (root, nodes.collect())
}
