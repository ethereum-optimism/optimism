//! Per-block log commitments, computed by the `log-commitment` program.
//!
//! An entry `(r, L, R)` commits to the logs under receipts root `r`. The `L` logs are numbered
//! across the block: receipts in transaction order, logs in receipt order. `R` is the root of a
//! binary Merkle tree with depth `ceil(log2 L)` (0 for one leaf) over the leaves
//! `keccak256(address ‖ keccak256(topics ‖ data))`, padded with zero leaves, with nodes
//! `keccak256(left ‖ right)`. `L = 0` gives `R = 0`.

use alloy_consensus::EMPTY_ROOT_HASH;
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::{Address, B256, Keccak256, Log, keccak256};
use alloy_rlp::EMPTY_STRING_CODE;
use anyhow::{Result, ensure};
use kona_preimage::PreimageKey;
use kona_proof_interop::receipts_from_trie;
use op_alloy_consensus::OpReceiptEnvelope;

use crate::witness::{
    BlobData, DefaultWitnessData, WitnessData,
    preimage_store::{PreimageStore, WitnessOracle},
};

#[cfg(any(test, feature = "log-commitment-test-utils"))]
pub mod test_utils;

/// Length of an encoded [`LogCommitmentEntry`].
pub const LOG_COMMITMENT_ENTRY_LEN: usize = 72;

/// The commitment to the logs under one receipts root.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct LogCommitmentEntry {
    /// The receipts root.
    pub receipts_root: B256,
    /// The number of logs under the receipts root.
    pub log_count: u64,
    /// The Merkle root over the log leaves.
    pub log_root: B256,
}

impl LogCommitmentEntry {
    /// Encodes the entry as `receipts_root ‖ log_count (big-endian) ‖ log_root`.
    pub fn encode(&self) -> [u8; LOG_COMMITMENT_ENTRY_LEN] {
        let mut out = [0u8; LOG_COMMITMENT_ENTRY_LEN];
        out[..32].copy_from_slice(self.receipts_root.as_slice());
        out[32..40].copy_from_slice(&self.log_count.to_be_bytes());
        out[40..].copy_from_slice(self.log_root.as_slice());
        out
    }
}

/// The leaf of a log with `address` and `payload_hash = keccak256(topics ‖ data)`. Equals
/// `logHash` in `CrossL2Inbox.calculateChecksum`.
pub fn log_leaf(address: &Address, payload_hash: &B256) -> B256 {
    let mut preimage = [0u8; 52];
    preimage[..20].copy_from_slice(address.as_slice());
    preimage[20..].copy_from_slice(payload_hash.as_slice());
    keccak256(preimage)
}

fn payload_hash(log: &Log) -> B256 {
    let mut hasher = Keccak256::new();
    for topic in log.topics() {
        hasher.update(topic);
    }
    hasher.update(&log.data.data);
    hasher.finalize()
}

fn node(left: &B256, right: &B256) -> B256 {
    let mut preimage = [0u8; 64];
    preimage[..32].copy_from_slice(left.as_slice());
    preimage[32..].copy_from_slice(right.as_slice());
    keccak256(preimage)
}

/// The Merkle root over `leaves`, in the format of the module docs.
pub fn log_root(mut leaves: Vec<B256>) -> B256 {
    // Root of an all-zero subtree with the height of the current level.
    let mut zero = B256::ZERO;
    while leaves.len() > 1 {
        let parents = leaves.len().div_ceil(2);
        for i in 0..parents {
            let right = leaves.get(2 * i + 1).copied().unwrap_or(zero);
            leaves[i] = node(&leaves[2 * i], &right);
        }
        leaves.truncate(parents);
        zero = node(&zero, &zero);
    }
    leaves.first().copied().unwrap_or(B256::ZERO)
}

/// Commits to the logs under `receipts_root`.
///
/// A receipts trie that does not decode commits as `(receipts_root, 0, 0)`: the root fixes its
/// bytes, and the interop message check rejects every message into such a block. A trie node
/// missing from the witness panics in [`WitnessOracle`] instead, so the prover cannot choose
/// `L = 0` by omitting one.
pub fn commit_receipts_root(oracle: &WitnessOracle, receipts_root: B256) -> LogCommitmentEntry {
    let Ok(receipts) = receipts_from_trie(oracle, receipts_root) else {
        return LogCommitmentEntry { receipts_root, log_count: 0, log_root: B256::ZERO };
    };
    let leaves = log_leaves(&receipts);
    LogCommitmentEntry { receipts_root, log_count: leaves.len() as u64, log_root: log_root(leaves) }
}

fn log_leaves(receipts: &[OpReceiptEnvelope]) -> Vec<B256> {
    receipts
        .iter()
        .flat_map(OpReceiptEnvelope::logs)
        .map(|log| log_leaf(&log.address, &payload_hash(log)))
        .collect()
}

/// Runs the `log-commitment` program: the encoded entry of each receipts root, in input order.
///
/// Panics if the witness does not decode, holds a preimage that does not match its key, or lacks a
/// receipts trie node (see [`commit_receipts_root`]).
pub fn log_commitment_public_values(receipts_roots: &[B256], witness_rkyv: &[u8]) -> Vec<u8> {
    let witness = rkyv::from_bytes::<DefaultWitnessData, rkyv::rancor::Error>(witness_rkyv)
        .expect("log-commitment witness decodes");
    let (oracle, _) = kona_proof::block_on(witness.get_oracle_and_blob_provider())
        .expect("log-commitment witness preimages verify");
    receipts_roots.iter().flat_map(|root| commit_receipts_root(&oracle, *root).encode()).collect()
}

/// Builds the `log-commitment` witness for blocks given as `(receipts root, receipts)`.
///
/// Fails if the receipts of a block do not hash to its receipts root.
pub fn log_commitment_witness(
    blocks: &[(B256, Vec<OpReceiptEnvelope>)],
) -> Result<DefaultWitnessData> {
    let mut store = PreimageStore::default();
    for (index, (receipts_root, receipts)) in blocks.iter().enumerate() {
        let mut trie = kona_mpt::ordered_trie_with_encoder(receipts, |receipt, out| {
            receipt.encode_2718(out);
        });
        let root = trie.root();
        ensure!(
            root == *receipts_root,
            "block {index}: receipts hash to {root}, expected receipts root {receipts_root}"
        );
        // The trie of an empty list retains no nodes; its root is the empty-string node.
        if receipts.is_empty() {
            store.save_preimage(
                PreimageKey::new_keccak256(*EMPTY_ROOT_HASH),
                vec![EMPTY_STRING_CODE],
            )?;
        }
        for node in trie.take_proof_nodes().into_inner().into_values() {
            store.save_preimage(PreimageKey::new_keccak256(*keccak256(&node)), node.to_vec())?;
        }
    }
    Ok(DefaultWitnessData::from_parts(store, BlobData::default()))
}

#[cfg(test)]
mod tests {
    use std::sync::Arc;

    use alloy_consensus::Header;
    use alloy_primitives::hex;
    use alloy_rlp::Encodable;
    use kona_interop::{DependencySet, InteropProvider, RawMessagePayload, SuperRoot};
    use kona_proof::errors::OracleProviderError;
    use kona_proof_interop::{BootInfo, OracleInteropProvider, PreState};
    use rand::Rng;

    use super::{
        test_utils::{
            deposit, eip1559, logs, missing_trie_node_witness, mixed_blocks, receipts_root,
        },
        *,
    };

    const CHAIN_ID: u64 = 10;

    fn witness_bytes(witness: &DefaultWitnessData) -> Vec<u8> {
        rkyv::to_bytes::<rkyv::rancor::Error>(witness).unwrap().to_vec()
    }

    fn oracle(witness: DefaultWitnessData) -> Arc<WitnessOracle> {
        kona_proof::block_on(witness.get_oracle_and_blob_provider()).unwrap().0
    }

    fn save(store: &mut PreimageStore, preimage: Vec<u8>) -> B256 {
        let hash = keccak256(&preimage);
        store.save_preimage(PreimageKey::new_keccak256(*hash), preimage).unwrap();
        hash
    }

    /// Saves a header with `receipts_root` and returns its block hash.
    fn save_header(store: &mut PreimageStore, number: u64, receipts_root: B256) -> B256 {
        let header = Header { number, receipts_root, ..Default::default() };
        let mut rlp = Vec::new();
        header.encode(&mut rlp);
        save(store, rlp)
    }

    /// A fully padded tree, without the zero-subtree shortcut.
    fn padded_root(leaves: &[B256]) -> B256 {
        if leaves.is_empty() {
            return B256::ZERO;
        }
        let mut level = leaves.to_vec();
        level.resize(leaves.len().next_power_of_two(), B256::ZERO);
        while level.len() > 1 {
            level = level.chunks(2).map(|pair| keccak256([pair[0], pair[1]].concat())).collect();
        }
        level[0]
    }

    /// The receipts that `OracleInteropProvider::receipts_by_hash` returns for `block_hash`.
    fn interop_provider_receipts(
        oracle: Arc<WitnessOracle>,
        block_hash: B256,
    ) -> Result<Vec<OpReceiptEnvelope>, OracleProviderError> {
        let boot = BootInfo {
            l1_head: B256::ZERO,
            agreed_pre_state_commitment: B256::ZERO,
            agreed_pre_state: PreState::SuperRoot(SuperRoot::new(0, Vec::new())),
            claimed_post_state: B256::ZERO,
            claimed_l2_timestamp: 0,
            rollup_configs: Default::default(),
            dependency_set: DependencySet {
                dependencies: Default::default(),
                override_message_expiry_window: None,
            },
            l1_config: Default::default(),
        };
        let provider = OracleInteropProvider::new(oracle, boot, Default::default());
        kona_proof::block_on(provider.receipts_by_hash(CHAIN_ID, block_hash))
    }

    /// The leaves of the log list that `MessageGraph` indexes for the block with `block_hash`.
    fn interop_provider_leaves(oracle: Arc<WitnessOracle>, block_hash: B256) -> Vec<B256> {
        interop_provider_receipts(oracle, block_hash)
            .unwrap()
            .into_iter()
            .flat_map(|receipt| receipt.into_logs())
            .map(|log| {
                let payload_hash = keccak256(RawMessagePayload::from(&log).as_ref());
                keccak256([log.address.as_slice(), payload_hash.as_slice()].concat())
            })
            .collect()
    }

    /// Pins `log_root` to a fully padded reference tree.
    #[test]
    fn tree_matches_padded_reference() {
        let mut rng = rand::rng();
        let large = rng.random_range(1_000..5_000);
        for len in [0, 1, 2, 3, 4, 5, 7, 8, 9, large] {
            let leaves: Vec<B256> =
                (0..len).map(|_| B256::from(rng.random::<[u8; 32]>())).collect();
            assert_eq!(log_root(leaves.clone()), padded_root(&leaves), "L = {len}");
        }
    }

    /// Pins each entry to the log list from `OracleInteropProvider::receipts_by_hash`.
    #[test]
    fn entries_match_interop_provider_receipts() {
        let blocks = mixed_blocks();
        let (mut store, blobs) = log_commitment_witness(&blocks).unwrap().into_parts();
        let block_hashes: Vec<B256> = blocks
            .iter()
            .zip(0..)
            .map(|((receipts_root, _), number)| save_header(&mut store, number, *receipts_root))
            .collect();
        let oracle = oracle(DefaultWitnessData::from_parts(store, blobs));

        for ((receipts_root, input_receipts), block_hash) in blocks.iter().zip(block_hashes) {
            let leaves = interop_provider_leaves(oracle.clone(), block_hash);
            assert_eq!(log_leaves(input_receipts), leaves);
            assert_eq!(
                commit_receipts_root(&oracle, *receipts_root),
                LogCommitmentEntry {
                    receipts_root: *receipts_root,
                    log_count: leaves.len() as u64,
                    log_root: padded_root(&leaves),
                },
            );
        }
    }

    /// Witness and root of a receipts trie whose leaves hold `values` verbatim.
    fn raw_trie(values: &[Vec<u8>]) -> (B256, PreimageStore) {
        let mut trie =
            kona_mpt::ordered_trie_with_encoder(values, |value, out| out.put_slice(value));
        let root = trie.root();
        let mut store = PreimageStore::default();
        for node in trie.take_proof_nodes().into_inner().into_values() {
            save(&mut store, node.to_vec());
        }
        (root, store)
    }

    /// Asserts `(r, 0, 0)` and an `OracleInteropProvider` error for a header with `r`.
    fn assert_commits_empty(receipts_root: B256, mut store: PreimageStore) {
        let block_hash = save_header(&mut store, 0, receipts_root);
        let oracle = oracle(DefaultWitnessData::from_parts(store, BlobData::default()));
        assert_eq!(
            commit_receipts_root(&oracle, receipts_root),
            LogCommitmentEntry { receipts_root, log_count: 0, log_root: B256::ZERO },
        );
        let receipts = interop_provider_receipts(oracle, block_hash);
        assert!(receipts.is_err(), "the interop provider accepts the trie: {receipts:?}");
    }

    /// Pins `(r, 0, 0)` and a provider error for a receipts trie leaf that is not a receipt.
    #[test]
    fn invalid_receipt_leaf_commits_empty() {
        let mut valid = Vec::new();
        eip1559(logs(0, 2)).encode_2718(&mut valid);
        // A blob receipt: valid on L1, not an OP receipt type.
        let (root, store) = raw_trie(&[valid, vec![0x03, 0xc0]]);
        assert_commits_empty(root, store);
    }

    /// Pins `(r, 0, 0)` and a provider error for a trie node that does not decode.
    #[test]
    fn undecodable_trie_node_commits_empty() {
        let mut store = PreimageStore::default();
        // A three-item list is neither a branch nor a leaf or extension node.
        let root = save(&mut store, vec![0xc3, 0x80, 0x80, 0x80]);
        assert_commits_empty(root, store);
    }

    /// Pins a halt, not `L = 0`, when the witness omits a trie node.
    #[test]
    #[should_panic(expected = "requested preimage key not present in witness")]
    fn missing_trie_node_panics() {
        let (root, witness) = missing_trie_node_witness();
        log_commitment_public_values(&[root], &witness_bytes(&witness));
    }

    /// Pins a halt when a witness preimage does not match its key.
    #[test]
    #[should_panic(expected = "Failed to validate preimages")]
    fn forged_trie_node_panics() {
        let receipts = vec![eip1559(logs(0, 3))];
        let root = receipts_root(&receipts);
        let witness = log_commitment_witness(&[(root, receipts.clone())]).unwrap();
        let mut bytes = witness_bytes(&witness);
        // The single trie node is a leaf holding the encoded receipt; change one of its bytes.
        let mut receipt = Vec::new();
        receipts[0].encode_2718(&mut receipt);
        let at = bytes.windows(receipt.len()).position(|window| window == receipt).unwrap();
        bytes[at + receipt.len() - 1] ^= 1;

        log_commitment_public_values(&[root], &bytes);
    }

    /// Pins the public-value bytes of a batch with a duplicate root.
    #[test]
    fn public_values_golden_bytes() {
        let receipts = vec![deposit(vec![], Some((1, 1))), eip1559(logs(1, 2))];
        let root = receipts_root(&receipts);
        let blocks = vec![(root, receipts.clone()), (EMPTY_ROOT_HASH, vec![]), (root, receipts)];
        let roots: Vec<B256> = blocks.iter().map(|(root, _)| *root).collect();
        let witness = log_commitment_witness(&blocks).unwrap();

        let public_values = log_commitment_public_values(&roots, &witness_bytes(&witness));
        assert_eq!(public_values.len(), 3 * LOG_COMMITMENT_ENTRY_LEN);
        let entry = concat!(
            "dfa31e400bc7cffb5e5f337c228b49d65bc9d5bffa293c8e9940b95acbbd8af7",
            "0000000000000002",
            "34898a6d10756ec6df465094d8a76380282a257994e446a51308f44319fbc265",
        );
        let empty = concat!(
            "56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
            "0000000000000000",
            "0000000000000000000000000000000000000000000000000000000000000000",
        );
        assert_eq!(hex::encode(public_values), [entry, empty, entry].concat());
    }

    /// Pins the witness builder's receipts-root check.
    #[test]
    fn witness_rejects_wrong_receipts_root() {
        let blocks = vec![(B256::repeat_byte(1), vec![eip1559(logs(0, 1))])];
        let err = log_commitment_witness(&blocks).unwrap_err();
        assert!(err.to_string().starts_with("block 0: receipts hash to"), "{err}");
    }
}
