//! Log commitments in the native consolidation run that collects the witness.
//!
//! The run uses the guest's entry table and verdict code, but builds each opening from the
//! block's receipts. It reads the receipts through the unrecorded host oracle, so the witness has
//! no receipts trie node of a served block.

use std::{
    collections::{BTreeMap, BTreeSet},
    fmt::Debug,
    fs, io,
    num::NonZeroUsize,
    path::{Path, PathBuf},
    sync::{
        Arc, Mutex, MutexGuard,
        atomic::{AtomicU64, Ordering},
    },
};

use alloy_consensus::Header;
use alloy_primitives::{Address, B256};
use anyhow::{Context, Result, anyhow, ensure};
use async_trait::async_trait;
use kona_interop::LogCommitment;
use kona_preimage::{
    CommsClient, HintWriterClient, PreimageKey, PreimageOracleClient, errors::PreimageOracleResult,
};
use kona_proof::FlushableCache;
use kona_proof_interop::{HintType, LogCommitmentSource, receipts_from_trie};
use kona_sp1_client_utils::{
    log_commitment::{
        LogCommitmentEntry, LogCommitmentTable, LogOpening, LogTree, log_leaves, opened_log_matches,
    },
    super_root::{SuperConsolidationInputs, SuperConsolidationOutputs},
};
use kona_sp1_ethereum_client_utils::{
    chain_config::ChainConfigs, super_consolidation::build_consolidation_outputs,
};
use kona_sp1_host_utils::witness_generation::PreimageWitnessCollector;
use lru::LruCache;
use tokio::sync::Notify;

/// A block below a local safe head that the consolidation read through its receipts, because the
/// entry table had no entry for its receipts root.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct MissingLogCommitment {
    /// The block's chain.
    pub chain_id: u64,
    /// The block's hash, to fetch its receipts.
    pub block_hash: B256,
    /// The block's receipts root.
    pub receipts_root: B256,
}

/// The outputs of [`run_native_consolidation`].
#[derive(Debug)]
pub(crate) struct NativeConsolidation {
    pub(crate) outputs: SuperConsolidationOutputs,
    pub(crate) openings: Vec<LogOpening>,
    pub(crate) missing: Vec<MissingLogCommitment>,
    pub(crate) served: Vec<B256>,
    pub(crate) read_bytes: u64,
    pub(crate) log_check_siblings: u64,
}

/// Runs the consolidation natively over `collector`, with the entries of `batches`, the public
/// values of the batch proofs that the stdin will carry. With `log_tree_dir`, each log tree is
/// stored there when built and read back when the run needs it again.
///
/// After a host failure, the run stops at its next asynchronous host read. A read inside a trie
/// walk is synchronous, so a check that is already running finishes first.
pub(crate) async fn run_native_consolidation<P>(
    collector: Arc<PreimageWitnessCollector<P>>,
    inputs: SuperConsolidationInputs,
    configs: Option<&ChainConfigs>,
    batches: &[Vec<u8>],
    log_tree_dir: Option<&Path>,
) -> Result<NativeConsolidation>
where
    P: CommsClient + FlushableCache + Debug + Send + Sync + Clone + 'static,
{
    if let Some(dir) = log_tree_dir {
        fs::create_dir_all(dir)
            .with_context(|| format!("failed to create log tree directory {}", dir.display()))?;
    }
    let record = Arc::new(LogCommitmentRecord {
        tree_dir: log_tree_dir.map(Path::to_path_buf),
        ..Default::default()
    });
    let source =
        NativeLogCommitments::new(batches, collector.preimage_oracle.clone(), record.clone())?;
    let read_bytes = Arc::new(AtomicU64::new(0));
    let oracle = Arc::new(CountingOracle { inner: collector, bytes: read_bytes.clone() });
    let run = build_consolidation_outputs(inputs, oracle, configs, Arc::new(source));
    let outputs = tokio::select! {
        biased;
        () = record.failure.notified() => None,
        outputs = run => Some(outputs),
    };
    // Check the record first: after a host failure, `outputs` is `None` or holds bogus verdicts.
    let Recorded { openings, missing, served, log_check_siblings } = record.finish()?;
    let outputs = outputs.context("the consolidation run stopped without a host failure")??;
    Ok(NativeConsolidation {
        outputs,
        openings,
        missing,
        served,
        read_bytes: read_bytes.load(Ordering::Relaxed),
        log_check_siblings,
    })
}

/// An oracle that counts the bytes it returns, with repeats: the reads that the guest repeats.
#[derive(Debug)]
struct CountingOracle<O> {
    inner: Arc<O>,
    bytes: Arc<AtomicU64>,
}

impl<O> Clone for CountingOracle<O> {
    fn clone(&self) -> Self {
        Self { inner: self.inner.clone(), bytes: self.bytes.clone() }
    }
}

#[async_trait]
impl<O> PreimageOracleClient for CountingOracle<O>
where
    O: PreimageOracleClient + Send + Sync,
{
    async fn get(&self, key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
        let value = self.inner.get(key).await?;
        self.bytes.fetch_add(value.len() as u64, Ordering::Relaxed);
        Ok(value)
    }

    async fn get_exact(&self, key: PreimageKey, buf: &mut [u8]) -> PreimageOracleResult<()> {
        self.inner.get_exact(key, buf).await?;
        self.bytes.fetch_add(buf.len() as u64, Ordering::Relaxed);
        Ok(())
    }
}

#[async_trait]
impl<O> HintWriterClient for CountingOracle<O>
where
    O: HintWriterClient + Send + Sync,
{
    async fn write(&self, hint: &str) -> PreimageOracleResult<()> {
        self.inner.write(hint).await
    }
}

/// What the native run records, shared by every timestamp and round.
#[derive(Debug, Default)]
struct LogCommitmentRecord {
    state: Mutex<RecordState>,
    /// Wakes [`run_native_consolidation`] on the first host failure.
    failure: Notify,
    /// Where built log trees are stored, one file of leaves per receipts root.
    tree_dir: Option<PathBuf>,
}

#[derive(Debug, Default)]
struct RecordState {
    /// One per `(receipts root, index)`: a log is checked in each round, at each timestamp, and
    /// for each message into it, and the guest rejects a second opening.
    openings: BTreeMap<(B256, u64), LogOpening>,
    missing: BTreeMap<B256, MissingLogCommitment>,
    served: BTreeSet<B256>,
    /// Siblings of the openings that the log checks against an entry verify.
    log_check_siblings: u64,
    trees: TreeCache,
    /// The first host failure; later log checks answer without host work.
    error: Option<anyhow::Error>,
}

/// What [`LogCommitmentRecord::finish`] returns.
#[derive(Debug)]
struct Recorded {
    openings: Vec<LogOpening>,
    missing: Vec<MissingLogCommitment>,
    served: Vec<B256>,
    log_check_siblings: u64,
}

impl LogCommitmentRecord {
    fn state(&self) -> MutexGuard<'_, RecordState> {
        self.state.lock().expect("record lock")
    }

    fn record_failure(&self, error: anyhow::Error) {
        self.state().error.get_or_insert(error);
        self.failure.notify_one();
    }

    /// The openings, the missing blocks, and the served receipts roots, or the first host failure.
    fn finish(&self) -> Result<Recorded> {
        let mut state = self.state();
        if let Some(error) = state.error.take() {
            return Err(error.context("failed to open a committed log from the host's receipts"));
        }
        Ok(Recorded {
            openings: std::mem::take(&mut state.openings).into_values().collect(),
            missing: std::mem::take(&mut state.missing).into_values().collect(),
            served: std::mem::take(&mut state.served).into_iter().collect(),
            log_check_siblings: state.log_check_siblings,
        })
    }
}

/// The most log trees kept. `resolve` sorts messages by initiating block, so a tree is built again
/// only when a later timestamp or round reads its block.
const MAX_CACHED_TREES: usize = 4;

/// Recently built log trees by receipts root.
#[derive(Debug)]
struct TreeCache(LruCache<B256, Arc<LogTree>>);

impl Default for TreeCache {
    fn default() -> Self {
        Self(LruCache::new(NonZeroUsize::new(MAX_CACHED_TREES).expect("nonzero tree count")))
    }
}

impl TreeCache {
    fn get(&mut self, receipts_root: &B256) -> Option<Arc<LogTree>> {
        self.0.get(receipts_root).cloned()
    }

    fn insert(&mut self, receipts_root: B256, tree: Arc<LogTree>) {
        self.0.put(receipts_root, tree);
    }
}

/// The [`LogCommitmentSource`] of the native run. A host failure in a log check gives `false` and
/// is recorded; [`LogCommitmentRecord::finish`] then fails the run.
#[derive(Debug)]
struct NativeLogCommitments<O> {
    table: LogCommitmentTable,
    receipts_oracle: Arc<O>,
    record: Arc<LogCommitmentRecord>,
}

impl<O> NativeLogCommitments<O> {
    /// `receipts_oracle` must not record into the witness. The guest verifies the batches.
    fn new(
        batches: &[Vec<u8>],
        receipts_oracle: Arc<O>,
        record: Arc<LogCommitmentRecord>,
    ) -> Result<Self> {
        Ok(Self { table: LogCommitmentTable::from_batches(batches)?, receipts_oracle, record })
    }
}

impl<O> LogCommitmentSource for NativeLogCommitments<O>
where
    O: CommsClient + Debug + Send + Sync + 'static,
{
    fn log_commitment(&self, chain_id: u64, header: &Header) -> Option<Box<dyn LogCommitment>> {
        let block_hash = header.hash_slow();
        let receipts_root = header.receipts_root;
        let Some(entry) = self.table.get(&receipts_root) else {
            self.record.state().missing.entry(receipts_root).or_insert(MissingLogCommitment {
                chain_id,
                block_hash,
                receipts_root,
            });
            return None;
        };
        self.record.state().served.insert(receipts_root);
        Some(Box::new(NativeBlock {
            entry: *entry,
            chain_id,
            block_hash,
            receipts_oracle: self.receipts_oracle.clone(),
            record: self.record.clone(),
        }))
    }
}

#[derive(Debug)]
struct NativeBlock<O> {
    entry: LogCommitmentEntry,
    chain_id: u64,
    block_hash: B256,
    receipts_oracle: Arc<O>,
    record: Arc<LogCommitmentRecord>,
}

impl<O> NativeBlock<O>
where
    O: CommsClient + Send + Sync,
{
    fn opening(&self, index: u64) -> Result<LogOpening> {
        let receipts_root = self.entry.receipts_root;
        if let Some(opening) = self.record.state().openings.get(&(receipts_root, index)) {
            return Ok(opening.clone());
        }
        let opening = self.tree()?.opening(receipts_root, index);
        self.record.state().openings.insert((receipts_root, index), opening.clone());
        Ok(opening)
    }

    fn tree(&self) -> Result<Arc<LogTree>> {
        let receipts_root = self.entry.receipts_root;
        if let Some(tree) = self.record.state().trees.get(&receipts_root) {
            return Ok(tree);
        }
        let path = self.record.tree_dir.as_ref().map(|dir| dir.join(receipts_root.to_string()));
        let leaves = match path.as_deref().map(read_leaves).transpose()?.flatten() {
            Some(leaves) => leaves,
            None => {
                let leaves = self.build_leaves()?;
                if let Some(path) = &path {
                    fs::write(path, leaves.concat())
                        .with_context(|| format!("failed to write log tree {}", path.display()))?;
                }
                leaves
            }
        };
        let tree = LogTree::new(leaves);
        ensure!(
            tree.log_count() == self.entry.log_count && tree.root() == self.entry.log_root,
            "log tree of block {} does not give the entry of receipts root {receipts_root}",
            self.block_hash,
        );
        let tree = Arc::new(tree);
        self.record.state().trees.insert(receipts_root, tree.clone());
        Ok(tree)
    }

    fn build_leaves(&self) -> Result<Vec<B256>> {
        kona_proof::block_on(
            HintType::L2Receipts
                .with_data(&[self.block_hash.as_slice(), &self.chain_id.to_be_bytes()])
                .send(self.receipts_oracle.as_ref()),
        )?;
        let receipts = receipts_from_trie(self.receipts_oracle.as_ref(), self.entry.receipts_root)
            .map_err(|err| anyhow!("receipts of block {}: {err}", self.block_hash))?;
        Ok(log_leaves(&receipts))
    }
}

/// The leaves stored at `path`, or `None` if no tree is stored there.
fn read_leaves(path: &Path) -> Result<Option<Vec<B256>>> {
    let bytes = match fs::read(path) {
        Ok(bytes) => bytes,
        Err(err) if err.kind() == io::ErrorKind::NotFound => return Ok(None),
        Err(err) => {
            return Err(err).with_context(|| format!("failed to read log tree {}", path.display()));
        }
    };
    ensure!(bytes.len() % 32 == 0, "log tree {} has {} bytes", path.display(), bytes.len());
    Ok(Some(bytes.chunks_exact(32).map(B256::from_slice).collect()))
}

impl<O> LogCommitment for NativeBlock<O>
where
    O: CommsClient + Debug + Send + Sync,
{
    fn log_count(&self) -> u64 {
        self.entry.log_count
    }

    fn log_matches(&self, index: u64, origin: &Address, payload_hash: &B256) -> bool {
        if self.record.state().error.is_some() {
            return false;
        }
        match self.opening(index) {
            Ok(opening) => {
                self.record.state().log_check_siblings += opening.siblings.len() as u64;
                opened_log_matches(&self.entry, &opening, origin, payload_hash)
            }
            Err(error) => {
                self.record.record_failure(error);
                false
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use alloy_consensus::{EMPTY_ROOT_HASH, Receipt, ReceiptWithBloom, Sealed};
    use alloy_primitives::{Bytes, LogData, keccak256, map::HashMap};
    use alloy_rlp::{EMPTY_STRING_CODE, Encodable};
    use kona_genesis::RollupConfig;
    use kona_interop::{
        ChainBuilder, ChainDependency, DependencySet, ExecutingMessageBuilder,
        MESSAGE_EXPIRY_WINDOW, MessageGraph, RawMessagePayload, SuperRoot,
    };
    use kona_proof_interop::{BootInfo, OracleInteropProvider, PreState};
    use kona_sp1_client_utils::{
        log_commitment::{
            LogCommitmentInputs, VerifiedLogCommitments,
            test_utils::{receipts_root, save, save_receipts},
        },
        witness::{BlobData, DefaultWitnessData, WitnessData, preimage_store::PreimageStore},
    };
    use op_alloy_consensus::OpReceiptEnvelope;

    use super::*;

    const A: u64 = 1;
    const B: u64 = 2;

    /// Chain A's blocks 1 and 2 sit below its head, block 3; chain B's head executes messages
    /// into them.
    struct Chains {
        /// Every preimage except the receipts tries of blocks 1 and 2.
        withheld: PreimageStore,
        full: PreimageStore,
        below_head: [(B256, Vec<OpReceiptEnvelope>); 2],
        heads: HashMap<u64, Sealed<Header>>,
        boot: BootInfo,
    }

    /// Two messages into one log of block 1 and one into block 2.
    fn chains() -> Chains {
        chains_with(&[(1, 1, 2), (1, 1, 2), (2, 0, 4)])
    }

    /// Block 1 holds logs with seeds 1 to 3 and block 2 one log with seed 4. Each message is
    /// `(block number, log index, seed)`, carrying the origin and payload of the seed's log.
    fn chains_with(messages: &[(u64, u64, u8)]) -> Chains {
        let log = |seed: u8| alloy_primitives::Log {
            address: Address::repeat_byte(seed),
            data: LogData::new_unchecked(vec![B256::repeat_byte(seed)], Bytes::from(vec![seed])),
        };
        let receipt = |logs| {
            OpReceiptEnvelope::Eip1559(ReceiptWithBloom {
                receipt: Receipt { status: true.into(), cumulative_gas_used: 1, logs },
                logs_bloom: Default::default(),
            })
        };
        let blocks = [vec![receipt(vec![log(1), log(2), log(3)])], vec![receipt(vec![log(4)])]];
        let mut executing = ChainBuilder::default();
        for &(number, index, seed) in messages {
            executing.add_executing_message(
                ExecutingMessageBuilder::default()
                    .with_origin_address(Address::repeat_byte(seed))
                    .with_message_hash(keccak256(RawMessagePayload::from(&log(seed)).as_ref()))
                    .with_origin_chain_id(A)
                    .with_origin_block_number(number)
                    .with_origin_log_index(index)
                    .with_origin_timestamp(2 * number),
            );
        }

        let mut withheld = PreimageStore::default();
        save(&mut withheld, vec![EMPTY_STRING_CODE]);
        let seal = |store: &mut PreimageStore, number, parent_hash, receipts_root| {
            let header = Header {
                number,
                timestamp: 2 * number,
                parent_hash,
                receipts_root,
                ..Default::default()
            };
            let mut rlp = Vec::new();
            header.encode(&mut rlp);
            Sealed::new_unchecked(header, save(store, rlp))
        };
        let below_head = blocks.map(|receipts| (receipts_root(&receipts), receipts));
        let a1 = seal(&mut withheld, 1, B256::ZERO, below_head[0].0);
        let a2 = seal(&mut withheld, 2, a1.hash(), below_head[1].0);
        let a3 = seal(&mut withheld, 3, a2.hash(), EMPTY_ROOT_HASH);
        let executing_root = save_receipts(&mut withheld, &executing.receipts);
        let b3 = seal(&mut withheld, 3, B256::ZERO, executing_root);
        let mut full = withheld.clone();
        for (_, receipts) in &below_head {
            save_receipts(&mut full, receipts);
        }

        let mut config = RollupConfig { block_time: 2, ..Default::default() };
        config.hardforks.lagoon_time = Some(0);
        let boot = BootInfo {
            l1_head: B256::ZERO,
            agreed_pre_state_commitment: B256::ZERO,
            agreed_pre_state: PreState::SuperRoot(SuperRoot::new(0, Vec::new())),
            claimed_post_state: B256::ZERO,
            claimed_l2_timestamp: 0,
            rollup_configs: [(A, config.clone()), (B, config)].into_iter().collect(),
            dependency_set: DependencySet {
                dependencies: [(A, ChainDependency {}), (B, ChainDependency {})].into(),
                override_message_expiry_window: None,
            },
            l1_config: Default::default(),
        };
        Chains { withheld, full, below_head, heads: [(A, a3), (B, b3)].into_iter().collect(), boot }
    }

    impl Chains {
        fn batch(&self) -> Vec<u8> {
            self.below_head
                .iter()
                .flat_map(|(receipts_root, receipts)| {
                    let tree = LogTree::new(log_leaves(receipts));
                    LogCommitmentEntry {
                        receipts_root: *receipts_root,
                        log_count: tree.log_count(),
                        log_root: tree.root(),
                    }
                    .encode()
                })
                .collect()
        }

        /// Resolves the graph `rounds` times over the `witness` oracle and `source`.
        fn resolve(
            &self,
            witness: PreimageStore,
            source: Arc<dyn LogCommitmentSource>,
            rounds: usize,
        ) -> bool {
            let oracle = kona_proof::block_on(
                DefaultWitnessData::from_parts(witness, BlobData::default())
                    .get_oracle_and_blob_provider(),
            )
            .unwrap()
            .0;
            let provider =
                OracleInteropProvider::new(oracle, self.boot.clone(), self.heads.clone())
                    .with_log_commitments(source);
            let mut valid = true;
            for _ in 0..rounds {
                valid &= kona_proof::block_on(async {
                    let blocks = provider.blocks_with_receipts(&self.heads).await.unwrap();
                    MessageGraph::derive(
                        &blocks,
                        &provider,
                        &self.boot.rollup_configs,
                        &self.boot.dependency_set,
                        MESSAGE_EXPIRY_WINDOW,
                    )
                    .resolve()
                    .await
                    .is_ok()
                });
            }
            valid
        }
    }

    /// Pins one opening per log and each log check across rounds, accepted by the guest's source.
    #[tokio::test(flavor = "multi_thread")]
    async fn records_each_opening_once_for_the_guest() {
        let chains = chains();
        let record = Arc::new(LogCommitmentRecord::default());
        let batches = vec![chains.batch()];
        let native = Arc::new(
            NativeLogCommitments::new(&batches, Arc::new(chains.full.clone()), record.clone())
                .unwrap(),
        );
        assert!(chains.resolve(chains.withheld.clone(), native, 2));
        let Recorded { openings, missing, log_check_siblings, .. } = record.finish().unwrap();
        // Each round checks one log of block 1 (three logs: two siblings) and one of block 2 (one
        // log: none).
        assert_eq!(log_check_siblings, 4);
        assert!(missing.is_empty());
        let keys: Vec<_> =
            openings.iter().map(|opening| (opening.receipts_root, opening.index)).collect();
        assert_eq!(keys, [(chains.below_head[0].0, 1), (chains.below_head[1].0, 0)]);

        let inputs = LogCommitmentInputs { batches, openings };
        let inputs = LogCommitmentInputs::decode(&inputs.encode());
        let guest = Arc::new(VerifiedLogCommitments::new(inputs, &[0; 8], |_, _| {}));
        assert!(chains.resolve(chains.withheld.clone(), guest, 2));
    }

    /// Pins one report per block without an entry.
    #[tokio::test(flavor = "multi_thread")]
    async fn records_each_missing_block_once() {
        let chains = chains();
        let record = Arc::new(LogCommitmentRecord::default());
        let native = Arc::new(
            NativeLogCommitments::new(&[], Arc::new(chains.full.clone()), record.clone()).unwrap(),
        );
        assert!(chains.resolve(chains.full.clone(), native, 2));
        let Recorded { openings, missing, .. } = record.finish().unwrap();
        assert!(openings.is_empty());
        let roots: Vec<_> = missing.iter().map(|block| block.receipts_root).collect();
        let mut expected: Vec<_> = chains.below_head.iter().map(|(root, _)| *root).collect();
        expected.sort();
        assert_eq!(roots, expected);
        assert!(missing.iter().all(|block| block.chain_id == A));
    }

    /// Pins a served root for every looked-up block with an entry, also when no log is opened.
    #[tokio::test(flavor = "multi_thread")]
    async fn records_served_roots() {
        // Block 2 has one log, so its only message, at index 5, is rejected without an opening.
        let chains = chains_with(&[(1, 1, 2), (2, 5, 4)]);
        let record = Arc::new(LogCommitmentRecord::default());
        let native = Arc::new(
            NativeLogCommitments::new(
                &[chains.batch()],
                Arc::new(chains.full.clone()),
                record.clone(),
            )
            .unwrap(),
        );
        assert!(!chains.resolve(chains.withheld.clone(), native, 1));
        let Recorded { openings, missing, served, .. } = record.finish().unwrap();
        assert!(missing.is_empty());
        let mut expected: Vec<_> = chains.below_head.iter().map(|(root, _)| *root).collect();
        expected.sort();
        assert_eq!(served, expected);
        let opened: Vec<_> = openings.iter().map(|opening| opening.receipts_root).collect();
        assert_eq!(opened, [chains.below_head[0].0]);
    }

    /// Pins a stored log tree that serves a later run without the block's receipts.
    #[tokio::test(flavor = "multi_thread")]
    async fn stored_tree_serves_a_later_run() {
        let chains = chains();
        let dir = tempfile::tempdir().unwrap();
        let run = |receipts: &PreimageStore| {
            let record = Arc::new(LogCommitmentRecord {
                tree_dir: Some(dir.path().to_path_buf()),
                ..Default::default()
            });
            let native = Arc::new(
                NativeLogCommitments::new(
                    &[chains.batch()],
                    Arc::new(receipts.clone()),
                    record.clone(),
                )
                .unwrap(),
            );
            assert!(chains.resolve(chains.withheld.clone(), native, 1));
            record.finish().unwrap().openings
        };
        let built = run(&chains.full);
        assert_eq!(std::fs::read_dir(dir.path()).unwrap().count(), chains.below_head.len());
        // The withheld oracle has no receipts of a block below the head, so the trees come from
        // the directory.
        assert_eq!(run(&chains.withheld), built);
    }

    /// Pins at most [`MAX_CACHED_TREES`] trees, evicting the least recently used.
    #[test]
    fn tree_cache_is_bounded() {
        let mut cache = TreeCache::default();
        let tree = Arc::new(LogTree::new(vec![B256::ZERO]));
        let roots: Vec<_> = (0..=MAX_CACHED_TREES as u8).map(B256::repeat_byte).collect();
        for root in &roots[..MAX_CACHED_TREES] {
            cache.insert(*root, tree.clone());
        }
        assert!(cache.get(&roots[0]).is_some());
        cache.insert(roots[MAX_CACHED_TREES], tree);
        assert_eq!(cache.0.len(), MAX_CACHED_TREES);
        assert!(cache.get(&roots[1]).is_none(), "the least recently used tree stays");
        assert!(cache.get(&roots[0]).is_some());
    }
}
