//! Runs the SP1 consolidation with log commitments, natively and through the super-range ELF.
//! ELF tests skip when `KONA_SP1_ELF_DIR` is unset; build the ELFs with
//! `just build-elfs-native test-config-fallback` in `rust/kona/sp1`.

use std::{
    collections::BTreeSet,
    path::PathBuf,
    sync::{Arc, Mutex},
};

use alloy_chains::Chain;
use alloy_consensus::{EMPTY_ROOT_HASH, Header, Receipt, ReceiptWithBloom};
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::{Address, B256, Bytes, Log, LogData, U256, keccak256};
use alloy_rlp::{EMPTY_STRING_CODE, Encodable};
use kona_genesis::RollupConfig;
use kona_interop::{
    ChainBuilder, ChainDependency, DependencySet, ExecutingMessageBuilder, RawMessagePayload,
};
use kona_preimage::{
    CommsClient, DEPENDENCY_SET_KEY, HintWriterClient, L1_CONFIG_KEY, L2_ROLLUP_CONFIG_KEY,
    PreimageKey, PreimageOracleClient, errors::PreimageOracleResult,
};
use kona_proof::FlushableCache;
use kona_sp1_client_utils::{
    log_commitment::{
        BlockReceipts, LOG_COMMITMENT_ENTRY_LEN, LogCommitmentEntry, LogCommitmentInputs,
        LogOpening, LogTree, VerifiedLogCommitments, log_leaves,
    },
    super_root::{
        SuperConsolidationInputs, SuperConsolidationOutputs, SuperConsolidationTransitionInput,
        SuperOptimisticBlock, SuperOutputRoot, SuperRoot, SuperRootProof, TimestampSpan,
    },
    witness::{BlobData, DefaultWitnessData, WitnessData, preimage_store::PreimageStore},
};
use kona_sp1_elfs::{ELF_DIR_ENV, load_elf_from};
use kona_sp1_ethereum_client_utils::{
    chain_config::ChainConfigs, super_consolidation::build_consolidation_outputs,
};
use kona_sp1_host_utils::witness_generation::PreimageWitnessCollector;
use op_alloy_consensus::OpReceiptEnvelope;
use sp1_sdk::{
    Elf, ExecutionReport, ProveRequest, Prover, ProverClient, ProvingKey, SP1Proof,
    SP1ProofWithPublicValues, SP1PublicValues, SP1Stdin,
};

use crate::{
    BATCH_PROOF_BYTES, ConsolidationCost, LogCommitmentProofs, MAX_ENTRIES_PER_BATCH,
    MAX_STDIN_BYTES, MAX_WITNESS_BYTES, ScannedTimestampCost, bound_receipts_path_pgu,
    build_log_commitment_stdin_from_receipts, build_super_consolidation_stdin,
    decode_super_consolidation_public_values, estimate_consolidation_pgu,
    log_commitments::run_native_consolidation, receipts_trie_bytes,
};

const INITIATING: u64 = u64::MAX - 1;
const EXECUTING: u64 = u64::MAX;
/// Idle chain `i` of a fixture has chain ID `IDLE + i`.
const IDLE: u64 = 1_000;
/// Block `k` of each chain has timestamp `GENESIS_TIME + k`.
const GENESIS_TIME: u64 = 1_000;

/// A one-timestamp consolidation. The initiating chain's blocks `1..=n` sit below its head
/// `n + 1`; the executing chain's head executes messages into them; idle chains have empty blocks.
struct Fixture {
    /// Every preimage of the consolidation except the receipts tries of blocks `1..=n`.
    withheld: PreimageStore,
    /// The receipts of blocks `1..=n`.
    below_head: Vec<BlockReceipts>,
    /// The `(receipts root, log index)` of each message.
    references: Vec<(B256, u64)>,
    inputs: SuperConsolidationInputs,
    configs: ChainConfigs,
    chains: u64,
    messages: u64,
    /// [`receipts_trie_bytes`] of the executing chain's head.
    head_receipts_bytes: u64,
}

impl Fixture {
    /// `messages` are `(block number, log index)` pairs into `blocks`, the receipts of blocks
    /// `1..=n`. Each message carries the origin and payload hash of the log it references.
    fn new(blocks: Vec<Vec<OpReceiptEnvelope>>, messages: &[(u64, u64)]) -> Self {
        Self::with_idle_chains(blocks, messages, 0)
    }

    fn with_idle_chains(
        blocks: Vec<Vec<OpReceiptEnvelope>>,
        messages: &[(u64, u64)],
        idle_chains: u64,
    ) -> Self {
        let mut witness = PreimageStore::default();
        save(&mut witness, vec![EMPTY_STRING_CODE]);
        let configs = configs(idle_chains);
        for (key, value) in [
            (DEPENDENCY_SET_KEY, serde_json::to_vec(&configs.dependency_set).unwrap()),
            (L2_ROLLUP_CONFIG_KEY, serde_json::to_vec(&configs.rollup_configs).unwrap()),
            (L1_CONFIG_KEY, serde_json::to_vec(&configs.l1_config).unwrap()),
        ] {
            witness.save_preimage(PreimageKey::new_local(key.to()), value).unwrap();
        }

        let mut executing = ChainBuilder::default();
        for (number, index) in messages {
            let log = blocks[*number as usize - 1]
                .iter()
                .flat_map(OpReceiptEnvelope::logs)
                .nth(*index as usize);
            let log = log.expect("message references an existing log");
            executing.add_executing_message(
                ExecutingMessageBuilder::default()
                    .with_message_hash(keccak256(RawMessagePayload::from(log).as_ref()))
                    .with_origin_address(log.address)
                    .with_origin_chain_id(INITIATING)
                    .with_origin_block_number(*number)
                    .with_origin_log_index(*index)
                    .with_origin_timestamp(GENESIS_TIME + number),
            );
        }

        let head_number = blocks.len() as u64 + 1;
        let mut parent = B256::ZERO;
        let mut below_head = Vec::new();
        let mut previous_initiating = B256::ZERO;
        for (number, receipts) in (1..).zip(blocks) {
            let receipts_root = receipts_root(&receipts);
            parent = save_header(&mut witness, INITIATING, number, parent, receipts_root);
            previous_initiating = parent;
            below_head.push(BlockReceipts { receipts_root, receipts });
        }
        let initiating_head =
            save_header(&mut witness, INITIATING, head_number, parent, EMPTY_ROOT_HASH);
        let previous_executing =
            save_header(&mut witness, EXECUTING, head_number - 1, B256::ZERO, EMPTY_ROOT_HASH);
        let executing_receipts = save_receipts(&mut witness, &executing.receipts);
        let executing_head = save_header(
            &mut witness,
            EXECUTING,
            head_number,
            previous_executing,
            executing_receipts,
        );

        let mut chains: Vec<(u64, B256, B256)> = (1..=idle_chains)
            .map(|index| {
                let chain_id = IDLE + index;
                let previous = save_header(
                    &mut witness,
                    chain_id,
                    head_number - 1,
                    B256::ZERO,
                    EMPTY_ROOT_HASH,
                );
                let head =
                    save_header(&mut witness, chain_id, head_number, previous, EMPTY_ROOT_HASH);
                (chain_id, previous, head)
            })
            .collect();
        chains.push((INITIATING, previous_initiating, initiating_head));
        chains.push((EXECUTING, previous_executing, executing_head));
        let previous_outputs = chains
            .iter()
            .map(|(chain_id, previous, _)| SuperOutputRoot {
                chain_id: *chain_id,
                output_root: save_output_root(&mut witness, *previous),
            })
            .collect();
        let heads: Vec<_> = chains
            .iter()
            .map(|(chain_id, _, head)| SuperOptimisticBlock {
                chain_id: U256::from(*chain_id),
                block_hash: *head,
                output_root: save_output_root(&mut witness, *head),
            })
            .collect();
        let previous_super_root = SuperRoot::new(GENESIS_TIME + head_number - 1, previous_outputs);
        let mut encoded = Vec::new();
        previous_super_root.encode(&mut encoded);
        let previous_super_root = save(&mut witness, encoded);
        let timestamp = GENESIS_TIME + head_number;
        let inputs = SuperConsolidationInputs {
            span: TimestampSpan::new(timestamp, timestamp).unwrap(),
            previous_super_root,
            transitions: vec![SuperConsolidationTransitionInput {
                claimed_super_root_proof: SuperRootProof::new(
                    timestamp,
                    heads
                        .iter()
                        .map(|head| SuperOutputRoot {
                            chain_id: head.chain_id.to(),
                            output_root: head.output_root,
                        })
                        .collect(),
                ),
                optimistic_blocks: heads,
            }],
        };
        let references = messages
            .iter()
            .map(|(number, index)| (below_head[*number as usize - 1].receipts_root, *index))
            .collect();
        Self {
            withheld: witness,
            below_head,
            references,
            inputs,
            configs,
            chains: idle_chains + 2,
            messages: messages.len() as u64,
            head_receipts_bytes: receipts_trie_bytes(&executing.receipts),
        }
    }

    /// Every preimage of the consolidation.
    fn full_witness(&self) -> PreimageStore {
        let mut witness = self.withheld.clone();
        for block in &self.below_head {
            save_receipts(&mut witness, &block.receipts);
        }
        witness
    }

    /// Saves the configs that only the test guest reads into `witness`, and returns their bytes.
    async fn save_configs(&self, witness: &mut PreimageStore) -> u64 {
        let mut bytes = 0;
        for key in [DEPENDENCY_SET_KEY, L2_ROLLUP_CONFIG_KEY, L1_CONFIG_KEY] {
            let key = PreimageKey::new_local(key.to());
            let config = self.withheld.get(key).await.unwrap();
            bytes += config.len() as u64;
            witness.save_preimage(key, config).unwrap();
        }
        bytes
    }

    /// One batch with the entry of every block below the head, computed natively.
    fn native_batch(&self) -> Vec<u8> {
        self.below_head
            .iter()
            .flat_map(|BlockReceipts { receipts_root, receipts }| {
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

    /// The opening of each referenced log, once per log.
    fn openings(&self) -> Vec<LogOpening> {
        let references: BTreeSet<_> = self.references.iter().copied().collect();
        references
            .into_iter()
            .map(|(receipts_root, index)| {
                let block = self
                    .below_head
                    .iter()
                    .find(|block| block.receipts_root == receipts_root)
                    .unwrap();
                LogTree::new(log_leaves(&block.receipts)).opening(receipts_root, index)
            })
            .collect()
    }

    fn log_commitments(&self) -> LogCommitmentInputs {
        LogCommitmentInputs { batches: vec![self.native_batch()], openings: self.openings() }
    }

    /// Runs the consolidation natively, through the guest's source and the witness oracle.
    fn run_native(
        &self,
        witness: PreimageStore,
        log_commitments: LogCommitmentInputs,
        verify: impl FnMut(&[u32; 8], &[u8; 32]),
    ) -> SuperConsolidationOutputs {
        let log_commitments =
            Arc::new(VerifiedLogCommitments::new(log_commitments, &[0; 8], verify));
        let inputs = self.inputs.clone();
        let configs = &self.configs;
        kona_proof::block_on(async move {
            let (oracle, _) = DefaultWitnessData::from_parts(witness, BlobData::default())
                .get_oracle_and_blob_provider()
                .await
                .unwrap();
            build_consolidation_outputs(inputs, oracle, Some(configs), log_commitments).await
        })
        .expect("consolidation succeeds")
    }

    fn stdin(&self, witness: PreimageStore, log_commitments: &LogCommitmentInputs) -> SP1Stdin {
        build_super_consolidation_stdin(
            &self.inputs,
            DefaultWitnessData::from_parts(witness, BlobData::default()),
            log_commitments,
            None,
        )
        .unwrap()
    }
}

fn configs(idle_chains: u64) -> ChainConfigs {
    let rollup_config = |chain_id: u64| {
        let mut config = RollupConfig {
            block_time: 1,
            l1_chain_id: 1,
            l2_chain_id: Chain::from(chain_id),
            ..Default::default()
        };
        config.genesis.l2_time = GENESIS_TIME;
        config.hardforks.lagoon_time = Some(0);
        (chain_id, config)
    };
    let chain_ids: Vec<u64> =
        (1..=idle_chains).map(|index| IDLE + index).chain([INITIATING, EXECUTING]).collect();
    ChainConfigs {
        dependency_set: DependencySet {
            dependencies: chain_ids
                .iter()
                .map(|chain_id| (*chain_id, ChainDependency {}))
                .collect(),
            override_message_expiry_window: None,
        },
        rollup_configs: chain_ids.iter().map(|chain_id| rollup_config(*chain_id)).collect(),
        l1_config: kona_registry::L1_CONFIGS[&1].clone(),
    }
}

fn save(witness: &mut PreimageStore, preimage: Vec<u8>) -> B256 {
    let hash = keccak256(&preimage);
    witness.save_preimage(PreimageKey::new_keccak256(*hash), preimage).unwrap();
    hash
}

fn receipts_root(receipts: &[OpReceiptEnvelope]) -> B256 {
    alloy_consensus::proofs::ordered_trie_root_with_encoder(receipts, |receipt, out| {
        receipt.encode_2718(out)
    })
}

fn save_receipts(witness: &mut PreimageStore, receipts: &[OpReceiptEnvelope]) -> B256 {
    let mut trie =
        kona_mpt::ordered_trie_with_encoder(receipts, |receipt, out| receipt.encode_2718(out));
    let root = trie.root();
    for node in trie.take_proof_nodes().into_inner().into_values() {
        save(witness, node.to_vec());
    }
    root
}

fn save_header(
    witness: &mut PreimageStore,
    chain_id: u64,
    number: u64,
    parent: B256,
    receipts: B256,
) -> B256 {
    let header = Header {
        beneficiary: Address::left_padding_from(&chain_id.to_be_bytes()),
        number,
        timestamp: GENESIS_TIME + number,
        parent_hash: parent,
        receipts_root: receipts,
        transactions_root: EMPTY_ROOT_HASH,
        ..Default::default()
    };
    let mut rlp = Vec::new();
    header.encode(&mut rlp);
    save(witness, rlp)
}

/// A version-0 output root preimage that commits to `block_hash`.
fn save_output_root(witness: &mut PreimageStore, block_hash: B256) -> B256 {
    let mut preimage = [0u8; 128];
    preimage[96..].copy_from_slice(block_hash.as_slice());
    save(witness, preimage.to_vec())
}

/// `count` logs with distinct addresses, `topics` topics, and `data` data bytes each.
fn logs(first_seed: u64, count: u64, topics: usize, data: usize) -> Vec<Log> {
    (first_seed..first_seed + count)
        .map(|seed| Log {
            address: Address::left_padding_from(&seed.to_be_bytes()),
            data: LogData::new_unchecked(
                vec![B256::repeat_byte(0x11); topics],
                Bytes::from(vec![seed as u8; data]),
            ),
        })
        .collect()
}

fn receipt(logs: Vec<Log>) -> OpReceiptEnvelope {
    OpReceiptEnvelope::Eip1559(ReceiptWithBloom {
        receipt: Receipt { status: true.into(), cumulative_gas_used: 21_000, logs },
        logs_bloom: Default::default(),
    })
}

/// Three blocks below the head with messages into each, two of them into one log.
fn small_fixture() -> Fixture {
    Fixture::new(
        vec![
            vec![receipt(logs(0, 2, 1, 8)), receipt(vec![]), receipt(logs(2, 3, 2, 40))],
            vec![receipt(logs(10, 1, 0, 0))],
            (0..20).map(|i| receipt(logs(100 + i, 1, 4, 16))).collect(),
        ],
        &[(1, 0), (1, 4), (2, 0), (3, 19), (3, 7), (1, 4)],
    )
}

/// Pins equal outputs through the table, with the served receipts withheld, and through receipts.
#[test]
fn table_serves_below_head_references_without_their_receipts() {
    let fixture = small_fixture();
    let through_receipts =
        fixture.run_native(fixture.full_witness(), LogCommitmentInputs::default(), |_, _| {});
    let through_table =
        fixture.run_native(fixture.withheld.clone(), fixture.log_commitments(), |_, _| {});
    assert_eq!(through_table, through_receipts);
}

/// Pins today's path for a below-head block without an entry.
#[test]
fn block_without_entry_reads_its_receipts() {
    let fixture = small_fixture();
    let expected =
        fixture.run_native(fixture.full_witness(), LogCommitmentInputs::default(), |_, _| {});
    let mut partial = fixture.log_commitments();
    partial.batches = vec![partial.batches[0][..2 * LOG_COMMITMENT_ENTRY_LEN].to_vec()];
    partial.openings.retain(|opening| opening.receipts_root != fixture.below_head[2].receipts_root);
    let mut witness = fixture.withheld.clone();
    save_receipts(&mut witness, &fixture.below_head[2].receipts);
    assert_eq!(fixture.run_native(witness, partial, |_, _| {}), expected);
}

/// The native run over a collector with `host` as its host oracle, and the collected witness.
async fn collect<H>(
    fixture: &Fixture,
    host: H,
    batches: &[Vec<u8>],
) -> (anyhow::Result<crate::log_commitments::NativeConsolidation>, PreimageStore)
where
    H: CommsClient + FlushableCache + std::fmt::Debug + Send + Sync + Clone + 'static,
{
    let witness = Arc::new(Mutex::new(PreimageStore::default()));
    let collector = Arc::new(PreimageWitnessCollector {
        preimage_oracle: Arc::new(host),
        preimage_witness_store: witness.clone(),
    });
    let native = run_native_consolidation(
        collector,
        fixture.inputs.clone(),
        Some(&fixture.configs),
        batches,
    )
    .await;
    let witness = witness.lock().unwrap().clone();
    (native, witness)
}

/// Pins a collected witness without served receipts that replays to the native outputs.
#[tokio::test(flavor = "multi_thread")]
async fn collected_witness_omits_served_receipts() {
    let fixture = small_fixture();
    let batches = vec![fixture.native_batch()];
    let (native, witness) = collect(&fixture, fixture.full_witness(), &batches).await;
    let native = native.unwrap();
    assert!(native.missing.is_empty());
    assert_eq!(native.openings, fixture.openings());
    assert_eq!(native.witness_bytes, witness.preimage_bytes() as u64);
    for block in &fixture.below_head {
        let mut trie = kona_mpt::ordered_trie_with_encoder(&block.receipts, |receipt, out| {
            receipt.encode_2718(out)
        });
        trie.root();
        for node in trie.take_proof_nodes().into_inner().into_values() {
            let key = PreimageKey::new_keccak256(*keccak256(&node));
            assert!(witness.get(key).await.is_err(), "served receipts trie node {key} collected");
        }
    }

    let log_commitments = LogCommitmentInputs { batches, openings: native.openings };
    let replayed = fixture.run_native(witness, log_commitments, |_, _| {});
    assert_eq!(replayed, native.outputs);
}

/// Pins the blocks without an entry, once each.
#[tokio::test(flavor = "multi_thread")]
async fn collection_reports_missing_blocks() {
    let fixture = small_fixture();
    let (native, _) = collect(&fixture, fixture.full_witness(), &[]).await;
    let missing: Vec<B256> =
        native.unwrap().missing.iter().map(|block| block.receipts_root).collect();
    let mut expected: Vec<B256> =
        fixture.below_head.iter().map(|block| block.receipts_root).collect();
    expected.sort();
    assert_eq!(missing, expected);
}

/// A host oracle that suspends once before each read and records the keys it reads.
#[derive(Debug, Clone)]
struct SuspendingHost {
    store: PreimageStore,
    reads: Arc<Mutex<Vec<PreimageKey>>>,
}

/// Returns `Pending` on its first poll.
#[derive(Default)]
struct SuspendOnce(bool);

impl std::future::Future for SuspendOnce {
    type Output = ();

    fn poll(
        mut self: std::pin::Pin<&mut Self>,
        cx: &mut std::task::Context<'_>,
    ) -> std::task::Poll<()> {
        if self.0 {
            return std::task::Poll::Ready(());
        }
        self.0 = true;
        cx.waker().wake_by_ref();
        std::task::Poll::Pending
    }
}

#[async_trait::async_trait]
impl PreimageOracleClient for SuspendingHost {
    async fn get(&self, key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
        SuspendOnce::default().await;
        self.reads.lock().unwrap().push(key);
        self.store.get(key).await
    }

    async fn get_exact(&self, key: PreimageKey, buf: &mut [u8]) -> PreimageOracleResult<()> {
        SuspendOnce::default().await;
        self.reads.lock().unwrap().push(key);
        self.store.get_exact(key, buf).await
    }
}

#[async_trait::async_trait]
impl HintWriterClient for SuspendingHost {
    async fn write(&self, hint: &str) -> PreimageOracleResult<()> {
        self.store.write(hint).await
    }
}

impl FlushableCache for SuspendingHost {
    fn flush(&self) {}
}

/// Pins the failing host read as the run's last host read.
#[tokio::test(flavor = "multi_thread")]
async fn host_failure_stops_at_the_next_host_read() {
    let fixture = small_fixture();
    let mut store = fixture.withheld.clone();
    for block in &fixture.below_head[1..] {
        save_receipts(&mut store, &block.receipts);
    }
    let reads = Arc::new(Mutex::new(Vec::new()));
    let host = SuspendingHost { store, reads: reads.clone() };
    let (native, _) = collect(&fixture, host, &[fixture.native_batch()]).await;
    let err = native.expect_err("a host failure must fail the collection");
    assert!(format!("{err:#}").contains("failed to open a committed log"), "{err:#}");
    // Messages are checked in initiating-block order, so block 1 fails first.
    let failing = PreimageKey::new_keccak256(*fixture.below_head[0].receipts_root);
    let reads = reads.lock().unwrap();
    assert_eq!(reads.last(), Some(&failing), "{} reads after the failure", {
        reads.iter().rev().position(|key| *key == failing).unwrap_or(reads.len())
    });
}

/// Pins a collection error, not outputs, when the host cannot serve a served block's receipts.
#[tokio::test(flavor = "multi_thread")]
async fn collection_fails_on_host_failure() {
    let fixture = small_fixture();
    let (native, _) = collect(&fixture, fixture.withheld.clone(), &[fixture.native_batch()]).await;
    let err = native.expect_err("a host failure must fail the collection");
    assert!(format!("{err:#}").contains("failed to open a committed log"), "{err:#}");
}

fn run_with(edit: impl FnOnce(&Fixture, &mut LogCommitmentInputs)) {
    let fixture = small_fixture();
    let mut log_commitments = fixture.log_commitments();
    edit(&fixture, &mut log_commitments);
    fixture.run_native(fixture.withheld.clone(), log_commitments, |_, _| {});
}

#[test]
#[should_panic(expected = "no opening of log 19")]
fn halts_on_missing_opening() {
    run_with(|_, log_commitments| log_commitments.openings.retain(|opening| opening.index != 19));
}

#[test]
#[should_panic(expected = "SiblingCount")]
fn halts_on_wrong_sibling_count() {
    run_with(|_, log_commitments| {
        log_commitments.openings[0].siblings.push(B256::ZERO);
    });
}

#[test]
#[should_panic(expected = "RootMismatch")]
fn halts_on_path_that_does_not_give_the_root() {
    run_with(|_, log_commitments| {
        log_commitments.openings[0].siblings[0] = B256::repeat_byte(1);
    });
}

#[test]
#[should_panic(expected = "two log-commitment entries for receipts root")]
fn halts_on_two_entries_for_one_receipts_root() {
    run_with(|fixture, log_commitments| {
        let conflicting = LogCommitmentEntry {
            receipts_root: fixture.below_head[0].receipts_root,
            log_count: 0,
            log_root: B256::ZERO,
        };
        log_commitments.batches.push(conflicting.encode().to_vec());
    });
}

#[test]
#[should_panic(expected = "batch proof does not verify")]
fn halts_on_batch_that_does_not_verify() {
    let fixture = small_fixture();
    fixture.run_native(fixture.withheld.clone(), fixture.log_commitments(), |_, _| {
        panic!("batch proof does not verify")
    });
}

fn elf_dir() -> Option<PathBuf> {
    std::env::var(ELF_DIR_ENV).ok().map(PathBuf::from)
}

async fn execute(
    dir: &std::path::Path,
    name: &str,
    stdin: SP1Stdin,
    deferred_proof_verification: bool,
) -> (SP1PublicValues, ExecutionReport) {
    let elf = Arc::<[u8]>::from(load_elf_from(dir, name).unwrap());
    let client = ProverClient::builder().cpu().build().await;
    client
        .execute(Elf::Dynamic(elf), stdin)
        .deferred_proof_verification(deferred_proof_verification)
        .calculate_gas(true)
        .await
        .unwrap()
}

/// Executes the super-range ELF in consolidation mode and decodes its outputs.
async fn execute_consolidation(
    dir: &std::path::Path,
    stdin: SP1Stdin,
    deferred_proof_verification: bool,
) -> (SuperConsolidationOutputs, ExecutionReport) {
    let (mut public_values, report) =
        execute(dir, "super-range-elf", stdin, deferred_proof_verification).await;
    assert_eq!(report.exit_code, 0, "super-range guest halted");
    (decode_super_consolidation_public_values(&mut public_values).unwrap(), report)
}

/// The batch that the log-commitment ELF commits for the blocks below the head.
async fn elf_batch(dir: &std::path::Path, fixture: &Fixture) -> Vec<u8> {
    let stdin = build_log_commitment_stdin_from_receipts(&fixture.below_head).unwrap();
    let (public_values, report) = execute(dir, "log-commitment-elf", stdin, true).await;
    assert_eq!(report.exit_code, 0);
    public_values.as_slice().to_vec()
}

/// Pins the ELF's consolidation public values to the native outputs, through receipts.
#[tokio::test(flavor = "multi_thread")]
async fn elf_matches_native_without_entries() {
    let Some(dir) = elf_dir() else { return };
    let fixture = small_fixture();
    let native =
        fixture.run_native(fixture.full_witness(), LogCommitmentInputs::default(), |_, _| {});
    let stdin = fixture.stdin(fixture.full_witness(), &LogCommitmentInputs::default());
    assert_eq!(execute_consolidation(&dir, stdin, true).await.0, native);
}

/// Pins the ELF's outputs through entries from the log-commitment ELF to the native outputs.
#[tokio::test(flavor = "multi_thread")]
async fn elf_matches_native_with_entries() {
    let Some(dir) = elf_dir() else { return };
    let fixture = small_fixture();
    let batch = elf_batch(&dir, &fixture).await;
    assert_eq!(batch, fixture.native_batch(), "both ELFs decode the receipts alike");
    let native =
        fixture.run_native(fixture.full_witness(), LogCommitmentInputs::default(), |_, _| {});
    let log_commitments =
        LogCommitmentInputs { batches: vec![batch], openings: fixture.openings() };
    let stdin = fixture.stdin(fixture.withheld.clone(), &log_commitments);
    assert_eq!(execute_consolidation(&dir, stdin, false).await.0, native);
}

/// Pins the native outputs through a real batch proof, with the deferred-proof check on.
#[tokio::test(flavor = "multi_thread")]
#[ignore = "local CPU proving"]
async fn elf_verifies_real_batch_proof() {
    let dir = elf_dir().expect("set KONA_SP1_ELF_DIR to the built ELF directory");
    let fixture = small_fixture();
    let proofs = prove_batch(&dir, &fixture).await;
    let native =
        fixture.run_native(fixture.full_witness(), LogCommitmentInputs::default(), |_, _| {});
    let log_commitments =
        LogCommitmentInputs { batches: proofs.batches(), openings: fixture.openings() };
    let stdin = build_super_consolidation_stdin(
        &fixture.inputs,
        DefaultWitnessData::from_parts(fixture.withheld.clone(), BlobData::default()),
        &log_commitments,
        Some(&proofs),
    )
    .unwrap();
    assert_eq!(stdin.proofs.len(), 1);
    assert_eq!(execute_consolidation(&dir, stdin, true).await.0, native);
}

/// Pins a proof through a real batch proof, and no proof for entries that differ from it.
#[tokio::test(flavor = "multi_thread")]
#[ignore = "local CPU proving"]
async fn consolidation_proof_binds_batch_proof() {
    let dir = elf_dir().expect("set KONA_SP1_ELF_DIR to the built ELF directory");
    let fixture = small_fixture();
    let proofs = prove_batch(&dir, &fixture).await;
    let native =
        fixture.run_native(fixture.full_witness(), LogCommitmentInputs::default(), |_, _| {});
    let elf = Arc::<[u8]>::from(load_elf_from(&dir, "super-range-elf").unwrap());
    let client = ProverClient::builder().cpu().build().await;
    let pk = client.setup(Elf::Dynamic(elf)).await.unwrap();

    let honest = LogCommitmentInputs { batches: proofs.batches(), openings: fixture.openings() };
    let stdin = build_super_consolidation_stdin(
        &fixture.inputs,
        DefaultWitnessData::from_parts(fixture.withheld.clone(), BlobData::default()),
        &honest,
        Some(&proofs),
    )
    .unwrap();
    let mut proof =
        client.prove(&pk, stdin).compressed().await.expect("honest consolidation proves");
    assert_eq!(decode_super_consolidation_public_values(&mut proof.public_values).unwrap(), native);

    let mut batch = proofs.batches().remove(0);
    // An extra entry the consolidation never reads: the guest runs to completion.
    batch.extend(
        LogCommitmentEntry {
            receipts_root: B256::repeat_byte(0xee),
            log_count: 1,
            log_root: B256::repeat_byte(0xee),
        }
        .encode(),
    );
    let forged = LogCommitmentInputs { batches: vec![batch], openings: fixture.openings() };
    let mut stdin = fixture.stdin(fixture.withheld.clone(), &forged);
    let sp1_sdk::SP1Proof::Compressed(batch_proof) = &proofs.proofs[0].proof else {
        unreachable!()
    };
    stdin.write_proof((**batch_proof).clone(), proofs.vk.vk.clone());
    let err =
        client.prove(&pk, stdin).compressed().await.expect_err(
            "the consolidation proved with entries that its batch proof did not commit",
        );
    println!("RESULT prove error: {err:#}");
}

/// Pins a CPU batch proof and a consolidation proof that verifies with it under the log vkey.
#[tokio::test(flavor = "multi_thread")]
#[ignore = "local CPU proving"]
async fn cpu_proves_batch_and_consolidation() {
    let dir = elf_dir().expect("set KONA_SP1_ELF_DIR to the built ELF directory");
    let fixture = small_fixture();
    let receipts_roots: Vec<B256> =
        fixture.below_head.iter().map(|block| block.receipts_root).collect();
    let expected = crate::expected_batch_public_values(
        &build_log_commitment_stdin_from_receipts(&fixture.below_head).unwrap(),
    );
    let proofs = prove_batch(&dir, &fixture).await;
    assert_eq!(proofs.batches(), [expected]);

    let (native, mut witness) = collect(&fixture, fixture.full_witness(), &proofs.batches()).await;
    let native = native.unwrap();
    assert!(native.missing.is_empty());
    fixture.save_configs(&mut witness).await;
    let mut served = native.served.clone();
    served.sort();
    let mut all = receipts_roots;
    all.sort();
    assert_eq!(served, all);
    let log_commitments =
        LogCommitmentInputs { batches: proofs.batches(), openings: native.openings };
    let stdin = build_super_consolidation_stdin(
        &fixture.inputs,
        DefaultWitnessData::from_parts(witness, BlobData::default()),
        &log_commitments,
        Some(&proofs),
    )
    .unwrap();
    assert_eq!(stdin.proofs.len(), 1);
    assert_eq!(stdin.proofs[0].1, proofs.vk.vk);

    let elf = Arc::<[u8]>::from(load_elf_from(&dir, "super-range-elf").unwrap());
    let client = ProverClient::builder().cpu().build().await;
    let pk = client.setup(Elf::Dynamic(elf)).await.unwrap();
    let mut proof = client.prove(&pk, stdin).compressed().await.unwrap();
    client.verify(&proof, pk.verifying_key(), None).unwrap();
    assert_eq!(
        decode_super_consolidation_public_values(&mut proof.public_values).unwrap(),
        native.outputs
    );
}

async fn prove_batch(dir: &std::path::Path, fixture: &Fixture) -> LogCommitmentProofs {
    let elf = Arc::<[u8]>::from(load_elf_from(dir, "log-commitment-elf").unwrap());
    let client = ProverClient::builder().cpu().build().await;
    let pk = client.setup(Elf::Dynamic(elf)).await.unwrap();
    let stdin = build_log_commitment_stdin_from_receipts(&fixture.below_head).unwrap();
    let proof = client.prove(&pk, stdin).compressed().await.unwrap();
    LogCommitmentProofs { vk: pk.verifying_key().clone(), proofs: vec![proof] }
}

/// Measures `N` messages (default 4) into `N` distinct full 30M-gas blocks below the head.
#[tokio::test(flavor = "multi_thread")]
#[ignore = "manual SP1 cost measurement"]
async fn measure_consolidation_cost() {
    let dir = elf_dir().expect("set KONA_SP1_ELF_DIR to the built ELF directory");
    let n: u64 = std::env::var("N").map_or(4, |n| n.parse().unwrap());
    let blocks: Vec<_> =
        (0..n).map(|block| vec![receipt(logs(block * 10_000, 3_480, 0, 1_024))]).collect();
    let messages: Vec<_> = (1..=n).map(|number| (number, number * 997 % 3_480)).collect();
    for (label, fixture) in [
        ("baseline", Fixture::new(blocks.clone(), &[])),
        ("messages", Fixture::new(blocks, &messages)),
    ] {
        let one_batch = fixture.log_commitments();
        let per_block_batches = LogCommitmentInputs {
            batches: one_batch.batches[0]
                .chunks(LOG_COMMITMENT_ENTRY_LEN)
                .map(<[u8]>::to_vec)
                .collect(),
            openings: one_batch.openings.clone(),
        };
        let modes = [
            ("none", fixture.withheld.clone(), LogCommitmentInputs::default()),
            ("receipts", fixture.full_witness(), LogCommitmentInputs::default()),
            ("one-batch", fixture.withheld.clone(), one_batch),
            ("per-block-batches", fixture.withheld.clone(), per_block_batches),
        ];
        for (mode, witness, log_commitments) in modes {
            if mode == "none" && label == "messages" {
                continue;
            }
            let stdin = fixture.stdin(witness, &log_commitments);
            let (witness_bytes, section_bytes) = (stdin.buffer[1].len(), stdin.buffer[2].len());
            let (_, report) = execute_consolidation(&dir, stdin, false).await;
            println!(
                "RESULT {label} n={n} mode={mode} batches={} witness_bytes={witness_bytes} \
                 section_bytes={section_bytes} instructions={} pgu={:?}",
                log_commitments.batches.len(),
                report.total_instruction_count(),
                report.gas(),
            );
        }
    }
}

/// `batch` split into `count` batches of whole entries.
fn split_batch(batch: &[u8], count: usize) -> Vec<Vec<u8>> {
    let entries = batch.len() / LOG_COMMITMENT_ENTRY_LEN;
    batch.chunks(entries.div_ceil(count) * LOG_COMMITMENT_ENTRY_LEN).map(<[u8]>::to_vec).collect()
}

/// The measured PGU of the consolidation through `batches`, and its estimate from the collection.
///
/// The estimate also counts recursion work, which the executor does not measure.
async fn measure_against_estimate(
    dir: &std::path::Path,
    fixture: &Fixture,
    batches: Vec<Vec<u8>>,
) -> (u64, u64) {
    let (native, mut witness) = collect(fixture, fixture.full_witness(), &batches).await;
    let native = native.unwrap();
    assert!(batches.is_empty() || native.missing.is_empty());
    let config_bytes = fixture.save_configs(&mut witness).await;
    let cost = ConsolidationCost {
        chain_timestamps: fixture.chains,
        message_checks: fixture.messages,
        witness_bytes: witness.preimage_bytes() as u64,
        read_bytes: native.read_bytes + config_bytes,
        log_checks: native.log_checks,
        log_check_siblings: native.log_check_siblings,
        batch_proofs: batches.len() as u64,
        batch_entries: batches
            .iter()
            .map(|batch| (batch.len() / LOG_COMMITMENT_ENTRY_LEN) as u64)
            .sum(),
        replaced_gas: 0,
    };
    let log_commitments = LogCommitmentInputs { batches, openings: native.openings };
    let stdin = fixture.stdin(witness, &log_commitments);
    let (outputs, report) = execute_consolidation(dir, stdin, false).await;
    assert_eq!(outputs, native.outputs);
    (report.gas().expect("gas is calculated"), estimate_consolidation_pgu(&cost))
}

/// Pins a consolidation estimate at or above the measured PGU on both paths for each shape.
#[tokio::test(flavor = "multi_thread")]
async fn consolidation_estimate_bounds_measured_pgu() {
    let Some(dir) = elf_dir() else { return };
    let mixed = vec![
        vec![receipt(logs(0, 300, 1, 1_024))],
        vec![receipt(logs(1_000, 2_000, 0, 0))],
        (0..200).map(|i| receipt(if i == 0 { logs(5_000, 1, 2, 8) } else { vec![] })).collect(),
        vec![receipt(logs(6_000, 1, 0, 20 * 1_024)), receipt(logs(6_001, 3, 4, 0))],
    ];
    let mixed_messages = [(1, 0), (1, 299), (2, 1_999), (3, 0), (4, 0), (4, 3), (4, 3)];
    let many_messages: Vec<_> = (0..3_239).map(|index| (1, index)).collect();
    let deep_trees: Vec<_> =
        (0..4).map(|b| vec![receipt(logs(b * 100_000, 75_000, 0, 0))]).collect();
    let deep_messages: Vec<_> =
        (1..=4).flat_map(|number| [(number, 0), (number, 37_500), (number, 74_999)]).collect();
    let entry_cap = (0..MAX_ENTRIES_PER_BATCH as u64).map(|b| vec![receipt(logs(b, 1, 0, 0))]);
    let shapes = [
        ("mixed, 2 chains", Fixture::with_idle_chains(mixed.clone(), &mixed_messages, 0)),
        ("mixed, 6 chains", Fixture::with_idle_chains(mixed, &mixed_messages, 4)),
        ("3,239 messages", Fixture::new(vec![vec![receipt(logs(0, 3_239, 1, 0))]], &many_messages)),
        ("deep trees", Fixture::new(deep_trees, &deep_messages)),
        (
            "entry cap",
            Fixture::new(entry_cap.collect(), &[(1, 0), (MAX_ENTRIES_PER_BATCH as u64, 0)]),
        ),
    ];
    for (shape, fixture) in shapes {
        let batch = fixture.native_batch();
        let mut runs = vec![vec![], split_batch(&batch, 1)];
        if fixture.below_head.len() >= 4 {
            runs.push(split_batch(&batch, 4));
        }
        for batches in runs {
            let count = batches.len();
            let (measured, estimate) = measure_against_estimate(&dir, &fixture, batches).await;
            println!(
                "RESULT shape={shape:?} batches={count} measured={measured} estimate={estimate}"
            );
            assert!(
                measured <= estimate,
                "{shape}, {count} batches: measured {measured} PGU over estimate {estimate}"
            );
        }
    }
}

/// The least gas of a block with `receipts`: 21,000 per transaction, and 375 per log, 375 per
/// topic, and 8 per data byte.
fn min_gas_used(receipts: &[OpReceiptEnvelope]) -> u64 {
    receipts
        .iter()
        .flat_map(|receipt| {
            receipt
                .logs()
                .iter()
                .map(|log| 375 + 375 * log.topics().len() as u64 + 8 * log.data.data.len() as u64)
        })
        .sum::<u64>() +
        21_000 * receipts.len() as u64
}

/// Pins a receipts-path bound, at each shape's least gas, over the estimate and measured PGU.
#[tokio::test(flavor = "multi_thread")]
async fn receipts_path_bound_bounds_measured_pgu() {
    let Some(dir) = elf_dir() else { return };
    let shapes = [
        ("75,000 empty logs", vec![receipt(logs(0, 75_000, 0, 0))]),
        ("3,480 logs of 1 KB", vec![receipt(logs(0, 3_480, 0, 1_024))]),
        ("36 logs of 100 KB", vec![receipt(logs(0, 36, 0, 100 * 1_024))]),
        ("16,000 logs of 4 topics", vec![receipt(logs(0, 16_000, 4, 0))]),
        (
            "1,428 receipts and one log",
            (0..1_428).map(|i| receipt(if i == 0 { logs(0, 1, 0, 0) } else { vec![] })).collect(),
        ),
    ];
    // The fixture's lookups walk parent headers, so the bound's far walk must stay unused.
    let walks = |referenced_gas_used| {
        bound_receipts_path_pgu(&[ScannedTimestampCost {
            referenced_gas_used,
            ..Default::default()
        }])
    };
    let far_walk = walks([(1, vec![0]), (2, vec![0])].into()) - walks([(1, vec![0, 0])].into());
    for (shape, receipts) in shapes {
        let gas_used = min_gas_used(&receipts);
        let fixture = Fixture::new(vec![receipts], &[(1, 0)]);
        let bound = bound_receipts_path_pgu(&[ScannedTimestampCost {
            chains: fixture.chains,
            messages: fixture.messages,
            head_receipts_bytes: fixture.head_receipts_bytes,
            referenced_gas_used: [(INITIATING, vec![gas_used])].into(),
            ..Default::default()
        }]);
        let (measured, estimate) = measure_against_estimate(&dir, &fixture, vec![]).await;
        println!(
            "RESULT shape={shape:?} gas_used={gas_used} measured={measured} estimate={estimate} \
             bound={bound}"
        );
        assert!(measured <= estimate, "{shape}: measured {measured} PGU over estimate {estimate}");
        assert!(
            estimate + far_walk <= bound,
            "{shape}: estimate {estimate} PGU and a far walk {far_walk} over bound {bound}"
        );
    }
}

/// Measures `client.execute()` of a consolidation at the proposer's stdin caps.
///
/// The stdin has 6 chains, a [`MAX_WITNESS_BYTES`] witness, and as many batch proofs as fit in
/// [`MAX_STDIN_BYTES`]. The first run proves a batch and saves it at `BATCH_PROOF`; a later run
/// executes, which fails on x86-64 Linux when the executor child exceeds SP1's memory limit.
#[tokio::test(flavor = "multi_thread")]
#[ignore = "manual SP1 memory measurement"]
async fn measure_consolidation_memory_at_stdin_cap() {
    let dir = elf_dir().expect("set KONA_SP1_ELF_DIR to the built ELF directory");
    let path = PathBuf::from(std::env::var("BATCH_PROOF").expect("set BATCH_PROOF to a file"));
    if !path.exists() {
        prove_batch(&dir, &small_fixture()).await.proofs[0].save(&path).unwrap();
        println!("RESULT saved a batch proof at {}; run again to measure", path.display());
        return;
    }
    let proof = SP1ProofWithPublicValues::load(&path).unwrap();
    let SP1Proof::Compressed(batch_proof) = &proof.proof else { panic!("not a compressed proof") };
    let vk = {
        let elf = Arc::<[u8]>::from(load_elf_from(&dir, "log-commitment-elf").unwrap());
        let client = ProverClient::builder().cpu().build().await;
        client.setup(Elf::Dynamic(elf)).await.unwrap().verifying_key().vk.clone()
    };

    let count = (MAX_STDIN_BYTES - MAX_WITNESS_BYTES) / BATCH_PROOF_BYTES;
    let blocks = (0..count).map(|block| vec![receipt(logs(block, 1, 1, 32))]).collect();
    let fixture = Fixture::with_idle_chains(blocks, &[(1, 0), (count, 0)], 4);
    let mut witness = fixture.withheld.clone();
    let filler = 1 << 16;
    let fillers = (MAX_WITNESS_BYTES - witness.preimage_bytes() as u64) / (filler + 32);
    for nonce in 0..fillers {
        let mut preimage = vec![0; filler as usize];
        preimage[..8].copy_from_slice(&nonce.to_le_bytes());
        save(&mut witness, preimage);
    }
    let witness_bytes = witness.preimage_bytes();
    let log_commitments = LogCommitmentInputs {
        batches: split_batch(&fixture.native_batch(), count as usize),
        openings: fixture.openings(),
    };
    let mut stdin = fixture.stdin(witness, &log_commitments);
    for _ in 0..count {
        stdin.write_proof((**batch_proof).clone(), vk.clone());
    }
    let buffer_bytes: usize = stdin.buffer.iter().map(Vec::len).sum();
    let (_, report) = execute_consolidation(&dir, stdin, false).await;
    println!(
        "RESULT batch_proofs={count} witness_bytes={witness_bytes} buffer_bytes={buffer_bytes} \
         pgu={:?}",
        report.gas()
    );
}

/// Measures the CPU time of a compressed consolidation proof that carries 1, 2, and 4 real batch
/// proofs; the slope is the recursion time of one deferred proof.
#[tokio::test(flavor = "multi_thread")]
#[ignore = "manual SP1 cost measurement"]
async fn measure_recursion_time_per_deferred_proof() {
    let dir = elf_dir().expect("set KONA_SP1_ELF_DIR to the built ELF directory");
    let blocks = (0..4).map(|block| vec![receipt(logs(block * 10, 4, 1, 32))]).collect();
    let fixture = Fixture::new(blocks, &[(1, 0), (2, 1), (3, 2), (4, 3)]);
    let client = ProverClient::builder().cpu().build().await;
    let setup = async |name| {
        let elf = Arc::<[u8]>::from(load_elf_from(&dir, name).unwrap());
        client.setup(Elf::Dynamic(elf)).await.unwrap()
    };
    let log_pk = setup("log-commitment-elf").await;
    let range_pk = setup("super-range-elf").await;
    for count in [1, 2, 4] {
        let mut proofs = Vec::new();
        for blocks in fixture.below_head.chunks(4 / count) {
            let stdin = build_log_commitment_stdin_from_receipts(blocks).unwrap();
            proofs.push(client.prove(&log_pk, stdin).compressed().await.unwrap());
        }
        let proofs = LogCommitmentProofs { vk: log_pk.verifying_key().clone(), proofs };
        let log_commitments =
            LogCommitmentInputs { batches: proofs.batches(), openings: fixture.openings() };
        let stdin = build_super_consolidation_stdin(
            &fixture.inputs,
            DefaultWitnessData::from_parts(fixture.withheld.clone(), BlobData::default()),
            &log_commitments,
            Some(&proofs),
        )
        .unwrap();
        let started = std::time::Instant::now();
        client.prove(&range_pk, stdin).compressed().await.unwrap();
        println!("RESULT batch_proofs={count} seconds={:.1}", started.elapsed().as_secs_f64());
    }
}
