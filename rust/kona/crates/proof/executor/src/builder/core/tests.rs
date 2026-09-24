use crate::{
    BlockBuildingOutcome, ExecutorError, StatelessL2Builder,
    test_utils::{
        ExecutorTestFixture, LoadedExecutorTestFixture, execute_loaded_fixture, load_test_fixture,
        run_test_fixture,
    },
};
use alloy_consensus::{Header, Transaction};
use alloy_eips::Encodable2718;
use alloy_evm::block::{BlockExecutionError, BlockValidationError};
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

/// Projection proofs must apply exactly the deposit policy used by op-reth. A portal
/// deposit may mint, transfer, create or call arbitrary code on the private chain;
/// none of those effects may enter its public projection's output root.
#[test]
fn projection_deposit_policy_is_applied_by_stateless_execution() {
    use crate::NoopTrieDBProvider;
    use alloy_eips::eip2718::Encodable2718;
    use alloy_primitives::TxKind;
    use alloy_rpc_types_engine::PayloadAttributes;
    use kona_genesis::RollupConfig;
    use op_alloy_consensus::{OpTxEnvelope, TxDeposit};
    use op_alloy_rpc_types_engine::OpPayloadAttributes;

    for create in [false, true] {
        let mut roots = Vec::new();
        for projection in [false, true] {
            let cfg = RollupConfig {
                private_projection: projection.then(projection_profile),
                ..Default::default()
            };
            let parent = Header {
                state_root: alloy_trie::EMPTY_ROOT_HASH,
                gas_limit: 30_000_000,
                base_fee_per_gas: Some(0),
                ..Default::default()
            };
            let tx = OpTxEnvelope::from(TxDeposit {
                from: Address::with_last_byte(0xaa),
                to: if create { TxKind::Create } else { Address::with_last_byte(0xbb).into() },
                mint: 1000,
                value: U256::from(500),
                gas_limit: 100_000,
                ..Default::default()
            });
            let mut builder = StatelessL2Builder::new(
                &cfg,
                OpEvmFactory::<alloy_op_evm::OpTx>::default(),
                alloy_op_evm::block::OpAlloyReceiptBuilder::default(),
                NoopTrieDBProvider,
                NoopTrieHinter,
                parent.seal_slow(),
            );
            let outcome = builder
                .build_block(OpPayloadAttributes {
                    payload_attributes: PayloadAttributes { timestamp: 2, ..Default::default() },
                    transactions: Some(vec![tx.encoded_2718().into()]),
                    gas_limit: Some(30_000_000),
                    no_tx_pool: Some(true),
                    ..Default::default()
                })
                .unwrap();
            if projection {
                assert_eq!(outcome.header.state_root, alloy_trie::EMPTY_ROOT_HASH);
                assert_eq!(outcome.header.gas_used, 0);
            } else {
                assert_ne!(outcome.header.state_root, alloy_trie::EMPTY_ROOT_HASH);
                assert!(outcome.header.gas_used > 0);
            }
            roots.push(outcome.header.state_root);
        }
        assert_ne!(roots[0], roots[1]);
    }
}

/// A projection admission profile. The executor only tests for its presence, so it is built from
/// the minimal JSON form, which stays valid as the profile gains defaulted fields.
fn projection_profile() -> kona_genesis::PrivateProjectionConfig {
    serde_json::from_value(serde_json::json!({
        "genesis_output_root": alloy_primitives::B256::repeat_byte(1),
        "verifier": "insecure-stub-v1",
    }))
    .expect("minimal projection profile")
}

/// Serves the trie nodes and bytecode of a vector prestate. Every prestate account has empty
/// storage.
#[derive(Debug)]
struct PrestateProvider {
    nodes: std::collections::HashMap<alloy_primitives::B256, alloy_primitives::Bytes>,
    code: std::collections::HashMap<alloy_primitives::B256, alloy_primitives::Bytes>,
}

impl PrestateProvider {
    /// Builds the state trie of `accounts` (address, nonce, balance, code) and returns its root.
    fn new(
        accounts: Vec<(Address, u64, U256, alloy_primitives::Bytes)>,
    ) -> (Self, alloy_primitives::B256) {
        use alloy_primitives::keccak256;
        use alloy_trie::{HashBuilder, Nibbles, TrieAccount, proof::ProofRetainer};

        let mut leaves: Vec<_> = accounts
            .into_iter()
            .map(|(address, nonce, balance, code)| {
                let account = TrieAccount {
                    nonce,
                    balance,
                    storage_root: alloy_trie::EMPTY_ROOT_HASH,
                    code_hash: keccak256(&code),
                };
                (Nibbles::unpack(keccak256(address)), alloy_rlp::encode(account), code)
            })
            .collect();
        leaves.sort_by_key(|leaf| leaf.0);
        let mut builder = HashBuilder::default()
            .with_proof_retainer(ProofRetainer::new(leaves.iter().map(|l| l.0).collect()));
        let mut code = std::collections::HashMap::new();
        for (key, value, bytecode) in leaves {
            builder.add_leaf(key, &value);
            code.insert(keccak256(&bytecode), bytecode);
        }
        let root = builder.root();
        let nodes = builder
            .take_proof_nodes()
            .into_inner()
            .into_values()
            .map(|node| (keccak256(&node), node))
            .collect();
        (Self { nodes, code }, root)
    }
}

impl kona_mpt::TrieProvider for PrestateProvider {
    type Error = String;

    fn trie_node_by_hash(&self, key: alloy_primitives::B256) -> Result<kona_mpt::TrieNode, String> {
        use alloy_rlp::Decodable;
        let raw = self.nodes.get(&key).ok_or_else(|| format!("missing trie node {key}"))?;
        kona_mpt::TrieNode::decode(&mut raw.as_ref()).map_err(|e| e.to_string())
    }
}

impl crate::TrieDBProvider for PrestateProvider {
    fn bytecode_by_hash(
        &self,
        code_hash: alloy_primitives::B256,
    ) -> Result<alloy_primitives::Bytes, String> {
        self.code.get(&code_hash).cloned().ok_or_else(|| format!("missing code {code_hash}"))
    }

    fn header_by_hash(&self, hash: alloy_primitives::B256) -> Result<Header, String> {
        Err(format!("no headers in a vector prestate: {hash}"))
    }
}

/// The shared op-reth/Kona projection executor vectors, generated by
/// `op-private-interop/projection/execution_vectors_test.go`.
const EXECUTION_VECTORS: &str =
    include_str!("../../../../../../../../op-private-interop/projection/testdata/execution.json");

/// Every case of the shared executor vectors (spec-sound-profile §E.4), in both modes, through
/// the projection trigger in [`StatelessL2Builder::new`]: a failed non-deposit transaction
/// invalidates the block with `ProjectionSequencerTxFailed`, a user deposit never does, and valid
/// blocks match the reference statuses, gas used and state root.
#[test]
fn projection_execution_vectors() {
    use alloy_eips::eip1559::BaseFeeParams;
    use alloy_op_evm::block::OpBlockExecutionError;
    use alloy_primitives::{B64, B256, Bytes};
    use alloy_rpc_types_engine::PayloadAttributes;
    use kona_genesis::{HardForkConfig, RollupConfig};
    use op_alloy_rpc_types_engine::OpPayloadAttributes;
    use serde_json::Value;

    fn b256(v: &Value) -> B256 {
        v.as_str().unwrap().parse().unwrap()
    }

    let vectors: Value = serde_json::from_str(EXECUTION_VECTORS).unwrap();
    let env = &vectors["env"];
    let u64_of = |key: &str| env[key].as_u64().unwrap();
    let cases = vectors["cases"].as_array().unwrap();
    assert!(cases.len() >= 5);
    let prestate = || {
        PrestateProvider::new(
            env["prestate"]
                .as_object()
                .unwrap()
                .iter()
                .map(|(address, account)| {
                    (
                        address.parse().unwrap(),
                        account["nonce"].as_u64().unwrap(),
                        account["balance"].as_str().unwrap().parse().unwrap(),
                        account["code"].as_str().unwrap().parse().unwrap(),
                    )
                })
                .collect(),
        )
    };

    for case in cases {
        let name = case["name"].as_str().unwrap();
        let txs: Vec<Bytes> = case["transactions"]
            .as_array()
            .unwrap()
            .iter()
            .map(|tx| tx.as_str().unwrap().parse().unwrap())
            .collect();
        for (mode, projection) in [("execution", false), ("projection", true)] {
            let expected = &case[mode];
            let cfg = RollupConfig {
                l2_chain_id: u64_of("chain_id").into(),
                block_time: 2,
                hardforks: HardForkConfig {
                    regolith_time: Some(0),
                    canyon_time: Some(0),
                    delta_time: Some(0),
                    ecotone_time: Some(0),
                    fjord_time: Some(0),
                    granite_time: Some(0),
                    holocene_time: Some(0),
                    isthmus_time: Some(0),
                    jovian_time: Some(0),
                    karst_time: Some(0),
                    lagoon_time: Some(0),
                    ..Default::default()
                },
                private_projection: projection.then(projection_profile),
                ..Default::default()
            };
            let (provider, prestate_root) = prestate();
            let parent = Header {
                number: u64_of("number") - 1,
                timestamp: u64_of("timestamp") - 2,
                state_root: prestate_root,
                gas_limit: u64_of("gas_limit"),
                base_fee_per_gas: Some(u64_of("base_fee")),
                extra_data: op_alloy_consensus::encode_jovian_extra_data(
                    B64::ZERO,
                    BaseFeeParams::new(250, 6),
                    0,
                )
                .unwrap(),
                ..Default::default()
            };
            let mut builder = StatelessL2Builder::new(
                &cfg,
                OpEvmFactory::<alloy_op_evm::OpTx>::default(),
                alloy_op_evm::block::OpAlloyReceiptBuilder::default(),
                provider,
                NoopTrieHinter,
                parent.seal_slow(),
            );
            let result = builder.build_block(OpPayloadAttributes {
                payload_attributes: PayloadAttributes {
                    timestamp: u64_of("timestamp"),
                    prev_randao: b256(&env["prev_randao"]),
                    suggested_fee_recipient: env["coinbase"].as_str().unwrap().parse().unwrap(),
                    withdrawals: Some(Vec::new()),
                    parent_beacon_block_root: Some(b256(&env["parent_beacon_block_root"])),
                    ..Default::default()
                },
                transactions: Some(txs.clone()),
                no_tx_pool: Some(true),
                gas_limit: Some(u64_of("gas_limit")),
                eip_1559_params: Some(B64::ZERO),
                min_base_fee: Some(0),
            });
            if expected["valid"].as_bool().unwrap() {
                let outcome = result.unwrap_or_else(|e| panic!("{name}/{mode}: {e}"));
                let statuses: Vec<u64> = outcome
                    .execution_result
                    .receipts
                    .iter()
                    .map(|r| u64::from(r.status()))
                    .collect();
                let want: Vec<u64> = expected["statuses"]
                    .as_array()
                    .unwrap()
                    .iter()
                    .map(|s| s.as_u64().unwrap())
                    .collect();
                assert_eq!(statuses, want, "{name}/{mode}: statuses");
                assert_eq!(
                    outcome.header.gas_used,
                    expected["gas_used"].as_u64().unwrap(),
                    "{name}/{mode}: gas used"
                );
                assert_eq!(
                    outcome.header.state_root,
                    b256(&expected["state_root"]),
                    "{name}/{mode}: state root"
                );
            } else {
                assert_eq!(expected["error"], "ProjectionSequencerTxFailed", "{name}/{mode}");
                let err = result.err().unwrap_or_else(|| panic!("{name}/{mode}: must be invalid"));
                let ExecutorError::ExecutionError(BlockExecutionError::Validation(
                    BlockValidationError::Other(inner),
                )) = err
                else {
                    panic!("{name}/{mode}: expected a validation error, got {err}");
                };
                match inner.downcast_ref::<OpBlockExecutionError>() {
                    Some(OpBlockExecutionError::ProjectionSequencerTxFailed { tx_index }) => {
                        assert_eq!(
                            *tx_index,
                            expected["tx_index"].as_u64().unwrap(),
                            "{name}/{mode}: tx index"
                        )
                    }
                    _ => panic!("{name}/{mode}: expected ProjectionSequencerTxFailed, got {inner}"),
                }
            }
        }
    }
}
