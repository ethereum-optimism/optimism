//! Log commitments in the native consolidation run that collects the witness.
//!
//! The run uses the guest's entry table and verdict code, but builds each opening from the
//! block's receipts. It reads the receipts through the unrecorded host oracle, so the witness has
//! no receipts trie node of a served block.

use std::{
    collections::BTreeMap,
    fmt::Debug,
    sync::{Arc, Mutex},
};

use alloy_consensus::Header;
use alloy_primitives::{Address, B256};
use anyhow::{Result, anyhow, ensure};
use kona_interop::LogCommitment;
use kona_preimage::CommsClient;
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
}

/// Runs the consolidation natively over `collector`, with the entries of `batches`, the public
/// values of the batch proofs that the stdin will carry.
pub(crate) async fn run_native_consolidation<P>(
    collector: Arc<PreimageWitnessCollector<P>>,
    inputs: SuperConsolidationInputs,
    configs: Option<&ChainConfigs>,
    batches: &[Vec<u8>],
) -> Result<NativeConsolidation>
where
    P: CommsClient + FlushableCache + Debug + Send + Sync + Clone + 'static,
{
    let record = Arc::new(Mutex::new(LogCommitmentRecord::default()));
    let source =
        NativeLogCommitments::new(batches, collector.preimage_oracle.clone(), record.clone())?;
    let outputs = build_consolidation_outputs(inputs, collector, configs, Arc::new(source)).await;
    // Check the record before `outputs`: on a host failure, `log_matches` returns a bogus `false`.
    let (openings, missing) =
        record.lock().map_err(|_| anyhow!("record lock poisoned"))?.finish()?;
    Ok(NativeConsolidation { outputs: outputs?, openings, missing })
}

/// What the native run records, shared by every timestamp and round.
#[derive(Debug, Default)]
struct LogCommitmentRecord {
    /// One per `(receipts root, index)`: a log is checked in each round, at each timestamp, and
    /// for each message into it, and the guest rejects a second opening.
    openings: BTreeMap<(B256, u64), LogOpening>,
    missing: BTreeMap<B256, MissingLogCommitment>,
    trees: BTreeMap<B256, Arc<LogTree>>,
    error: Option<anyhow::Error>,
}

impl LogCommitmentRecord {
    /// The openings and the missing blocks, or the first host failure.
    fn finish(&mut self) -> Result<(Vec<LogOpening>, Vec<MissingLogCommitment>)> {
        if let Some(error) = self.error.take() {
            return Err(error.context("failed to open a committed log from the host's receipts"));
        }
        Ok((
            std::mem::take(&mut self.openings).into_values().collect(),
            std::mem::take(&mut self.missing).into_values().collect(),
        ))
    }
}

/// The [`LogCommitmentSource`] of the native run. A host failure in a log check gives `false` and
/// is recorded; [`LogCommitmentRecord::finish`] then fails the run.
#[derive(Debug)]
struct NativeLogCommitments<O> {
    table: LogCommitmentTable,
    receipts_oracle: Arc<O>,
    record: Arc<Mutex<LogCommitmentRecord>>,
}

impl<O> NativeLogCommitments<O> {
    /// `receipts_oracle` must not record into the witness. The guest verifies the batches.
    fn new(
        batches: &[Vec<u8>],
        receipts_oracle: Arc<O>,
        record: Arc<Mutex<LogCommitmentRecord>>,
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
            self.record
                .lock()
                .expect("record lock")
                .missing
                .entry(receipts_root)
                .or_insert(MissingLogCommitment { chain_id, block_hash, receipts_root });
            return None;
        };
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
    record: Arc<Mutex<LogCommitmentRecord>>,
}

impl<O> NativeBlock<O>
where
    O: CommsClient + Send + Sync,
{
    fn opening(&self, index: u64) -> Result<LogOpening> {
        let receipts_root = self.entry.receipts_root;
        if let Some(opening) =
            self.record.lock().expect("record lock").openings.get(&(receipts_root, index))
        {
            return Ok(opening.clone());
        }
        let opening = self.tree()?.opening(receipts_root, index);
        self.record
            .lock()
            .expect("record lock")
            .openings
            .insert((receipts_root, index), opening.clone());
        Ok(opening)
    }

    fn tree(&self) -> Result<Arc<LogTree>> {
        let receipts_root = self.entry.receipts_root;
        if let Some(tree) = self.record.lock().expect("record lock").trees.get(&receipts_root) {
            return Ok(tree.clone());
        }
        kona_proof::block_on(
            HintType::L2Receipts
                .with_data(&[self.block_hash.as_slice(), &self.chain_id.to_be_bytes()])
                .send(self.receipts_oracle.as_ref()),
        )?;
        let receipts = receipts_from_trie(self.receipts_oracle.as_ref(), receipts_root)
            .map_err(|err| anyhow!("receipts of block {}: {err}", self.block_hash))?;
        let tree = LogTree::new(log_leaves(&receipts));
        ensure!(
            tree.log_count() == self.entry.log_count && tree.root() == self.entry.log_root,
            "receipts of block {} do not give the entry of receipts root {receipts_root}",
            self.block_hash,
        );
        let tree = Arc::new(tree);
        self.record.lock().expect("record lock").trees.insert(receipts_root, tree.clone());
        Ok(tree)
    }
}

impl<O> LogCommitment for NativeBlock<O>
where
    O: CommsClient + Debug + Send + Sync,
{
    fn log_count(&self) -> u64 {
        self.entry.log_count
    }

    fn log_matches(&self, index: u64, origin: &Address, payload_hash: &B256) -> bool {
        match self.opening(index) {
            Ok(opening) => opened_log_matches(&self.entry, &opening, origin, payload_hash),
            Err(error) => {
                self.record.lock().expect("record lock").error.get_or_insert(error);
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

    /// Chain A's blocks 1 and 2 sit below its head, block 3; chain B's head executes two
    /// messages into one log of block 1 and one into block 2.
    struct Chains {
        /// Every preimage except the receipts tries of blocks 1 and 2.
        withheld: PreimageStore,
        full: PreimageStore,
        below_head: [(B256, Vec<OpReceiptEnvelope>); 2],
        heads: HashMap<u64, Sealed<Header>>,
        boot: BootInfo,
    }

    fn chains() -> Chains {
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
        for (number, seed) in [(1, 2), (1, 2), (2, 4)] {
            executing.add_executing_message(
                ExecutingMessageBuilder::default()
                    .with_origin_address(Address::repeat_byte(seed))
                    .with_message_hash(keccak256(RawMessagePayload::from(&log(seed)).as_ref()))
                    .with_origin_chain_id(A)
                    .with_origin_block_number(number)
                    .with_origin_log_index(u64::from(seed) - 1 - 3 * (number - 1))
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

    /// Pins one opening per log across rounds, accepted by the guest's source.
    #[tokio::test(flavor = "multi_thread")]
    async fn records_each_opening_once_for_the_guest() {
        let chains = chains();
        let record = Arc::new(Mutex::new(LogCommitmentRecord::default()));
        let batches = vec![chains.batch()];
        let native = Arc::new(
            NativeLogCommitments::new(&batches, Arc::new(chains.full.clone()), record.clone())
                .unwrap(),
        );
        assert!(chains.resolve(chains.withheld.clone(), native, 2));
        let (openings, missing) = record.lock().unwrap().finish().unwrap();
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
        let record = Arc::new(Mutex::new(LogCommitmentRecord::default()));
        let native = Arc::new(
            NativeLogCommitments::new(&[], Arc::new(chains.full.clone()), record.clone()).unwrap(),
        );
        assert!(chains.resolve(chains.full.clone(), native, 2));
        let (openings, missing) = record.lock().unwrap().finish().unwrap();
        assert!(openings.is_empty());
        let roots: Vec<_> = missing.iter().map(|block| block.receipts_root).collect();
        let mut expected: Vec<_> = chains.below_head.iter().map(|(root, _)| *root).collect();
        expected.sort();
        assert_eq!(roots, expected);
        assert!(missing.iter().all(|block| block.chain_id == A));
    }

    /// Pins an error, not a verdict, when the host cannot serve a served block's receipts.
    #[tokio::test(flavor = "multi_thread")]
    async fn host_failure_fails_the_run() {
        let chains = chains();
        let record = Arc::new(Mutex::new(LogCommitmentRecord::default()));
        let native = Arc::new(
            NativeLogCommitments::new(
                &[chains.batch()],
                Arc::new(chains.withheld.clone()),
                record.clone(),
            )
            .unwrap(),
        );
        chains.resolve(chains.withheld.clone(), native, 1);
        let err = record.lock().unwrap().finish().unwrap_err();
        assert!(format!("{err:#}").contains("receipts of block"), "{err:#}");
    }
}
