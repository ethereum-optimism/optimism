//! Output root and super root computation.
//!
//! Thin wrappers around existing kona-protocol implementations.

use alloy_primitives::B256;
use kona_interop::{OutputRootWithChain, SuperRoot};
use kona_protocol::OutputRoot;

use crate::config::L2_TO_L1_MESSAGE_PASSER;
use crate::l2::L2Block;
use crate::state::StateSnapshot;

/// Compute the output root (V0) for an L2 block.
///
/// `message_passer_storage_root` is the storage root of the `L2ToL1MessagePasser` contract.
pub fn compute_output_root(block: &L2Block, message_passer_storage_root: B256) -> B256 {
    OutputRoot::from_parts(
        block.header.inner().state_root,
        message_passer_storage_root,
        block.header.hash(),
    )
    .hash()
}

/// Compute the super root (V1) for a set of chains.
///
/// Chains are sorted by chain ID internally.
pub fn compute_super_root(timestamp: u64, chains: Vec<(u64, B256)>) -> B256 {
    let output_roots: Vec<OutputRootWithChain> = chains
        .into_iter()
        .map(|(chain_id, output_root)| OutputRootWithChain::new(chain_id, output_root))
        .collect();
    SuperRoot::new(timestamp, output_roots).hash()
}

/// Compute the output root from a state snapshot, extracting the `L2ToL1MessagePasser` storage root.
pub fn compute_output_root_from_state(block: &L2Block, snapshot: &StateSnapshot) -> B256 {
    use crate::state::roots::{EMPTY_ROOT_HASH, TrieNodeStore, compute_storage_root};

    let message_passer_storage_root = snapshot
        .storage
        .get(&L2_TO_L1_MESSAGE_PASSER)
        .map(|storage| {
            let mut node_store = TrieNodeStore::new();
            compute_storage_root(storage, &mut node_store)
        })
        .unwrap_or(EMPTY_ROOT_HASH);

    compute_output_root(block, message_passer_storage_root)
}

/// Compute a super root for a single chain (pre-interop convenience).
pub fn compute_single_chain_super_root(timestamp: u64, chain_id: u64, output_root: B256) -> B256 {
    compute_super_root(timestamp, vec![(chain_id, output_root)])
}
