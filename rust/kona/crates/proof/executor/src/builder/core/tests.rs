use crate::{
    BlockBuildingOutcome, ExecutorError, StatelessL2Builder,
    test_utils::{
        ExecutorTestFixture, LoadedExecutorTestFixture, execute_loaded_fixture, load_test_fixture,
        run_test_fixture,
    },
};
use alloy_consensus::{Header, Transaction};
use alloy_eips::Encodable2718;
use alloy_op_evm::OpEvmFactory;
use alloy_primitives::{Address, Sealable, U256};
use kona_mpt::NoopTrieHinter;
use op_alloy_consensus::{OpReceiptEnvelope, SDMGasEntry, build_post_exec_tx};
use op_revm::constants::{BASE_FEE_RECIPIENT, OPERATOR_FEE_RECIPIENT};
use rstest::rstest;
use std::{
    collections::{BTreeMap, BTreeSet},
    path::PathBuf,
};

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

/// Executes a fixture and reads selected balances from its resulting state trie.
fn execute_loaded_fixture_with_balances(
    loaded: LoadedExecutorTestFixture,
    sdm_active_override: Option<bool>,
    addresses: &BTreeSet<Address>,
) -> (BlockBuildingOutcome<OpReceiptEnvelope>, BTreeMap<Address, U256>) {
    let LoadedExecutorTestFixture { fixture_dir: _fixture_dir, fixture, provider } = loaded;
    let ExecutorTestFixture { rollup_config, parent_header, executing_payload, .. } = fixture;

    let mut executor = StatelessL2Builder::new(
        &rollup_config,
        OpEvmFactory::<alloy_op_evm::OpTx>::default(),
        alloy_op_evm::block::OpAlloyReceiptBuilder::default(),
        provider,
        NoopTrieHinter,
        parent_header.seal_slow(),
    );
    executor.set_sdm_active_override(sdm_active_override);
    let outcome = executor.build_block(executing_payload).expect("fixture executes");
    let balances = addresses
        .iter()
        .map(|address| {
            let balance = executor
                .trie_db
                .get_trie_account(address)
                .expect("account proof is available")
                .unwrap_or_else(|| panic!("account {address} exists"))
                .balance;
            (*address, balance)
        })
        .collect();

    (outcome, balances)
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
async fn post_exec_nonzero_payload_applies_refunds() {
    let mut loaded = load_test_fixture(post_exec_fixture_path()).await;
    let block_number = fixture_block_number(&loaded.fixture.parent_header);
    let timestamp = loaded.fixture.executing_payload.payload_attributes.timestamp;
    assert!(
        !loaded.fixture.rollup_config.is_isthmus_active(timestamp),
        "fixture must remain pre-Isthmus so operator-fee settlement is zero"
    );

    let recovered = loaded
        .fixture
        .executing_payload
        .recovered_transactions()
        .collect::<Result<Vec<_>, _>>()
        .expect("fixture transactions recover");
    assert_eq!(recovered.len(), 10, "fixture must contain ten pre-PostExec transactions");

    // Leave the deposit at index 0 and the first regular transaction unrefunded, then refund one
    // gas from each of the remaining eight transactions.
    let refunded = &recovered[2..];
    let refund_senders = refunded.iter().map(|tx| tx.signer()).collect::<BTreeSet<_>>();
    let beneficiary = loaded.fixture.executing_payload.payload_attributes.suggested_fee_recipient;
    assert_ne!(beneficiary, BASE_FEE_RECIPIENT);
    assert_ne!(beneficiary, OPERATOR_FEE_RECIPIENT);
    assert_ne!(BASE_FEE_RECIPIENT, OPERATOR_FEE_RECIPIENT);
    assert!(
        refund_senders.iter().all(|sender| ![
            beneficiary,
            BASE_FEE_RECIPIENT,
            OPERATOR_FEE_RECIPIENT
        ]
        .contains(sender)),
        "refund senders must not alias fee recipients"
    );

    let mut affected_accounts = refund_senders.clone();
    affected_accounts.extend([beneficiary, BASE_FEE_RECIPIENT, OPERATOR_FEE_RECIPIENT]);
    let (baseline, baseline_balances) = execute_loaded_fixture_with_balances(
        load_test_fixture(post_exec_fixture_path()).await,
        None,
        &affected_accounts,
    );

    let transactions = loaded.fixture.executing_payload.transactions.as_mut().unwrap();
    let gas_refund_entries = (2..transactions.len())
        .map(|index| SDMGasEntry { index: index as u64, gas_refund: 1 })
        .collect::<Vec<_>>();
    let refund_total = gas_refund_entries.len() as u64;
    assert_eq!(refund_total, 8);
    append_post_exec_tx(transactions, block_number, gas_refund_entries);

    let (outcome, refunded_balances) =
        execute_loaded_fixture_with_balances(loaded, Some(true), &affected_accounts);
    assert_eq!(
        outcome.execution_result.receipts.len(),
        baseline.execution_result.receipts.len() + 1
    );
    assert!(matches!(
        outcome.execution_result.receipts.last(),
        Some(OpReceiptEnvelope::PostExec(_))
    ));
    assert_eq!(
        outcome.execution_result.gas_used,
        baseline.execution_result.gas_used - refund_total
    );
    assert_eq!(outcome.header.gas_used, baseline.header.gas_used - refund_total);

    let base_fee = baseline.header.base_fee_per_gas.expect("fixture has a base fee");
    let mut expected_sender_credits = BTreeMap::<Address, U256>::new();
    let mut expected_beneficiary_debit = U256::ZERO;
    for tx in refunded {
        let effective_gas_price = tx.inner().effective_gas_price(Some(base_fee));
        *expected_sender_credits.entry(tx.signer()).or_default() += U256::from(effective_gas_price);
        expected_beneficiary_debit +=
            U256::from(effective_gas_price.saturating_sub(u128::from(base_fee)));
    }
    let expected_base_fee_debit = U256::from(base_fee) * U256::from(refund_total);

    for (sender, expected_credit) in expected_sender_credits {
        assert_eq!(
            refunded_balances[&sender],
            baseline_balances[&sender] + expected_credit,
            "refunded sender {sender} must receive its exact effective-gas-price credit"
        );
    }
    assert_eq!(
        refunded_balances[&beneficiary],
        baseline_balances[&beneficiary]
            .checked_sub(expected_beneficiary_debit)
            .expect("beneficiary funds the priority-fee refund"),
        "beneficiary must fund the exact priority-fee share"
    );
    assert_eq!(
        refunded_balances[&BASE_FEE_RECIPIENT],
        baseline_balances[&BASE_FEE_RECIPIENT]
            .checked_sub(expected_base_fee_debit)
            .expect("base-fee recipient funds the base-fee refund"),
        "base-fee recipient must fund the exact base-fee share"
    );
    assert_eq!(
        refunded_balances[&OPERATOR_FEE_RECIPIENT], baseline_balances[&OPERATOR_FEE_RECIPIENT],
        "pre-Isthmus settlement must not debit the operator-fee recipient"
    );

    let total = |balances: &BTreeMap<Address, U256>| {
        balances.values().copied().fold(U256::ZERO, |total, balance| total + balance)
    };
    assert_eq!(
        total(&refunded_balances),
        total(&baseline_balances),
        "settlement must only transfer value among affected accounts"
    );
    assert_ne!(outcome.header.state_root, baseline.header.state_root);
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
