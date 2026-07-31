use crate::{
    ExecutorError,
    test_utils::{execute_loaded_fixture, load_test_fixture, run_test_fixture},
};
use alloy_consensus::Header;
use alloy_eips::Encodable2718;
use op_alloy_consensus::{OpReceiptEnvelope, SDMGasEntry, build_post_exec_tx};
use rstest::rstest;
use std::path::PathBuf;

/// Path to the fixture used by all post-exec tests.
///
/// The chosen fixture must contain a regular (non-deposit, non-post-exec) tx at index 1, since
/// several tests target that index when constructing payload entries.
fn post_exec_fixture_path() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("testdata/block-26207960.tar.gz")
}

fn fixture_block_number(parent_header: &Header) -> u64 {
    parent_header.number + 1
}

fn append_post_exec_tx(
    transactions: &mut Vec<alloy_primitives::Bytes>,
    block_number: u64,
    gas_refund_entries: Vec<SDMGasEntry>,
) {
    let tx = build_post_exec_tx(block_number, gas_refund_entries);
    let mut encoded = Vec::with_capacity(tx.eip2718_encoded_length());
    tx.encode_2718(&mut encoded);
    transactions.push(encoded.into());
}

/// Asserts that `err` is a post-exec validation failure containing `expected`.
///
/// Matches both the parser-level [`ExecutorError::InvalidPostExecPayload`] and the
/// execution-level `OpBlockExecutionError::InvalidPostExecPayload` wrapped in
/// [`ExecutorError::ExecutionError`], since both render with the substring
/// `"invalid post-exec payload"`.
fn assert_post_exec_validation_failure(err: ExecutorError, expected: &str) {
    let err = err.to_string();
    assert!(err.to_lowercase().contains("invalid post-exec payload"), "unexpected error: {err}");
    assert!(err.contains(expected), "expected {err:?} to contain {expected:?}");
}

#[rstest]
#[tokio::test]
async fn test_statelessly_execute_block(
    #[base_dir = "./testdata"]
    #[files("*.tar.gz")]
    path: PathBuf,
) {
    run_test_fixture(path).await;
}

/// Verifies the default fallthrough: with no override, [`StatelessL2Builder`] consults the
/// rollup config, where SDM is currently unscheduled and reports inactive.
#[tokio::test]
async fn post_exec_sdm_inherit_rejects_post_exec_tx() {
    let mut loaded = load_test_fixture(post_exec_fixture_path()).await;
    let block_number = fixture_block_number(&loaded.fixture.parent_header);
    append_post_exec_tx(
        loaded.fixture.executing_payload.transactions.as_mut().unwrap(),
        block_number,
        Vec::new(),
    );

    let err = execute_loaded_fixture(loaded, None).unwrap_err();
    assert_post_exec_validation_failure(err, "SDM not active");
}

/// Verifies the explicit-override deactivation path. Pairs with
/// [`post_exec_sdm_inherit_rejects_post_exec_tx`] above, which exercises the inherit branch.
#[tokio::test]
async fn post_exec_sdm_forced_inactive_rejects_appended_post_exec_tx() {
    let mut loaded = load_test_fixture(post_exec_fixture_path()).await;
    let block_number = fixture_block_number(&loaded.fixture.parent_header);
    append_post_exec_tx(
        loaded.fixture.executing_payload.transactions.as_mut().unwrap(),
        block_number,
        Vec::new(),
    );

    let err = execute_loaded_fixture(loaded, Some(false)).unwrap_err();
    assert_post_exec_validation_failure(err, "SDM not active");
}

#[tokio::test]
async fn post_exec_sdm_enabled_rejects_wrong_block_number() {
    let mut loaded = load_test_fixture(post_exec_fixture_path()).await;
    let block_number = fixture_block_number(&loaded.fixture.parent_header);
    append_post_exec_tx(
        loaded.fixture.executing_payload.transactions.as_mut().unwrap(),
        block_number + 1,
        Vec::new(),
    );

    let err = execute_loaded_fixture(loaded, Some(true)).unwrap_err();
    assert_post_exec_validation_failure(err, "does not match block number");
}

#[tokio::test]
async fn post_exec_sdm_enabled_rejects_duplicate_post_exec_txs() {
    let mut loaded = load_test_fixture(post_exec_fixture_path()).await;
    let block_number = fixture_block_number(&loaded.fixture.parent_header);
    let transactions = loaded.fixture.executing_payload.transactions.as_mut().unwrap();
    append_post_exec_tx(transactions, block_number, Vec::new());
    append_post_exec_tx(transactions, block_number, Vec::new());

    let err = execute_loaded_fixture(loaded, Some(true)).unwrap_err();
    assert_post_exec_validation_failure(err, "multiple post-exec transactions");
}

#[tokio::test]
async fn post_exec_valid_empty_payload_executes_without_state_or_gas_change() {
    let baseline = execute_loaded_fixture(load_test_fixture(post_exec_fixture_path()).await, None)
        .expect("baseline fixture must execute");

    let mut loaded = load_test_fixture(post_exec_fixture_path()).await;
    let block_number = fixture_block_number(&loaded.fixture.parent_header);
    append_post_exec_tx(
        loaded.fixture.executing_payload.transactions.as_mut().unwrap(),
        block_number,
        Vec::new(),
    );

    let outcome = execute_loaded_fixture(loaded, Some(true)).expect("post-exec fixture executes");
    assert_eq!(
        outcome.execution_result.receipts.len(),
        baseline.execution_result.receipts.len() + 1
    );
    assert!(matches!(
        outcome.execution_result.receipts.last(),
        Some(OpReceiptEnvelope::PostExec(_))
    ));
    assert_eq!(outcome.execution_result.gas_used, baseline.execution_result.gas_used);
    assert_eq!(outcome.header.state_root, baseline.header.state_root);
    assert_ne!(outcome.header.transactions_root, baseline.header.transactions_root);
    assert_ne!(outcome.header.receipts_root, baseline.header.receipts_root);
}

#[tokio::test]
async fn post_exec_payload_rejects_deposit_target() {
    let mut loaded = load_test_fixture(post_exec_fixture_path()).await;
    let block_number = fixture_block_number(&loaded.fixture.parent_header);
    append_post_exec_tx(
        loaded.fixture.executing_payload.transactions.as_mut().unwrap(),
        block_number,
        vec![SDMGasEntry { index: 0, gas_refund: 1 }],
    );

    let err = execute_loaded_fixture(loaded, Some(true)).unwrap_err();
    assert_post_exec_validation_failure(err, "payload entry targets deposit tx index 0");
}

#[tokio::test]
async fn post_exec_payload_rejects_post_exec_target() {
    let mut loaded = load_test_fixture(post_exec_fixture_path()).await;
    let block_number = fixture_block_number(&loaded.fixture.parent_header);
    let post_exec_index =
        loaded.fixture.executing_payload.transactions.as_ref().unwrap().len() as u64;
    append_post_exec_tx(
        loaded.fixture.executing_payload.transactions.as_mut().unwrap(),
        block_number,
        vec![SDMGasEntry { index: post_exec_index, gas_refund: 1 }],
    );

    let err = execute_loaded_fixture(loaded, Some(true)).unwrap_err();
    assert_post_exec_validation_failure(
        err,
        &format!("payload entry targets post-exec tx index {post_exec_index}"),
    );
}

#[tokio::test]
async fn post_exec_payload_rejects_duplicate_entries() {
    let mut loaded = load_test_fixture(post_exec_fixture_path()).await;
    let block_number = fixture_block_number(&loaded.fixture.parent_header);
    append_post_exec_tx(
        loaded.fixture.executing_payload.transactions.as_mut().unwrap(),
        block_number,
        vec![SDMGasEntry { index: 1, gas_refund: 1 }, SDMGasEntry { index: 1, gas_refund: 2 }],
    );

    let err = execute_loaded_fixture(loaded, Some(true)).unwrap_err();
    assert_post_exec_validation_failure(err, "duplicate post-exec payload entry for tx index 1");
}

#[tokio::test]
async fn post_exec_payload_rejects_unconsumed_entry() {
    let mut loaded = load_test_fixture(post_exec_fixture_path()).await;
    let block_number = fixture_block_number(&loaded.fixture.parent_header);
    let out_of_range_index =
        loaded.fixture.executing_payload.transactions.as_ref().unwrap().len() as u64 + 1;
    append_post_exec_tx(
        loaded.fixture.executing_payload.transactions.as_mut().unwrap(),
        block_number,
        vec![SDMGasEntry { index: out_of_range_index, gas_refund: 1 }],
    );

    let err = execute_loaded_fixture(loaded, Some(true)).unwrap_err();
    assert_post_exec_validation_failure(err, "unconsumed post-exec payload entries");
}

#[tokio::test]
async fn post_exec_payload_rejects_refund_exceeding_gas_used() {
    let mut loaded = load_test_fixture(post_exec_fixture_path()).await;
    let block_number = fixture_block_number(&loaded.fixture.parent_header);
    append_post_exec_tx(
        loaded.fixture.executing_payload.transactions.as_mut().unwrap(),
        block_number,
        vec![SDMGasEntry { index: 1, gas_refund: u64::MAX }],
    );

    let err = execute_loaded_fixture(loaded, Some(true)).unwrap_err();
    assert_post_exec_validation_failure(err, "exceeds evm_gas_used");
}
