//! State root, storage root, and proof computation using reth's in-memory trie infrastructure.

use alloy_primitives::{Address, B256, Bytes, keccak256};
use alloy_rpc_types_eth::EIP1186AccountProofResponse;
use op_alloy_consensus::OpTxEnvelope;
use reth_trie::{
    HashedPostState, StorageRoot,
    hashed_cursor::{HashedPostStateCursorFactory, noop::NoopHashedCursorFactory},
    proof::Proof,
    trie_cursor::noop::NoopTrieCursorFactory,
};
use std::collections::HashMap;

pub(crate) use reth_trie::EMPTY_ROOT_HASH;

/// Store of trie nodes accumulated during root computation.
///
/// Maps node hash -> RLP-encoded node data. Also stores contract code
/// with a `0x63<code_hash>` prefix key (matching kona-host's `debug_dbGet` convention).
#[derive(Debug, Clone, Default)]
pub struct TrieNodeStore {
    /// Trie nodes keyed by hash.
    pub nodes: HashMap<B256, Bytes>,
}

impl TrieNodeStore {
    /// Create an empty node store.
    pub fn new() -> Self {
        Self::default()
    }

    /// Insert a node.
    pub fn insert(&mut self, hash: B256, data: Bytes) {
        self.nodes.insert(hash, data);
    }

    /// Look up a node by hash.
    pub fn get(&self, hash: &B256) -> Option<&Bytes> {
        self.nodes.get(hash)
    }
}

/// Compute the transactions trie root.
pub fn compute_transactions_root(txs: &[OpTxEnvelope]) -> B256 {
    if txs.is_empty() {
        return EMPTY_ROOT_HASH;
    }
    use alloy_eips::Encodable2718;
    kona_mpt::ordered_trie_with_encoder(txs, |tx, buf| tx.encode_2718(buf)).root()
}

/// Compute the receipts trie root.
pub fn compute_receipts_root(receipts: &[op_alloy_consensus::OpReceiptEnvelope]) -> B256 {
    if receipts.is_empty() {
        return EMPTY_ROOT_HASH;
    }
    use alloy_eips::Encodable2718;
    kona_mpt::ordered_trie_with_encoder(receipts, |r, buf| r.encode_2718(buf)).root()
}

/// Generate an EIP-1186 account proof response for a given address.
pub fn generate_account_proof(
    hashed_state: &HashedPostState,
    address: Address,
    storage_keys: &[B256],
) -> EIP1186AccountProofResponse {
    let sorted = hashed_state.clone().into_sorted();
    let prefix_sets = hashed_state.construct_prefix_sets();

    Proof::new(
        NoopTrieCursorFactory::default(),
        HashedPostStateCursorFactory::new(NoopHashedCursorFactory::default(), &sorted),
    )
    .with_prefix_sets_mut(prefix_sets)
    .account_proof(address, storage_keys)
    .expect("proof generation should succeed")
    .into_eip1186_response(
        storage_keys.iter().map(|k| alloy_serde::storage::JsonStorageKey::Hash(*k)).collect(),
    )
}

/// Compute the storage root for a single account from the accumulated hashed state.
pub fn storage_root_from_hashed_state(hashed_state: &HashedPostState, address: Address) -> B256 {
    let sorted = hashed_state.clone().into_sorted();
    let hashed_address = keccak256(address);

    let prefix_set = hashed_state
        .construct_prefix_sets()
        .storage_prefix_sets
        .remove(&hashed_address)
        .unwrap_or_default()
        .freeze();

    StorageRoot::new_hashed(
        NoopTrieCursorFactory::default(),
        HashedPostStateCursorFactory::new(NoopHashedCursorFactory::default(), &sorted),
        hashed_address,
        prefix_set,
    )
    .root()
    .expect("storage root computation should succeed")
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::state::TestStateDb;
    use alloy_genesis::GenesisAccount;
    use alloy_primitives::{U256, address};
    use std::collections::BTreeMap;

    #[test]
    fn empty_state_root() {
        let db = TestStateDb::new();
        let snapshot = db.snapshot();
        assert_eq!(snapshot.state_root, EMPTY_ROOT_HASH);
    }

    #[test]
    fn genesis_state_root_is_deterministic() {
        let mut allocs = BTreeMap::new();
        allocs.insert(
            address!("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
            GenesisAccount::default().with_balance(U256::from(1000u64)),
        );

        let mut db1 = TestStateDb::new();
        db1.init_genesis(&allocs);
        let snap1 = db1.snapshot();

        let mut db2 = TestStateDb::new();
        db2.init_genesis(&allocs);
        let snap2 = db2.snapshot();

        assert_eq!(snap1.state_root, snap2.state_root);
        assert_ne!(snap1.state_root, EMPTY_ROOT_HASH);
    }

    #[test]
    fn state_root_after_transfer() {
        let config = crate::config::DeterministicConfig::default();
        let mut l2 = crate::l2::L2ChainBuilder::new(&config);

        let genesis_root = l2.head_snapshot().state_root;
        assert_ne!(genesis_root, EMPTY_ROOT_HASH, "genesis state root should not be empty");

        // Build an L1 block to serve as epoch
        let mut l1 = crate::l1::L1ChainBuilder::new(&config);
        l1.emit_empty_block();
        let l1_block = l1.block_at(1).unwrap().clone();
        l2.set_epoch(&l1_block);

        // Build a block with the L1 info deposit tx (modifies state)
        l2.build_empty_block().unwrap();

        let post_root = l2.head_snapshot().state_root;
        assert_ne!(post_root, genesis_root, "state root should change after executing deposit tx");
        assert_ne!(post_root, EMPTY_ROOT_HASH);
    }

    #[test]
    fn storage_root_ordering_deterministic() {
        let addr = address!("0x0000000000000000000000000000000000001234");

        let mut allocs1 = BTreeMap::new();
        let mut storage1 = BTreeMap::new();
        storage1.insert(B256::from(U256::from(1u64)), B256::from(U256::from(100u64)));
        storage1.insert(B256::from(U256::from(2u64)), B256::from(U256::from(200u64)));
        allocs1.insert(
            addr,
            GenesisAccount::default().with_balance(U256::from(1u64)).with_storage(Some(storage1)),
        );

        let mut db1 = TestStateDb::new();
        db1.init_genesis(&allocs1);
        let root1 = storage_root_from_hashed_state(db1.hashed_state(), addr);

        let mut allocs2 = BTreeMap::new();
        let mut storage2 = BTreeMap::new();
        storage2.insert(B256::from(U256::from(2u64)), B256::from(U256::from(200u64)));
        storage2.insert(B256::from(U256::from(1u64)), B256::from(U256::from(100u64)));
        allocs2.insert(
            addr,
            GenesisAccount::default().with_balance(U256::from(1u64)).with_storage(Some(storage2)),
        );

        let mut db2 = TestStateDb::new();
        db2.init_genesis(&allocs2);
        let root2 = storage_root_from_hashed_state(db2.hashed_state(), addr);

        assert_eq!(root1, root2);
        assert_ne!(root1, EMPTY_ROOT_HASH);
    }
}
