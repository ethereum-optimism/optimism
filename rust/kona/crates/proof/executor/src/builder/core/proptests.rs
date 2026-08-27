//! Model-based property tests for SDM post-exec verification through [`StatelessL2Builder`].
//!
//! Arbitrary post-exec payloads are appended to the fixture block and executed through the
//! fault-proof integration path (structural parsing plus block execution). Accept/reject decisions
//! are compared against [`model_accepts`], an independent restatement of the consensus rules, so a
//! divergence between this integration layer and the specified rules fails the test.

use crate::{
    BlockBuildingOutcome, ExecutorResult, StatelessL2Builder,
    test_utils::{DiskTrieNodeProvider, ExecutorTestFixture},
};
use alloy_eips::Encodable2718;
use alloy_op_evm::OpEvmFactory;
use alloy_primitives::{B256, Sealable};
use kona_mpt::NoopTrieHinter;
use op_alloy_consensus::{OpReceiptEnvelope, SDMGasEntry, build_post_exec_tx};
use proptest::prelude::*;
use rocksdb::{DB, Options};
use std::{
    collections::HashSet,
    path::{Path, PathBuf},
    process::Command,
    sync::OnceLock,
};

/// The fixture block, untarred once, plus its baseline (no post-exec tx) execution outcome.
struct FixtureContext {
    baseline: BaselineExecution,
    fixture: ExecutorTestFixture,
    kv_path: PathBuf,

    /// Keeps the untarred fixture directory alive for the lifetime of the tests.
    _fixture_dir: tempfile::TempDir,
}

/// Facts about the fixture block established by executing it without a post-exec transaction.
struct BaselineExecution {
    block_number: u64,
    gas_used: u64,
    state_root: B256,
    transactions: Vec<TransactionProfile>,
}

#[derive(Clone, Copy, Debug)]
struct TransactionProfile {
    evm_gas_used: u64,
    is_deposit: bool,
}

/// One generated verification scenario: an SDM activation state and a post-exec payload to append
/// to the fixture block as its trailing `0x7D` transaction.
#[derive(Clone, Debug)]
struct VerificationCase {
    entries: Vec<SDMGasEntry>,
    payload_block_number: u64,
    sdm_active_override: Option<bool>,
}

fn fixture_context() -> &'static FixtureContext {
    static CONTEXT: OnceLock<FixtureContext> = OnceLock::new();
    CONTEXT.get_or_init(|| {
        let fixture_path =
            PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("testdata/block-26207960.tar.gz");
        let fixture_dir = tempfile::tempdir().expect("create fixture dir");
        let untar_status = Command::new("tar")
            .arg("-xf")
            .arg(&fixture_path)
            .arg("-C")
            .arg(fixture_dir.path())
            .arg("--strip-components=1")
            .status()
            .expect("untar fixture");
        assert!(untar_status.success(), "failed to untar fixture at {fixture_path:?}");

        let fixture: ExecutorTestFixture = serde_json::from_slice(
            &std::fs::read(fixture_dir.path().join("fixture.json")).expect("read fixture.json"),
        )
        .expect("deserialize fixture");
        let kv_path = fixture_dir.path().join("kv");

        let baseline_outcome =
            execute_fixture_block(&fixture, &kv_path, None, None).expect("baseline executes");
        let mut previous_cumulative_gas = 0;
        let transactions = baseline_outcome
            .execution_result
            .receipts
            .iter()
            .map(|receipt| {
                let cumulative_gas = receipt.cumulative_gas_used();
                let profile = TransactionProfile {
                    evm_gas_used: cumulative_gas - previous_cumulative_gas,
                    is_deposit: matches!(receipt, OpReceiptEnvelope::Deposit(_)),
                };
                previous_cumulative_gas = cumulative_gas;
                profile
            })
            .collect();

        let baseline = BaselineExecution {
            block_number: fixture.parent_header.number + 1,
            gas_used: baseline_outcome.execution_result.gas_used,
            state_root: baseline_outcome.header.state_root,
            transactions,
        };
        FixtureContext { baseline, fixture, kv_path, _fixture_dir: fixture_dir }
    })
}

/// Executes the fixture block, optionally appending a trailing post-exec transaction built from
/// the given block number and refund entries.
fn execute_fixture_block(
    fixture: &ExecutorTestFixture,
    kv_path: &Path,
    sdm_active_override: Option<bool>,
    post_exec_tx: Option<(u64, Vec<SDMGasEntry>)>,
) -> ExecutorResult<BlockBuildingOutcome<OpReceiptEnvelope>> {
    let mut options = Options::default();
    options.set_compression_type(rocksdb::DBCompressionType::Snappy);
    let kv_store = DB::open_for_read_only(&options, kv_path, false).expect("open fixture kv store");
    let provider = DiskTrieNodeProvider::new(kv_store);

    let mut attrs = fixture.executing_payload.clone();
    if let Some((block_number, entries)) = post_exec_tx {
        let tx = build_post_exec_tx(block_number, entries);
        let mut encoded = Vec::with_capacity(tx.eip2718_encoded_length());
        tx.encode_2718(&mut encoded);
        attrs.transactions.as_mut().expect("fixture carries transactions").push(encoded.into());
    }

    let mut executor = StatelessL2Builder::new(
        &fixture.rollup_config,
        OpEvmFactory::<alloy_op_evm::OpTx>::default(),
        alloy_op_evm::block::OpAlloyReceiptBuilder::default(),
        provider,
        NoopTrieHinter,
        fixture.parent_header.clone().seal_slow(),
    );
    executor.set_sdm_active_override(sdm_active_override);
    executor.build_block(attrs)
}

/// Independent restatement of the consensus acceptance rules for a block whose trailing `0x7D`
/// transaction carries `case`'s payload: SDM must be active, the payload must anchor to the
/// containing block number, and every entry must carry a non-zero refund, target a unique
/// non-deposit, non-post-exec transaction in range, and stay within that transaction's EVM gas.
fn model_accepts(case: &VerificationCase, baseline: &BaselineExecution) -> bool {
    // The fixture chain schedules no SDM activation, so inheriting from the rollup config
    // (`None`) behaves the same as forcing SDM inactive.
    if case.sdm_active_override != Some(true) {
        return false;
    }
    if case.payload_block_number != baseline.block_number {
        return false;
    }

    let mut seen_indices = HashSet::new();
    case.entries.iter().all(|entry| {
        entry.gas_refund > 0 &&
            seen_indices.insert(entry.index) &&
            baseline.transactions.get(entry.index as usize).is_some_and(|profile| {
                !profile.is_deposit && entry.gas_refund <= profile.evm_gas_used
            })
    })
}

/// Generates one refund entry, biased toward valid entries with each violation class
/// (zero refund, refund above EVM gas, deposit target, post-exec/out-of-range target)
/// as a lower-weight arm.
fn refund_entry_strategy() -> impl Strategy<Value = SDMGasEntry> {
    let baseline = &fixture_context().baseline;
    let refundable: Vec<(u64, u64)> = baseline
        .transactions
        .iter()
        .enumerate()
        .filter(|(_, profile)| !profile.is_deposit)
        .map(|(index, profile)| (index as u64, profile.evm_gas_used))
        .collect();
    let deposit_indices: Vec<u64> = baseline
        .transactions
        .iter()
        .enumerate()
        .filter(|(_, profile)| profile.is_deposit)
        .map(|(index, _)| index as u64)
        .collect();
    let transaction_count = baseline.transactions.len() as u64;

    prop_oneof![
        10 => prop::sample::select(refundable.clone()).prop_flat_map(|(index, evm_gas_used)| {
            (1..=evm_gas_used).prop_map(move |gas_refund| SDMGasEntry { index, gas_refund })
        }),
        1 => prop::sample::select(refundable.clone())
            .prop_map(|(index, _)| SDMGasEntry { index, gas_refund: 0 }),
        1 => prop::sample::select(refundable).prop_flat_map(|(index, evm_gas_used)| {
            prop_oneof![Just(evm_gas_used + 1), Just(u64::MAX)]
                .prop_map(move |gas_refund| SDMGasEntry { index, gas_refund })
        }),
        1 => prop::sample::select(deposit_indices)
            .prop_map(|index| SDMGasEntry { index, gas_refund: 1 }),
        // The appended post-exec tx sits at index `transaction_count`; anything beyond it stays
        // unconsumed. Both must reject.
        1 => (transaction_count..transaction_count + 3, 1..1_000u64)
            .prop_map(|(index, gas_refund)| SDMGasEntry { index, gas_refund }),
    ]
}

fn verification_case_strategy() -> impl Strategy<Value = VerificationCase> {
    let block_number = fixture_context().baseline.block_number;
    (
        prop::collection::vec(refund_entry_strategy(), 0..8),
        prop_oneof![
            8 => Just(block_number),
            1 => Just(block_number - 1),
            1 => Just(block_number + 1),
        ],
        prop_oneof![8 => Just(Some(true)), 1 => Just(Some(false)), 1 => Just(None)],
    )
        .prop_map(|(entries, payload_block_number, sdm_active_override)| VerificationCase {
            entries,
            payload_block_number,
            sdm_active_override,
        })
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(256))]

    #[test]
    fn post_exec_verification_matches_consensus_model(case in verification_case_strategy()) {
        let context = fixture_context();
        let expected_accept = model_accepts(&case, &context.baseline);
        let result = execute_fixture_block(
            &context.fixture,
            &context.kv_path,
            case.sdm_active_override,
            Some((case.payload_block_number, case.entries.clone())),
        );

        match result {
            Ok(outcome) => {
                prop_assert!(
                    expected_accept,
                    "executor accepted a payload the consensus model rejects: {:?}",
                    case,
                );

                let total_refund: u64 = case.entries.iter().map(|entry| entry.gas_refund).sum();
                prop_assert_eq!(
                    outcome.execution_result.gas_used,
                    context.baseline.gas_used - total_refund,
                    "canonical block gas must drop by exactly the refunded total",
                );
                prop_assert_eq!(
                    outcome.execution_result.receipts.len(),
                    context.baseline.transactions.len() + 1,
                );
                prop_assert!(matches!(
                    outcome.execution_result.receipts.last(),
                    Some(OpReceiptEnvelope::PostExec(_))
                ));
                if total_refund == 0 {
                    prop_assert_eq!(outcome.header.state_root, context.baseline.state_root);
                }
            }
            Err(err) => {
                prop_assert!(
                    !expected_accept,
                    "executor rejected a payload the consensus model accepts: {:?}, error: {}",
                    case,
                    err,
                );
                let message = err.to_string().to_lowercase();
                prop_assert!(
                    message.contains("invalid post-exec payload"),
                    "rejection must surface as a post-exec validation failure, got: {}",
                    err,
                );
            }
        }
    }
}
