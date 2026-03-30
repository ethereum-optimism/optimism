//! Output root and super root computation.
//!
//! Thin wrappers around existing kona-protocol implementations.

use alloy_primitives::B256;
use kona_interop::{OutputRootWithChain, SuperRoot};
use kona_protocol::OutputRoot;

use crate::{config::L2_TO_L1_MESSAGE_PASSER, l2::L2Block, state::StateSnapshot};

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

/// Compute the output root from a state snapshot, extracting the `L2ToL1MessagePasser` storage
/// root.
pub fn compute_output_root_from_state(block: &L2Block, snapshot: &StateSnapshot) -> B256 {
    use crate::state::storage_root_from_hashed_state;

    let message_passer_storage_root =
        storage_root_from_hashed_state(&snapshot.hashed_state, L2_TO_L1_MESSAGE_PASSER);

    compute_output_root(block, message_passer_storage_root)
}

/// Compute a super root for a single chain (pre-interop convenience).
pub fn compute_single_chain_super_root(timestamp: u64, chain_id: u64, output_root: B256) -> B256 {
    compute_super_root(timestamp, vec![(chain_id, output_root)])
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_output_root_deterministic() {
        let state_root = B256::from([0x11; 32]);
        let storage_root = B256::from([0x22; 32]);
        let block_hash = B256::from([0x33; 32]);

        let root1 = OutputRoot::from_parts(state_root, storage_root, block_hash).hash();
        let root2 = OutputRoot::from_parts(state_root, storage_root, block_hash).hash();

        assert_ne!(root1, B256::ZERO, "output root should not be zero");
        assert_eq!(root1, root2, "output root should be deterministic");
    }

    #[test]
    fn test_super_root_chain_ordering() {
        let timestamp = 1_700_000_000u64;
        let chain_a = (10u64, B256::from([0xAA; 32]));
        let chain_b = (20u64, B256::from([0xBB; 32]));

        // Two different orderings should produce the same super root
        // because SuperRoot sorts chains by chain ID internally.
        let root_ab = compute_super_root(timestamp, vec![chain_a, chain_b]);
        let root_ba = compute_super_root(timestamp, vec![chain_b, chain_a]);

        assert_eq!(root_ab, root_ba, "chain ordering should not affect super root");
    }

    #[test]
    fn test_single_chain_super_root_matches_manual() {
        let timestamp = 1_700_000_000u64;
        let chain_id = 901u64;
        let output_root = B256::from([0xCC; 32]);

        let convenience = compute_single_chain_super_root(timestamp, chain_id, output_root);
        let manual = compute_super_root(timestamp, vec![(chain_id, output_root)]);

        assert_eq!(
            convenience, manual,
            "single chain convenience function should match explicit computation"
        );
    }
}
