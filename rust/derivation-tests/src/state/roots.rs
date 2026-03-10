//! State root, storage root, and proof computation using `alloy-trie`.

use alloy_primitives::{keccak256, Address, B256, Bytes, U256, KECCAK256_EMPTY};
use alloy_rlp::Encodable;
use alloy_rpc_types_eth::{EIP1186AccountProofResponse, EIP1186StorageProof};
use alloy_trie::{
    HashBuilder, Nibbles,
    proof::{ProofNodes, ProofRetainer},
};
use op_alloy_consensus::OpTxEnvelope;
use std::collections::{BTreeMap, HashMap};

use super::db::AccountState;

/// Store of trie nodes accumulated during root computation.
///
/// Maps node hash → RLP-encoded node data. Also stores contract code
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

/// RLP-encodable trie account (nonce, balance, `storage_root`, `code_hash`).
#[derive(Debug)]
struct TrieAccount {
    nonce: u64,
    balance: U256,
    storage_root: B256,
    code_hash: B256,
}

impl Encodable for TrieAccount {
    fn encode(&self, out: &mut dyn alloy_rlp::BufMut) {
        let payload_len = self.nonce.length()
            + self.balance.length()
            + self.storage_root.length()
            + self.code_hash.length();
        alloy_rlp::Header {
            list: true,
            payload_length: payload_len,
        }
        .encode(out);
        self.nonce.encode(out);
        self.balance.encode(out);
        self.storage_root.encode(out);
        self.code_hash.encode(out);
    }

    fn length(&self) -> usize {
        let payload_len = self.nonce.length()
            + self.balance.length()
            + self.storage_root.length()
            + self.code_hash.length();
        payload_len + alloy_rlp::length_of_length(payload_len)
    }
}

/// The empty trie root hash (keccak256 of RLP-encoded empty string).
pub(crate) const EMPTY_ROOT_HASH: B256 =
    alloy_primitives::b256!("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421");

/// Compute the state root from account data.
pub fn compute_state_root(
    accounts: &BTreeMap<Address, AccountState>,
    storage_roots: &BTreeMap<Address, B256>,
    _code: &BTreeMap<B256, Vec<u8>>,
    node_store: &mut TrieNodeStore,
) -> B256 {
    if accounts.is_empty() {
        return EMPTY_ROOT_HASH;
    }

    // Sort accounts by keccak256(address) for trie ordering
    let mut sorted: Vec<(Nibbles, Vec<u8>)> = Vec::with_capacity(accounts.len());
    for (address, state) in accounts {
        let hashed_key = Nibbles::unpack(keccak256(address));
        let storage_root = storage_roots.get(address).copied().unwrap_or(EMPTY_ROOT_HASH);
        let trie_account = TrieAccount {
            nonce: state.nonce,
            balance: state.balance,
            storage_root,
            code_hash: state.code_hash,
        };
        let mut encoded = Vec::with_capacity(trie_account.length());
        trie_account.encode(&mut encoded);
        sorted.push((hashed_key, encoded));
    }
    sorted.sort_by(|a, b| a.0.cmp(&b.0));

    let keys: Vec<Nibbles> = sorted.iter().map(|(k, _)| *k).collect();
    let mut hb = HashBuilder::default().with_proof_retainer(ProofRetainer::new(keys));

    for (key, value) in &sorted {
        hb.add_leaf(*key, value);
    }

    let root = hb.root();

    // Store proof nodes
    let proof_nodes = hb.take_proof_nodes();
    store_proof_nodes(&proof_nodes, node_store);

    root
}

/// Compute the storage root for an account's storage.
pub fn compute_storage_root(
    storage: &BTreeMap<U256, U256>,
    node_store: &mut TrieNodeStore,
) -> B256 {
    if storage.is_empty() {
        return EMPTY_ROOT_HASH;
    }

    let mut sorted: Vec<(Nibbles, Vec<u8>)> = Vec::with_capacity(storage.len());
    for (slot, value) in storage {
        if value.is_zero() {
            continue;
        }
        let hashed_key = Nibbles::unpack(keccak256(slot.to_be_bytes::<32>()));
        let mut encoded = Vec::new();
        value.encode(&mut encoded);
        sorted.push((hashed_key, encoded));
    }

    if sorted.is_empty() {
        return EMPTY_ROOT_HASH;
    }

    sorted.sort_by(|a, b| a.0.cmp(&b.0));

    let keys: Vec<Nibbles> = sorted.iter().map(|(k, _)| *k).collect();
    let mut hb = HashBuilder::default().with_proof_retainer(ProofRetainer::new(keys));

    for (key, value) in &sorted {
        hb.add_leaf(*key, value);
    }

    let root = hb.root();
    let proof_nodes = hb.take_proof_nodes();
    store_proof_nodes(&proof_nodes, node_store);

    root
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
    accounts: &BTreeMap<Address, AccountState>,
    storage: &BTreeMap<Address, BTreeMap<U256, U256>>,
    storage_roots: &BTreeMap<Address, B256>,
    address: Address,
    storage_keys: &[B256],
) -> EIP1186AccountProofResponse {
    // Build account trie with proof retainer for the target address
    let target_key = Nibbles::unpack(keccak256(address));

    let mut sorted_accounts: Vec<(Nibbles, Vec<u8>)> = Vec::new();
    for (addr, state) in accounts {
        let hashed_key = Nibbles::unpack(keccak256(addr));
        let storage_root = storage_roots.get(addr).copied().unwrap_or(EMPTY_ROOT_HASH);
        let trie_account = TrieAccount {
            nonce: state.nonce,
            balance: state.balance,
            storage_root,
            code_hash: state.code_hash,
        };
        let mut encoded = Vec::with_capacity(trie_account.length());
        trie_account.encode(&mut encoded);
        sorted_accounts.push((hashed_key, encoded));
    }
    sorted_accounts.sort_by(|a, b| a.0.cmp(&b.0));

    let mut hb =
        HashBuilder::default().with_proof_retainer(ProofRetainer::new(vec![target_key]));
    for (key, value) in &sorted_accounts {
        hb.add_leaf(*key, value);
    }
    let _root = hb.root();
    let proof_nodes = hb.take_proof_nodes();
    let account_proof = proof_nodes_to_vec(&proof_nodes, &target_key);

    let account_state = accounts.get(&address);
    let account_storage = storage.get(&address);
    let storage_root = storage_roots.get(&address).copied().unwrap_or(EMPTY_ROOT_HASH);

    // Generate storage proofs
    let storage_proofs: Vec<EIP1186StorageProof> = storage_keys
        .iter()
        .map(|key| {
            let slot = U256::from_be_bytes(key.0);
            let value = account_storage
                .and_then(|s| s.get(&slot))
                .copied()
                .unwrap_or(U256::ZERO);

            let hashed_key = Nibbles::unpack(keccak256(key));

            // Build storage trie for this proof
            let proof = account_storage.map_or_else(Vec::new, |acct_storage| {
                let mut sorted_storage: Vec<(Nibbles, Vec<u8>)> = Vec::new();
                for (s, v) in acct_storage {
                    if v.is_zero() {
                        continue;
                    }
                    let k = Nibbles::unpack(keccak256(s.to_be_bytes::<32>()));
                    let mut encoded = Vec::new();
                    v.encode(&mut encoded);
                    sorted_storage.push((k, encoded));
                }
                sorted_storage.sort_by(|a, b| a.0.cmp(&b.0));

                let mut shb = HashBuilder::default()
                    .with_proof_retainer(ProofRetainer::new(vec![hashed_key]));
                for (k, v) in &sorted_storage {
                    shb.add_leaf(*k, v);
                }
                let _ = shb.root();
                let storage_proof_nodes = shb.take_proof_nodes();
                proof_nodes_to_vec(&storage_proof_nodes, &hashed_key)
            });

            EIP1186StorageProof {
                key: alloy_serde::storage::JsonStorageKey::Hash(*key),
                value,
                proof,
            }
        })
        .collect();

    EIP1186AccountProofResponse {
        address,
        balance: account_state.map(|a| a.balance).unwrap_or(U256::ZERO),
        code_hash: account_state
            .map(|a| a.code_hash)
            .unwrap_or(KECCAK256_EMPTY),
        nonce: account_state.map(|a| a.nonce).unwrap_or(0),
        storage_hash: storage_root,
        account_proof,
        storage_proof: storage_proofs,
    }
}

/// Convert proof nodes to a Vec<Bytes> for the target key path.
fn proof_nodes_to_vec(proof_nodes: &ProofNodes, _target: &Nibbles) -> Vec<Bytes> {
    // ProofNodes contains all nodes along the path. We return them sorted.
    let nodes = proof_nodes.nodes_sorted();
    nodes.into_iter().map(|(_, v)| v).collect()
}

/// Store proof nodes from a `HashBuilder` into the `TrieNodeStore`.
fn store_proof_nodes(proof_nodes: &ProofNodes, node_store: &mut TrieNodeStore) {
    for (_, node_bytes) in proof_nodes.nodes_sorted() {
        let hash = keccak256(&node_bytes);
        node_store.insert(hash, node_bytes);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::state::TestStateDb;
    use alloy_genesis::GenesisAccount;
    use alloy_primitives::address;

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
    fn storage_root_ordering_deterministic() {
        let mut store = TrieNodeStore::new();

        let mut storage1 = BTreeMap::new();
        storage1.insert(U256::from(1u64), U256::from(100u64));
        storage1.insert(U256::from(2u64), U256::from(200u64));
        let root1 = compute_storage_root(&storage1, &mut store);

        let mut storage2 = BTreeMap::new();
        storage2.insert(U256::from(2u64), U256::from(200u64));
        storage2.insert(U256::from(1u64), U256::from(100u64));
        let root2 = compute_storage_root(&storage2, &mut store);

        assert_eq!(root1, root2);
        assert_ne!(root1, EMPTY_ROOT_HASH);
    }
}
