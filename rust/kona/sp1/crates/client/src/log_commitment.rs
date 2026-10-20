//! Per-block log commitments, computed by the `log-commitment` program.
//!
//! An entry `(r, L, R)` commits to the logs under receipts root `r`. The `L` logs are numbered
//! across the block: receipts in transaction order, logs in receipt order. `R` is the root of a
//! binary Merkle tree with depth `ceil(log2 L)` (0 for one leaf) over the leaves
//! `keccak256(address ‖ keccak256(topics ‖ data))`, padded with zero leaves, with nodes
//! `keccak256(left ‖ right)`. `L = 0` gives `R = 0`.
//!
//! The SP1 consolidation checks a message into a block below the local safe head against the
//! entry of the block's receipts root and an opening of the message's log, through
//! [`VerifiedLogCommitments`].

use std::{
    collections::{BTreeMap, BTreeSet},
    sync::Arc,
};

use alloy_consensus::{EMPTY_ROOT_HASH, Header};
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::{Address, B256, Keccak256, Log, keccak256};
use alloy_rlp::EMPTY_STRING_CODE;
use anyhow::{Result, ensure};
use kona_interop::LogCommitment;
use kona_preimage::PreimageKey;
use kona_proof_interop::{LogCommitmentSource, receipts_from_trie};
use op_alloy_consensus::OpReceiptEnvelope;
use sha2::{Digest, Sha256};

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

    fn decode(bytes: &[u8; LOG_COMMITMENT_ENTRY_LEN]) -> Self {
        Self {
            receipts_root: B256::from_slice(&bytes[..32]),
            log_count: u64::from_be_bytes(bytes[32..40].try_into().expect("8 bytes")),
            log_root: B256::from_slice(&bytes[40..]),
        }
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
pub fn log_root(leaves: Vec<B256>) -> B256 {
    reduce_tree(leaves, |_, _| {})
}

/// The tree of [`log_root`] with every level kept, to open any log.
#[derive(Debug, Clone)]
pub struct LogTree {
    leaves: Vec<B256>,
    levels: Vec<TreeLevel>,
    root: B256,
}

/// A level below the root of a [`LogTree`].
#[derive(Debug, Clone)]
struct TreeLevel {
    nodes: Vec<B256>,
    /// The root of an all-zero subtree of the level's height.
    zero: B256,
}

impl LogTree {
    /// Builds the tree over `leaves`.
    pub fn new(leaves: Vec<B256>) -> Self {
        let mut levels = Vec::new();
        let root = reduce_tree(leaves.clone(), |nodes, zero| {
            levels.push(TreeLevel { nodes: nodes.to_vec(), zero: *zero });
        });
        Self { leaves, levels, root }
    }

    /// The root, equal to [`log_root`] over the same leaves.
    pub const fn root(&self) -> B256 {
        self.root
    }

    /// The number of leaves.
    pub const fn log_count(&self) -> u64 {
        self.leaves.len() as u64
    }

    /// The opening of log `index`, below [`Self::log_count`].
    pub fn opening(&self, receipts_root: B256, index: u64) -> LogOpening {
        let position = index as usize;
        let siblings = self
            .levels
            .iter()
            .enumerate()
            .map(|(height, level)| {
                level.nodes.get((position >> height) ^ 1).copied().unwrap_or(level.zero)
            })
            .collect();
        LogOpening { receipts_root, index, leaf: self.leaves[position], siblings }
    }
}

/// Hashes `leaves` up to the root, calling `visit` with each level below the root and the root of
/// an all-zero subtree of that level's height.
fn reduce_tree(mut leaves: Vec<B256>, mut visit: impl FnMut(&[B256], &B256)) -> B256 {
    let mut zero = B256::ZERO;
    while leaves.len() > 1 {
        visit(&leaves, &zero);
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

/// The depth of the tree over `log_count` leaves: `ceil(log2 log_count)`, and 0 for one leaf.
const fn log_tree_depth(log_count: u64) -> usize {
    if log_count <= 1 { 0 } else { (u64::BITS - (log_count - 1).leading_zeros()) as usize }
}

/// Why a log opening does not verify against its entry.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum LogOpeningError {
    /// The log index is not below the entry's log count.
    IndexOutOfRange,
    /// The opening does not have [`log_tree_depth`] siblings.
    SiblingCount {
        /// The depth of the entry's tree.
        expected: usize,
        /// The number of siblings in the opening.
        actual: usize,
    },
    /// The path from the leaf does not give the entry's log root.
    RootMismatch,
}

/// Checks that the opening's leaf is leaf `opening.index` of the tree committed by `entry`.
fn verify_log_opening(
    entry: &LogCommitmentEntry,
    opening: &LogOpening,
) -> Result<(), LogOpeningError> {
    let LogOpening { index, leaf, siblings, .. } = opening;
    let index = *index;
    if index >= entry.log_count {
        return Err(LogOpeningError::IndexOutOfRange);
    }
    let expected = log_tree_depth(entry.log_count);
    if siblings.len() != expected {
        return Err(LogOpeningError::SiblingCount { expected, actual: siblings.len() });
    }
    let mut hash = *leaf;
    let mut position = index;
    for sibling in siblings {
        hash = if position & 1 == 0 { node(&hash, sibling) } else { node(sibling, &hash) };
        position >>= 1;
    }
    if hash != entry.log_root {
        return Err(LogOpeningError::RootMismatch);
    }
    Ok(())
}

/// Whether the log of `opening` under `entry` has `origin` and `payload_hash`.
///
/// Panics if the opening does not verify. The prover supplies the opening, so a bad one must halt
/// the program, not reject the message.
pub fn opened_log_matches(
    entry: &LogCommitmentEntry,
    opening: &LogOpening,
    origin: &Address,
    payload_hash: &B256,
) -> bool {
    if let Err(err) = verify_log_opening(entry, opening) {
        panic!(
            "opening of log {} under receipts root {} does not verify: {err:?}",
            opening.index, entry.receipts_root
        );
    }
    opening.leaf == log_leaf(origin, payload_hash)
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

/// The leaves of the logs of `receipts`, in block order.
pub fn log_leaves(receipts: &[OpReceiptEnvelope]) -> Vec<B256> {
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

/// The receipts of one block, with the receipts root from its header.
#[derive(Debug, Clone)]
pub struct BlockReceipts {
    /// The receipts root.
    pub receipts_root: B256,
    /// The receipts, in transaction order.
    pub receipts: Vec<OpReceiptEnvelope>,
}

/// Builds the `log-commitment` witness for `blocks`.
///
/// Fails if the receipts of a block do not hash to its receipts root.
pub fn log_commitment_witness(blocks: &[BlockReceipts]) -> Result<DefaultWitnessData> {
    let mut store = PreimageStore::default();
    for (index, BlockReceipts { receipts_root, receipts }) in blocks.iter().enumerate() {
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

/// The opening of log `index` under `receipts_root`: its leaf and the siblings of the leaf.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct LogOpening {
    /// The receipts root of the log's block.
    pub receipts_root: B256,
    /// The log's index in the block.
    pub index: u64,
    /// The log's leaf.
    pub leaf: B256,
    /// The siblings of the leaf, from the leaf level up.
    pub siblings: Vec<B256>,
}

/// The log-commitment input of a consolidation: the public values of each batch proof, in proof
/// order, and the openings of the logs that the consolidation checks.
///
/// Encoded as flat big-endian bytes: a `u32` batch count, then per batch a `u32` entry count and
/// the encoded entries; a `u32` opening count, then per opening the receipts root, the `u64` index,
/// the leaf, a `u8` sibling count, and the siblings.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct LogCommitmentInputs {
    /// The public values of each batch proof: whole encoded entries.
    pub batches: Vec<Vec<u8>>,
    /// The openings, at most one for each receipts root and index.
    pub openings: Vec<LogOpening>,
}

impl LogCommitmentInputs {
    /// Checks the input rules that the stdin builder and the guest share. Fails where
    /// [`LogCommitmentTable::from_batches`] fails, on a second opening for one receipts root and
    /// index, or on an opening with more than 255 siblings.
    pub fn validate(&self) -> Result<()> {
        LogCommitmentTable::from_batches(&self.batches)?;
        let mut opened = BTreeSet::new();
        for opening in &self.openings {
            ensure!(
                opened.insert((opening.receipts_root, opening.index)),
                "second opening of log {} under receipts root {}",
                opening.index,
                opening.receipts_root,
            );
            ensure!(
                opening.siblings.len() <= usize::from(u8::MAX),
                "opening has too many siblings"
            );
        }
        Ok(())
    }

    /// Encodes the inputs. Panics if a batch is not whole entries or an opening has more than 255
    /// siblings.
    pub fn encode(&self) -> Vec<u8> {
        let mut out = Vec::new();
        out.extend_from_slice(&count_u32(self.batches.len()));
        for batch in &self.batches {
            assert_eq!(batch.len() % LOG_COMMITMENT_ENTRY_LEN, 0, "batch is not whole entries");
            out.extend_from_slice(&count_u32(batch.len() / LOG_COMMITMENT_ENTRY_LEN));
            out.extend_from_slice(batch);
        }
        out.extend_from_slice(&count_u32(self.openings.len()));
        for opening in &self.openings {
            out.extend_from_slice(opening.receipts_root.as_slice());
            out.extend_from_slice(&opening.index.to_be_bytes());
            out.extend_from_slice(opening.leaf.as_slice());
            out.push(u8::try_from(opening.siblings.len()).expect("at most 255 siblings"));
            for sibling in &opening.siblings {
                out.extend_from_slice(sibling.as_slice());
            }
        }
        out
    }

    /// Decodes inputs encoded by [`Self::encode`]. Panics on malformed bytes: the prover supplies
    /// them.
    pub fn decode(bytes: &[u8]) -> Self {
        let mut reader = Reader(bytes);
        let batches = (0..reader.u32())
            .map(|_| {
                let entry_count = reader.u32() as usize;
                reader.take(entry_count * LOG_COMMITMENT_ENTRY_LEN).to_vec()
            })
            .collect();
        let openings = (0..reader.u32())
            .map(|_| {
                let receipts_root = reader.b256();
                let index = u64::from_be_bytes(reader.take(8).try_into().expect("8 bytes"));
                let leaf = reader.b256();
                let siblings = (0..reader.take(1)[0]).map(|_| reader.b256()).collect();
                LogOpening { receipts_root, index, leaf, siblings }
            })
            .collect();
        assert!(reader.0.is_empty(), "{} trailing log-commitment input bytes", reader.0.len());
        Self { batches, openings }
    }
}

fn count_u32(count: usize) -> [u8; 4] {
    u32::try_from(count).expect("count fits in u32").to_be_bytes()
}

struct Reader<'a>(&'a [u8]);

impl<'a> Reader<'a> {
    fn take(&mut self, len: usize) -> &'a [u8] {
        assert!(self.0.len() >= len, "truncated log-commitment input");
        let (taken, rest) = self.0.split_at(len);
        self.0 = rest;
        taken
    }

    fn u32(&mut self) -> u32 {
        u32::from_be_bytes(self.take(4).try_into().expect("4 bytes"))
    }

    fn b256(&mut self) -> B256 {
        B256::from_slice(self.take(32))
    }
}

/// Verified log-commitment entries, keyed by receipts root.
#[derive(Debug, Clone, Default)]
pub struct LogCommitmentTable(BTreeMap<B256, LogCommitmentEntry>);

impl LogCommitmentTable {
    /// Indexes the entries of `batches`. Fails if a batch is not whole entries, or if two entries
    /// for one receipts root differ.
    pub fn from_batches(batches: &[Vec<u8>]) -> Result<Self> {
        let mut entries = BTreeMap::new();
        for batch in batches {
            ensure!(batch.len() % LOG_COMMITMENT_ENTRY_LEN == 0, "batch is not whole entries");
            for bytes in batch.chunks_exact(LOG_COMMITMENT_ENTRY_LEN) {
                let entry = LogCommitmentEntry::decode(bytes.try_into().expect("entry length"));
                let previous = entries.insert(entry.receipts_root, entry);
                ensure!(
                    previous.is_none_or(|previous| previous == entry),
                    "two log-commitment entries for receipts root {}",
                    entry.receipts_root
                );
            }
        }
        Ok(Self(entries))
    }

    /// Calls `verify(vkey, sha256(batch))` for each batch, in order, then indexes the entries.
    /// `verify` must halt unless a proof of the program with `vkey` committed `batch`.
    ///
    /// Panics where [`Self::from_batches`] fails.
    pub fn verify_batches(
        batches: &[Vec<u8>],
        vkey: &[u32; 8],
        mut verify: impl FnMut(&[u32; 8], &[u8; 32]),
    ) -> Self {
        for batch in batches {
            verify(vkey, &Sha256::digest(batch).into());
        }
        Self::from_batches(batches).unwrap_or_else(|err| panic!("{err}"))
    }

    /// The entry for `receipts_root`.
    pub fn get(&self, receipts_root: &B256) -> Option<&LogCommitmentEntry> {
        self.0.get(receipts_root)
    }
}

type Openings = BTreeMap<(B256, u64), LogOpening>;

/// The [`LogCommitmentSource`] of the SP1 consolidation: verified entries and the prover's
/// openings. A block without an entry is read through its receipts.
#[derive(Debug)]
pub struct VerifiedLogCommitments {
    table: LogCommitmentTable,
    openings: Arc<Openings>,
}

impl VerifiedLogCommitments {
    /// Verifies the batches of `inputs` (see [`LogCommitmentTable::verify_batches`]) and indexes
    /// its openings. Panics where [`LogCommitmentInputs::validate`] fails.
    pub fn new(
        inputs: LogCommitmentInputs,
        vkey: &[u32; 8],
        verify: impl FnMut(&[u32; 8], &[u8; 32]),
    ) -> Self {
        inputs.validate().unwrap_or_else(|err| panic!("{err}"));
        let table = LogCommitmentTable::verify_batches(&inputs.batches, vkey, verify);
        let openings = inputs
            .openings
            .into_iter()
            .map(|opening| ((opening.receipts_root, opening.index), opening))
            .collect();
        Self { table, openings: Arc::new(openings) }
    }
}

impl LogCommitmentSource for VerifiedLogCommitments {
    fn log_commitment(&self, _chain_id: u64, header: &Header) -> Option<Box<dyn LogCommitment>> {
        let entry = *self.table.get(&header.receipts_root)?;
        Some(Box::new(OpenedBlock { entry, openings: self.openings.clone() }))
    }
}

/// A block with an entry, checked through the prover's openings.
#[derive(Debug)]
struct OpenedBlock {
    entry: LogCommitmentEntry,
    openings: Arc<Openings>,
}

impl LogCommitment for OpenedBlock {
    fn log_count(&self) -> u64 {
        self.entry.log_count
    }

    fn log_matches(&self, index: u64, origin: &Address, payload_hash: &B256) -> bool {
        let Some(opening) = self.openings.get(&(self.entry.receipts_root, index)) else {
            panic!("no opening of log {index} under receipts root {}", self.entry.receipts_root);
        };
        opened_log_matches(&self.entry, opening, origin, payload_hash)
    }
}

#[cfg(test)]
mod tests {
    use std::sync::Arc;

    use alloy_consensus::Header;
    use alloy_primitives::hex;
    use alloy_rlp::Encodable;
    use kona_interop::{
        DependencySet, InitiatingLogs, InteropProvider, RawMessagePayload, SuperRoot,
    };
    use kona_proof::errors::OracleProviderError;
    use kona_proof_interop::{BootInfo, LogCommitmentSource, OracleInteropProvider, PreState};
    use rand::Rng;

    use super::{
        test_utils::{
            deposit, eip1559, logs, missing_trie_node_witness, mixed_blocks, receipts_root, save,
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

    /// The logs that `OracleInteropProvider::initiating_logs` returns for `block_hash`.
    fn interop_provider_logs(
        oracle: Arc<WitnessOracle>,
        block_hash: B256,
    ) -> Result<Vec<Log>, OracleProviderError> {
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
        let InitiatingLogs::Receipts(logs) = kona_proof::block_on(async {
            let header = provider.header_by_hash(CHAIN_ID, block_hash).await?;
            provider.initiating_logs(CHAIN_ID, &header).await
        })?
        else {
            panic!("a provider without a log commitment source serves receipts");
        };
        Ok(logs)
    }

    /// The leaves of the log list that `MessageGraph` indexes for the block with `block_hash`.
    fn interop_provider_leaves(oracle: Arc<WitnessOracle>, block_hash: B256) -> Vec<B256> {
        interop_provider_logs(oracle, block_hash)
            .unwrap()
            .into_iter()
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
            .map(|(block, number)| save_header(&mut store, number, block.receipts_root))
            .collect();
        let oracle = oracle(DefaultWitnessData::from_parts(store, blobs));

        for (block, block_hash) in blocks.iter().zip(block_hashes) {
            let leaves = interop_provider_leaves(oracle.clone(), block_hash);
            assert_eq!(log_leaves(&block.receipts), leaves);
            assert_eq!(
                commit_receipts_root(&oracle, block.receipts_root),
                LogCommitmentEntry {
                    receipts_root: block.receipts_root,
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
        let logs = interop_provider_logs(oracle, block_hash);
        assert!(logs.is_err(), "the interop provider accepts the trie: {logs:?}");
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
        let mut receipt = Vec::new();
        receipts[0].encode_2718(&mut receipt);
        let witness =
            log_commitment_witness(&[BlockReceipts { receipts_root: root, receipts }]).unwrap();
        let mut bytes = witness_bytes(&witness);
        // The single trie node is a leaf holding the encoded receipt; change one of its bytes.
        let at = bytes.windows(receipt.len()).position(|window| window == receipt).unwrap();
        bytes[at + receipt.len() - 1] ^= 1;

        log_commitment_public_values(&[root], &bytes);
    }

    /// Pins the public-value bytes of a batch with a duplicate root.
    #[test]
    fn public_values_golden_bytes() {
        let receipts = vec![deposit(vec![], Some((1, 1))), eip1559(logs(1, 2))];
        let block = BlockReceipts { receipts_root: receipts_root(&receipts), receipts };
        let empty_block = BlockReceipts { receipts_root: EMPTY_ROOT_HASH, receipts: vec![] };
        let blocks = vec![block.clone(), empty_block, block];
        let roots: Vec<B256> = blocks.iter().map(|block| block.receipts_root).collect();
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
        let blocks = vec![BlockReceipts {
            receipts_root: B256::repeat_byte(1),
            receipts: vec![eip1559(logs(0, 1))],
        }];
        let err = log_commitment_witness(&blocks).unwrap_err();
        assert!(err.to_string().starts_with("block 0: receipts hash to"), "{err}");
    }

    fn random_leaves(len: usize) -> Vec<B256> {
        let mut rng = rand::rng();
        (0..len).map(|_| B256::from(rng.random::<[u8; 32]>())).collect()
    }

    fn entry_for(leaves: &[B256]) -> LogCommitmentEntry {
        LogCommitmentEntry {
            receipts_root: B256::repeat_byte(0xee),
            log_count: leaves.len() as u64,
            log_root: log_root(leaves.to_vec()),
        }
    }

    /// Pins that the opening of every log verifies, for `L` = 1 to 9 and 3,481.
    #[test]
    fn opening_of_each_log_verifies() {
        for len in (1..=9).chain([3_481]) {
            let leaves = random_leaves(len);
            let entry = entry_for(&leaves);
            let tree = LogTree::new(leaves.clone());
            assert_eq!(tree.root(), entry.log_root);
            for index in 0..len as u64 {
                let opening = tree.opening(entry.receipts_root, index);
                assert_eq!(opening.leaf, leaves[index as usize]);
                assert_eq!(opening.siblings.len(), log_tree_depth(len as u64), "L = {len}");
                assert_eq!(verify_log_opening(&entry, &opening), Ok(()), "L = {len}, i = {index}");
            }
        }
    }

    /// Pins rejection of a wrong leaf or sibling, a wrong sibling count, and `i = L`.
    #[test]
    fn opening_rejects_tampering() {
        for len in [2, 5, 8, 9] {
            let leaves = random_leaves(len);
            let entry = entry_for(&leaves);
            let tree = LogTree::new(leaves);
            for index in 0..len as u64 {
                let LogOpening { leaf, siblings, .. } = tree.opening(entry.receipts_root, index);
                let verify = |leaf: &B256, siblings: &[B256]| {
                    let opening = LogOpening {
                        receipts_root: entry.receipts_root,
                        index,
                        leaf: *leaf,
                        siblings: siblings.to_vec(),
                    };
                    verify_log_opening(&entry, &opening)
                };
                assert_eq!(
                    verify(&B256::repeat_byte(1), &siblings),
                    Err(LogOpeningError::RootMismatch)
                );
                for position in 0..siblings.len() {
                    let mut wrong = siblings.clone();
                    wrong[position] ^= B256::with_last_byte(1);
                    assert_eq!(verify(&leaf, &wrong), Err(LogOpeningError::RootMismatch));
                }
                let depth = siblings.len();
                let mut longer = siblings.clone();
                longer.push(B256::ZERO);
                assert_eq!(
                    verify(&leaf, &longer),
                    Err(LogOpeningError::SiblingCount { expected: depth, actual: depth + 1 })
                );
                assert_eq!(
                    verify(&leaf, &siblings[..depth - 1]),
                    Err(LogOpeningError::SiblingCount { expected: depth, actual: depth - 1 })
                );
            }
            let beyond = LogOpening {
                receipts_root: entry.receipts_root,
                index: len as u64,
                leaf: B256::ZERO,
                siblings: Vec::new(),
            };
            assert_eq!(verify_log_opening(&entry, &beyond), Err(LogOpeningError::IndexOutOfRange));
        }
    }

    /// Pins the depth: `ceil(log2 L)`, 0 for one leaf.
    #[test]
    fn tree_depth_is_ceil_log2() {
        let depths: Vec<usize> = (0..=9).map(log_tree_depth).collect();
        assert_eq!(depths, [0, 0, 1, 2, 2, 3, 3, 3, 3, 4]);
        assert_eq!(log_tree_depth(1 << 32), 32);
        assert_eq!(log_tree_depth((1 << 32) + 1), 33);
        assert_eq!(log_tree_depth(u64::MAX), 64);
    }

    fn sample_inputs() -> LogCommitmentInputs {
        let entry = |fill: u8, log_count| {
            LogCommitmentEntry {
                receipts_root: B256::repeat_byte(fill),
                log_count,
                log_root: B256::repeat_byte(fill + 1),
            }
            .encode()
        };
        LogCommitmentInputs {
            batches: vec![[entry(0x10, 3), entry(0x20, 0)].concat(), Vec::new()],
            openings: vec![
                LogOpening {
                    receipts_root: B256::repeat_byte(0x10),
                    index: 2,
                    leaf: B256::repeat_byte(0x30),
                    siblings: vec![B256::repeat_byte(0x40), B256::repeat_byte(0x50)],
                },
                LogOpening {
                    receipts_root: B256::repeat_byte(0x60),
                    index: 0,
                    leaf: B256::repeat_byte(0x70),
                    siblings: Vec::new(),
                },
            ],
        }
    }

    /// Pins the flat encoding of the consolidation's log-commitment input.
    #[test]
    fn inputs_encoding_golden_bytes() {
        let inputs = sample_inputs();
        let encoded = inputs.encode();
        let expected = [
            "00000002",
            "00000002",
            &"10".repeat(32),
            "0000000000000003",
            &"11".repeat(32),
            &"20".repeat(32),
            "0000000000000000",
            &"21".repeat(32),
            "00000000",
            "00000002",
            &"10".repeat(32),
            "0000000000000002",
            &"30".repeat(32),
            "02",
            &"40".repeat(32),
            &"50".repeat(32),
            &"60".repeat(32),
            "0000000000000000",
            &"70".repeat(32),
            "00",
        ]
        .concat();
        assert_eq!(hex::encode(&encoded), expected);
        assert_eq!(LogCommitmentInputs::decode(&encoded), inputs);
        assert_eq!(LogCommitmentInputs::default().encode(), [0u8; 8]);
    }

    #[test]
    #[should_panic(expected = "truncated log-commitment input")]
    fn inputs_decode_rejects_truncation() {
        let encoded = sample_inputs().encode();
        LogCommitmentInputs::decode(&encoded[..encoded.len() - 1]);
    }

    #[test]
    #[should_panic(expected = "1 trailing log-commitment input bytes")]
    fn inputs_decode_rejects_trailing_bytes() {
        let mut encoded = sample_inputs().encode();
        encoded.push(0);
        LogCommitmentInputs::decode(&encoded);
    }

    #[test]
    #[should_panic(expected = "second opening of log 2")]
    fn source_rejects_duplicate_opening() {
        let mut inputs = sample_inputs();
        inputs.openings.push(inputs.openings[0].clone());
        VerifiedLogCommitments::new(inputs, &[0; 8], |_, _| {});
    }

    /// Pins one `verify` call per batch, in order, with the vkey and the SHA-256 of the batch.
    #[test]
    fn table_verifies_each_batch_once() {
        let inputs = sample_inputs();
        let vkey = [7u32; 8];
        let mut calls = Vec::new();
        let table = LogCommitmentTable::verify_batches(&inputs.batches, &vkey, |vkey, digest| {
            calls.push((*vkey, *digest))
        });
        let expected: Vec<([u32; 8], [u8; 32])> =
            inputs.batches.iter().map(|batch| (vkey, Sha256::digest(batch).into())).collect();
        assert_eq!(calls, expected);
        assert_eq!(table.get(&B256::repeat_byte(0x20)).unwrap().log_count, 0);
        assert_eq!(table.get(&B256::repeat_byte(0x10)).unwrap().log_count, 3);
        assert!(table.get(&B256::repeat_byte(0x60)).is_none());
    }

    #[test]
    fn table_accepts_equal_duplicate_entries() {
        let batch = sample_inputs().batches[0].clone();
        let table = LogCommitmentTable::verify_batches(&[batch.clone(), batch], &[0; 8], |_, _| {});
        assert_eq!(table.0.len(), 2);
    }

    /// Pins a halt when two batches commit different entries for one receipts root.
    #[test]
    #[should_panic(expected = "two log-commitment entries for receipts root")]
    fn table_rejects_conflicting_entries() {
        let entry = LogCommitmentEntry {
            receipts_root: B256::repeat_byte(1),
            log_count: 1,
            log_root: B256::repeat_byte(2),
        };
        let other = LogCommitmentEntry { log_count: 0, log_root: B256::ZERO, ..entry };
        LogCommitmentTable::verify_batches(
            &[entry.encode().to_vec(), other.encode().to_vec()],
            &[0; 8],
            |_, _| {},
        );
    }

    #[test]
    #[should_panic(expected = "batch is not whole entries")]
    fn table_rejects_partial_entry() {
        LogCommitmentTable::verify_batches(
            &[vec![0; LOG_COMMITMENT_ENTRY_LEN - 1]],
            &[0; 8],
            |_, _| {},
        );
    }

    /// The opening of every log under `receipts_root`.
    fn openings(receipts_root: B256, leaves: Vec<B256>) -> Vec<LogOpening> {
        let tree = LogTree::new(leaves);
        (0..tree.log_count()).map(|index| tree.opening(receipts_root, index)).collect()
    }

    /// One block of five logs with origin `i` and payload hash `i`, its commitment source, and
    /// header.
    fn opened_block(
        edit: impl FnOnce(&mut LogCommitmentInputs),
    ) -> (Vec<(Address, B256)>, VerifiedLogCommitments, Header) {
        let logs: Vec<(Address, B256)> =
            (0..5u8).map(|i| (Address::repeat_byte(i), B256::repeat_byte(i))).collect();
        let leaves: Vec<B256> = logs.iter().map(|(origin, hash)| log_leaf(origin, hash)).collect();
        let entry = LogCommitmentEntry {
            receipts_root: B256::repeat_byte(0xaa),
            log_count: leaves.len() as u64,
            log_root: log_root(leaves.clone()),
        };
        let mut inputs = LogCommitmentInputs {
            batches: vec![entry.encode().to_vec()],
            openings: openings(entry.receipts_root, leaves),
        };
        edit(&mut inputs);
        let source = VerifiedLogCommitments::new(inputs, &[0; 8], |_, _| {});
        let header = Header { receipts_root: entry.receipts_root, ..Default::default() };
        (logs, source, header)
    }

    #[test]
    fn source_checks_messages_through_openings() {
        let (logs, source, header) = opened_block(|_| {});
        let block = source.log_commitment(CHAIN_ID, &header).unwrap();
        assert_eq!(block.log_count(), 5);
        for (index, (origin, hash)) in logs.iter().enumerate() {
            assert!(block.log_matches(index as u64, origin, hash));
            assert!(!block.log_matches(index as u64, &Address::repeat_byte(9), hash));
            assert!(!block.log_matches(index as u64, origin, &B256::repeat_byte(9)));
        }
        let unknown = Header { receipts_root: B256::repeat_byte(0xbb), ..Default::default() };
        assert!(source.log_commitment(CHAIN_ID, &unknown).is_none());
    }

    #[test]
    #[should_panic(expected = "no opening of log 3")]
    fn source_halts_on_missing_opening() {
        let (logs, source, header) = opened_block(|inputs| {
            inputs.openings.remove(3);
        });
        let (origin, hash) = logs[3];
        source.log_commitment(CHAIN_ID, &header).unwrap().log_matches(3, &origin, &hash);
    }

    #[test]
    #[should_panic(expected = "SiblingCount { expected: 3, actual: 2 }")]
    fn source_halts_on_wrong_sibling_count() {
        let (logs, source, header) = opened_block(|inputs| {
            inputs.openings[1].siblings.pop();
        });
        let (origin, hash) = logs[1];
        source.log_commitment(CHAIN_ID, &header).unwrap().log_matches(1, &origin, &hash);
    }

    /// Pins a halt on an opening whose path does not give the root.
    #[test]
    #[should_panic(expected = "RootMismatch")]
    fn source_halts_on_path_that_does_not_give_root() {
        let (logs, source, header) =
            opened_block(|inputs| inputs.openings[4].leaf = B256::repeat_byte(9));
        let (origin, hash) = logs[4];
        source.log_commitment(CHAIN_ID, &header).unwrap().log_matches(4, &origin, &hash);
    }

    const INITIATING: u64 = 1;
    const EXECUTING: u64 = 2;

    /// Chain `INITIATING`'s block 1 holds 3 logs and sits below its head, block 2. Chain
    /// `EXECUTING`'s head executes `message` into block 1. Returns the verdict through the
    /// receipts, and through `VerifiedLogCommitments` with block 1's receipts trie withheld from
    /// the witness, so that reading it panics.
    fn verdicts(message: kona_interop::ExecutingMessageBuilder) -> (bool, bool) {
        use alloy_consensus::Sealed;
        use alloy_primitives::map::HashMap;
        use kona_genesis::RollupConfig;
        use kona_interop::{ChainBuilder, ChainDependency, MESSAGE_EXPIRY_WINDOW, MessageGraph};

        let block_1 = vec![eip1559(logs(0, 2)), deposit(vec![], None), eip1559(logs(2, 1))];
        let mut executing = ChainBuilder::default();
        executing.add_executing_message(
            message
                .with_origin_chain_id(INITIATING)
                .with_origin_block_number(1)
                .with_origin_timestamp(2),
        );
        let blocks: Vec<_> = [block_1, vec![], executing.receipts]
            .into_iter()
            .map(|receipts| BlockReceipts { receipts_root: receipts_root(&receipts), receipts })
            .collect();
        let block_1_root = blocks[0].receipts_root;
        let (mut withheld, _) = log_commitment_witness(&blocks[1..]).unwrap().into_parts();
        let (mut full, _) = log_commitment_witness(&blocks).unwrap().into_parts();
        let mut seal = |number, receipts_root, parent_hash| {
            let header = Header {
                number,
                timestamp: 2 * number,
                receipts_root,
                parent_hash,
                ..Default::default()
            };
            let mut rlp = Vec::new();
            header.encode(&mut rlp);
            save(&mut withheld, rlp.clone());
            Sealed::new_unchecked(header, save(&mut full, rlp))
        };
        let a1 = seal(1, block_1_root, B256::ZERO);
        let heads: HashMap<u64, Sealed<Header>> = [
            (INITIATING, seal(2, blocks[1].receipts_root, a1.hash())),
            (EXECUTING, seal(2, blocks[2].receipts_root, B256::ZERO)),
        ]
        .into_iter()
        .collect();

        let mut config = RollupConfig { block_time: 2, ..Default::default() };
        config.hardforks.lagoon_time = Some(0);
        let configs: HashMap<u64, RollupConfig> =
            [(INITIATING, config.clone()), (EXECUTING, config)].into_iter().collect();
        let dependency_set = DependencySet {
            dependencies: [(INITIATING, ChainDependency {}), (EXECUTING, ChainDependency {})]
                .into(),
            override_message_expiry_window: None,
        };
        let boot = BootInfo {
            l1_head: B256::ZERO,
            agreed_pre_state_commitment: B256::ZERO,
            agreed_pre_state: PreState::SuperRoot(SuperRoot::new(0, Vec::new())),
            claimed_post_state: B256::ZERO,
            claimed_l2_timestamp: 0,
            rollup_configs: configs.clone(),
            dependency_set: dependency_set.clone(),
            l1_config: Default::default(),
        };
        let leaves = log_leaves(&blocks[0].receipts);
        let entry = commit_receipts_root(
            &oracle(DefaultWitnessData::from_parts(full.clone(), BlobData::default())),
            block_1_root,
        );
        let inputs = LogCommitmentInputs {
            batches: vec![entry.encode().to_vec()],
            openings: openings(block_1_root, leaves),
        };
        let resolve = |store: PreimageStore, commitments: Option<VerifiedLogCommitments>| {
            let mut provider = OracleInteropProvider::new(
                oracle(DefaultWitnessData::from_parts(store, BlobData::default())),
                boot.clone(),
                heads.clone(),
            );
            if let Some(commitments) = commitments {
                provider = provider.with_log_commitments(Arc::new(commitments));
            }
            kona_proof::block_on(async {
                let blocks = provider.blocks_with_receipts(&heads).await.unwrap();
                MessageGraph::derive(
                    &blocks,
                    &provider,
                    &configs,
                    &dependency_set,
                    MESSAGE_EXPIRY_WINDOW,
                )
                .resolve()
                .await
                .is_ok()
            })
        };
        (
            resolve(full, None),
            resolve(withheld, Some(VerifiedLogCommitments::new(inputs, &[0; 8], |_, _| {}))),
        )
    }

    /// Pins receipts-path verdicts through openings, with the served receipts withheld.
    #[test]
    fn graph_verdicts_through_openings_match_receipts() {
        use kona_interop::ExecutingMessageBuilder;

        let log = |index: u64| logs(index, 1).remove(0);
        let payload_hash = |log: &Log| keccak256(RawMessagePayload::from(log).as_ref());
        let message = |index: u64, origin: Address, payload_hash: B256| {
            ExecutingMessageBuilder::default()
                .with_origin_log_index(index)
                .with_origin_address(origin)
                .with_message_hash(payload_hash)
        };
        let cases = [
            ("valid first", message(0, log(0).address, payload_hash(&log(0))), true),
            ("valid last", message(2, log(2).address, payload_hash(&log(2))), true),
            ("wrong origin", message(1, log(0).address, payload_hash(&log(1))), false),
            ("wrong payload", message(1, log(1).address, payload_hash(&log(0))), false),
            ("index = L", message(3, log(2).address, payload_hash(&log(2))), false),
        ];
        for (name, message, expected) in cases {
            assert_eq!(verdicts(message), (expected, expected), "{name}");
        }
    }
}
