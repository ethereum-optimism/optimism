use super::*;

fn recovered_post_exec(block_number: u64, entries: Vec<SDMGasEntry>) -> Recovered<OpTxEnvelope> {
    Recovered::new_unchecked(
        OpTxEnvelope::PostExec(build_post_exec_tx(block_number, entries).seal_slow()),
        Address::ZERO,
    )
}

fn legacy_tx(nonce: u64, to: Address) -> Recovered<OpTxEnvelope> {
    legacy_tx_with_gas(nonce, to, 50_000)
}

fn legacy_tx_with_gas(nonce: u64, to: Address, gas_limit: u64) -> Recovered<OpTxEnvelope> {
    recovered_legacy(TxLegacy { nonce, gas_limit, to: TxKind::Call(to), ..Default::default() })
}

fn legacy_tx_with_price(
    nonce: u64,
    to: Address,
    gas_limit: u64,
    gas_price: u128,
) -> Recovered<OpTxEnvelope> {
    recovered_legacy(TxLegacy {
        nonce,
        gas_limit,
        gas_price,
        to: TxKind::Call(to),
        ..Default::default()
    })
}

fn full_refund_for_second_tx(
    block_gas_limit: u64,
    tx0: &Recovered<OpTxEnvelope>,
    tx1: &Recovered<OpTxEnvelope>,
) -> Vec<SDMGasEntry> {
    let mut fixture = JovianExecutorFixture::new(
        DEFAULT_DA_FOOTPRINT_GAS_SCALAR,
        block_gas_limit,
        JOVIAN_TIMESTAMP,
    );
    let mut probe = fixture.executor();
    probe.execute_transaction(tx0).expect("probe first tx");
    let tx1_evm_gas_used = probe.execute_transaction(tx1).expect("probe second tx").tx_gas_used();

    vec![SDMGasEntry { index: 1, gas_refund: tx1_evm_gas_used }]
}

fn assert_invalid_post_exec(err: BlockExecutionError, expected_reason: &str) {
    match err {
        BlockExecutionError::Validation(BlockValidationError::Other(err)) => {
            match err.downcast_ref::<OpBlockExecutionError>() {
                Some(OpBlockExecutionError::InvalidPostExecPayload(reason)) => {
                    assert_eq!(reason, expected_reason);
                }
                other => panic!("expected invalid post-exec payload error, got: {other:?}"),
            }
        }
        other => panic!("expected invalid post-exec payload error, got: {other:?}"),
    }
}

#[derive(Debug, Clone, Default)]
struct FixedRefundPolicy {
    kind: Option<PostExecTxKind>,
}

impl PostExecRefundInspector for FixedRefundPolicy {
    type Snapshot = ();

    fn begin_tx(&mut self, ctx: PostExecTxContext) {
        self.kind = Some(ctx.kind);
    }

    fn note_account_touch(&mut self, _address: Address) {}

    fn finish_tx(&mut self) -> PostExecExecutedTx {
        PostExecExecutedTx {
            refund_total: u64::from(self.kind.take() == Some(PostExecTxKind::Normal)),
            refund_events: Vec::new(),
        }
    }

    fn inspect_step<CTX>(&mut self, _interp: &mut Interpreter, _context: &mut CTX)
    where
        CTX: ContextTr<Journal: JournalExt>,
    {
    }

    fn inspect_call<CTX>(&mut self, _context: &mut CTX, _inputs: &mut CallInputs)
    where
        CTX: ContextTr<Journal: JournalExt>,
    {
    }

    fn inspect_create<CTX>(&mut self, _context: &mut CTX, _inputs: &mut CreateInputs)
    where
        CTX: ContextTr<Journal: JournalExt>,
    {
    }

    fn inspect_selfdestruct(&mut self, _contract: Address, _target: Address, _value: U256) {}

    fn snapshot(&self) -> Self::Snapshot {}

    fn restore(&mut self, _snapshot: Self::Snapshot) {}
}

#[test]
fn fixed_policy_producer_verifier_roundtrip() {
    let tx = observer_test_tx();
    let mut producer_db = prepare_observer_db();
    let receipt_builder = OpAlloyReceiptBuilder::default();
    let hardforks = OpChainHardforks::op_mainnet();
    let mut producer =
        build_policy_executor::<FixedRefundPolicy>(&mut producer_db, &receipt_builder, &hardforks);

    producer.execute_transaction(&tx).expect("producer executes normal tx");
    let entries = producer.post_exec_entries().to_vec();
    assert_eq!(entries, vec![SDMGasEntry { index: 0, gas_refund: 1 }]);
    let post_exec = recovered_post_exec(0, entries.clone());
    producer.execute_transaction(&post_exec).expect("producer executes post-exec tx");
    let (_, produced) = producer.finish().expect("producer finishes block");

    let mut verifier_fixture =
        JovianExecutorFixture::new(DEFAULT_DA_FOOTPRINT_GAS_SCALAR, 500_000, JOVIAN_TIMESTAMP);
    verifier_fixture.db = prepare_observer_db();
    let mut verifier = verifier_fixture.verifier(0, entries);
    verifier.execute_transaction(&tx).expect("verifier executes normal tx");
    verifier.execute_transaction(&post_exec).expect("verifier executes post-exec tx");
    let (_, verified) = verifier.finish().expect("verifier finishes block");

    assert_eq!(verified.gas_used, produced.gas_used);
    assert_eq!(verified.receipts, produced.receipts);
}

#[test]
fn fixed_policy_tracks_pre_refund_gas() {
    let mut db = prepare_observer_db();
    let receipt_builder = OpAlloyReceiptBuilder::default();
    let hardforks = OpChainHardforks::op_mainnet();
    let mut executor =
        build_policy_executor::<FixedRefundPolicy>(&mut db, &receipt_builder, &hardforks);

    executor.execute_transaction(&observer_test_tx()).expect("normal tx executes");
    assert_eq!(executor.evm_gas_used - executor.gas_used, 1);
    assert_eq!(executor.post_exec_entries(), &[SDMGasEntry { index: 0, gas_refund: 1 }]);
}

#[derive(Debug, Clone, Default)]
struct ErroringRefundPolicy {
    block_state: u64,
}

impl PostExecRefundInspector for ErroringRefundPolicy {
    type Snapshot = u64;

    fn begin_tx(&mut self, _ctx: PostExecTxContext) {
        self.block_state += 1;
    }

    fn note_account_touch(&mut self, _address: Address) {}

    fn finish_tx(&mut self) -> PostExecExecutedTx {
        PostExecExecutedTx { refund_total: u64::MAX, refund_events: Vec::new() }
    }

    fn inspect_step<CTX>(&mut self, _interp: &mut Interpreter, _context: &mut CTX)
    where
        CTX: ContextTr<Journal: JournalExt>,
    {
    }

    fn inspect_call<CTX>(&mut self, _context: &mut CTX, _inputs: &mut CallInputs)
    where
        CTX: ContextTr<Journal: JournalExt>,
    {
    }

    fn inspect_create<CTX>(&mut self, _context: &mut CTX, _inputs: &mut CreateInputs)
    where
        CTX: ContextTr<Journal: JournalExt>,
    {
    }

    fn inspect_selfdestruct(&mut self, _contract: Address, _target: Address, _value: U256) {}

    fn snapshot(&self) -> Self::Snapshot {
        self.block_state
    }

    fn restore(&mut self, snapshot: Self::Snapshot) {
        self.block_state = snapshot;
    }
}

#[test]
fn execution_error_restores_refund_policy_snapshot() {
    let mut db = prepare_observer_db();
    let receipt_builder = OpAlloyReceiptBuilder::default();
    let hardforks = OpChainHardforks::op_mainnet();
    let mut executor =
        build_policy_executor::<ErroringRefundPolicy>(&mut db, &receipt_builder, &hardforks);

    assert_eq!(executor.refund_snapshot(), 0);
    executor
        .execute_transaction(&observer_test_tx())
        .expect_err("an impossible refund must fail execution");
    assert_eq!(executor.refund_snapshot(), 0, "failed execution must restore policy state");
}
#[test]
fn test_settlement_state_account_preserves_original_info() {
    type TestExecutor<'a> = OpBlockExecutor<
        OpEvm<&'a mut State<InMemoryDB>, NoOpInspector>,
        &'a OpAlloyReceiptBuilder,
        &'a OpChainHardforks,
    >;

    let mut backing_db = InMemoryDB::default();
    backing_db.insert_account_info(
        BASE_FEE_RECIPIENT,
        AccountInfo { balance: U256::from(10), ..Default::default() },
    );
    let mut db = State::builder().with_database(backing_db).with_bundle_update().build();
    revm::Database::basic(&mut db, BASE_FEE_RECIPIENT)
        .expect("failed to load base fee recipient into cache");

    let mut credited_account =
        Account::from(AccountInfo { balance: U256::from(15), ..Default::default() });
    credited_account.mark_touch();
    revm::DatabaseCommit::commit(
        &mut db,
        HashMap::from_iter([(BASE_FEE_RECIPIENT, credited_account)]),
    );

    let mut state = EvmState::default();
    let mut db_ref = &mut db;
    let account = TestExecutor::state_account_mut(&mut db_ref, &mut state, BASE_FEE_RECIPIENT)
        .expect("failed to materialize settlement account");
    assert_eq!(account.info.balance, U256::from(15));
    // original_info mirrors current info here — State::commit computes the
    // true previous value from its own cache, so the bundle stays correct.
    assert_eq!(account.original_info().balance, U256::from(15));

    account.info.balance = account.info.balance.saturating_sub(U256::from(3));
    revm::DatabaseCommit::commit(&mut db, state);
    db.merge_transitions(revm::database::states::bundle_state::BundleRetention::Reverts);

    let bundle = db.take_bundle();
    let bundle_account =
        bundle.account(&BASE_FEE_RECIPIENT).expect("bundle must contain the base fee recipient");
    assert_eq!(bundle_account.original_info.as_ref().unwrap().balance, U256::from(10));
    assert_eq!(bundle_account.info.as_ref().unwrap().balance, U256::from(12));
}

#[test]
fn test_post_exec_settlement_deltas_conserve_value() {
    const BASE_FEE: u64 = 7;
    const GAS_PRICE: u128 = 100;
    const EVM_GAS_USED: u64 = 50_000;
    const REFUND: u64 = 1_000;

    let mut fixture = JovianExecutorFixture { base_fee: BASE_FEE, ..Default::default() };
    let mut executor = fixture.executor();

    let tx = legacy_tx_with_price(0, Address::from([0x11; 20]), DEFAULT_GAS_LIMIT, GAS_PRICE);
    let deltas = executor
        .post_exec_settlement_deltas(
            &tx,
            EVM_GAS_USED,
            REFUND,
            /* is_deposit */ false,
            /* is_post_exec */ false,
        )
        .expect("settlement deltas computed");

    let refund = U256::from(REFUND);
    // Base-fee share: refund * basefee.
    assert_eq!(deltas.base_fee_balance_delta, refund * U256::from(BASE_FEE));
    // Beneficiary (priority) share: refund * (effective_gas_price - basefee).
    assert_eq!(
        deltas.beneficiary_balance_delta,
        refund * U256::from(GAS_PRICE - u128::from(BASE_FEE)),
    );
    // Operator-fee share is non-zero post-Isthmus (the Jovian fixture sets operator-fee
    // scalars).
    assert!(
        deltas.operator_fee_balance_delta > U256::ZERO,
        "operator-fee delta should be charged post-Isthmus",
    );

    // Conservation ("no infinite mint"): the sender credit equals the sum of the three
    // recipient debits, so settlement neither creates nor destroys ETH.
    assert_eq!(
        deltas.sender_balance_delta,
        deltas.beneficiary_balance_delta +
            deltas.base_fee_balance_delta +
            deltas.operator_fee_balance_delta,
    );
    // Cross-check the sender credit against the spec formula directly.
    assert_eq!(
        deltas.sender_balance_delta,
        refund * U256::from(GAS_PRICE) + deltas.operator_fee_balance_delta,
    );
}

#[test]
fn test_post_exec_settlement_deltas_skip_non_refunding_txs() {
    let mut fixture = JovianExecutorFixture { base_fee: 7, ..Default::default() };
    let mut executor = fixture.executor();
    let tx = legacy_tx_with_price(0, Address::from([0x11; 20]), DEFAULT_GAS_LIMIT, 100);

    let is_no_op = |d: PostExecAdjustment| {
        d.sender_balance_delta.is_zero() &&
            d.beneficiary_balance_delta.is_zero() &&
            d.base_fee_balance_delta.is_zero() &&
            d.operator_fee_balance_delta.is_zero()
    };

    // Deposit: warms state for later txs but is never refunded.
    assert!(
        is_no_op(
            executor
                .post_exec_settlement_deltas(
                    &tx, /* evm_gas_used */ 50_000, /* post_exec_refund */ 1_000,
                    /* is_deposit */ true, /* is_post_exec */ false,
                )
                .unwrap()
        ),
        "deposits never settle a refund",
    );
    // The post-exec (0x7D) tx itself never claims.
    assert!(
        is_no_op(
            executor
                .post_exec_settlement_deltas(
                    &tx, /* evm_gas_used */ 50_000, /* post_exec_refund */ 1_000,
                    /* is_deposit */ false, /* is_post_exec */ true,
                )
                .unwrap()
        ),
        "the post-exec tx never settles a refund",
    );
    // Zero refund: nothing to settle.
    assert!(
        is_no_op(
            executor
                .post_exec_settlement_deltas(
                    &tx, /* evm_gas_used */ 50_000, /* post_exec_refund */ 0,
                    /* is_deposit */ false, /* is_post_exec */ false,
                )
                .unwrap()
        ),
        "a zero refund produces no settlement",
    );
}

#[test]
fn test_post_exec_settlement_conserves_value_at_arithmetic_extremes() {
    for base_fee in [0u64, 1, 1_000_000_000, u64::MAX] {
        let mut fixture = JovianExecutorFixture { base_fee, ..Default::default() };
        let mut executor = fixture.executor();

        for refund in [1u64, 21_000, u64::MAX / 2, u64::MAX] {
            for gas_price in [u128::from(base_fee), u128::from(base_fee) + 1, u128::MAX] {
                let tx = legacy_tx_with_price(0, Address::from([0x11; 20]), 50_000, gas_price);
                let deltas = executor
                    .post_exec_settlement_deltas(
                        &tx,
                        /* evm_gas_used */ u64::MAX,
                        /* post_exec_refund */ refund,
                        /* is_deposit */ false,
                        /* is_post_exec */ false,
                    )
                    .expect("settlement deltas");
                let inputs = format!("base_fee={base_fee}, refund={refund}, gas_price={gas_price}");

                // Check each leg; conservation alone can hide offsetting clamps.
                let refund_u256 = U256::from(refund);
                assert_eq!(
                    deltas.base_fee_balance_delta,
                    refund_u256 * U256::from(base_fee),
                    "base-fee leg must not saturate ({inputs})",
                );
                assert_eq!(
                    deltas.beneficiary_balance_delta,
                    refund_u256 * U256::from(gas_price - u128::from(base_fee)),
                    "beneficiary leg must not saturate ({inputs})",
                );
                assert_eq!(
                    deltas.operator_fee_balance_delta,
                    refund_u256 * U256::from(5 * 100),
                    "operator-fee leg must not saturate ({inputs})",
                );
                assert_eq!(
                    deltas.sender_balance_delta,
                    refund_u256 * U256::from(gas_price) + deltas.operator_fee_balance_delta,
                    "sender credit must not saturate ({inputs})",
                );
                assert_eq!(
                    deltas.sender_balance_delta,
                    deltas.beneficiary_balance_delta +
                        deltas.base_fee_balance_delta +
                        deltas.operator_fee_balance_delta,
                    "settlement must conserve value ({inputs})",
                );
            }
        }
    }
}

#[test]
fn test_post_exec_settlement_underflow_is_rejected() {
    let mut fixture = JovianExecutorFixture::default();
    let mut executor = fixture.executor();

    // BASE_FEE_RECIPIENT is unfunded in the test DB, so any base-fee debit underflows.
    let deltas = PostExecAdjustment {
        refund: 1,
        sender_balance_delta: U256::from(5),
        base_fee_balance_delta: U256::from(5),
        ..Default::default()
    };

    let sender = Address::from([0x22; 20]);
    let mut state = EvmState::default();
    let err = executor
        .apply_post_exec_refund_to_state(&mut state, sender, &deltas)
        .expect_err("settlement must reject an unfunded recipient debit");

    match err {
        BlockExecutionError::Validation(BlockValidationError::Other(inner)) => {
            match inner.downcast_ref::<OpBlockExecutionError>() {
                Some(OpBlockExecutionError::PostExecSettlementUnderflow { address, delta }) => {
                    assert_eq!(*address, BASE_FEE_RECIPIENT);
                    assert_eq!(*delta, U256::from(5));
                }
                other => panic!("expected PostExecSettlementUnderflow, got: {other:?}"),
            }
        }
        other => panic!("expected a validation error, got: {other:?}"),
    }
}

#[test]
fn test_verifier_rejects_malicious_payload_whose_refunds_hide_pre_refund_overuse() {
    const BLOCK_GAS_LIMIT: u64 = 100_000;
    let target = Address::from([0x11; 20]);
    let tx0 = legacy_tx(0, target);
    let tx1 = legacy_tx(1, target);

    // Refund the second tx completely. The verifier accepts refund == evm_gas_used but must not
    // let that canonical-gas discount buy extra real compute later in the block.
    let entries = full_refund_for_second_tx(BLOCK_GAS_LIMIT, &tx0, &tx1);

    let mut fixture = JovianExecutorFixture::new(
        DEFAULT_DA_FOOTPRINT_GAS_SCALAR,
        BLOCK_GAS_LIMIT,
        JOVIAN_TIMESTAMP,
    );
    let mut verifier = fixture.verifier(0, entries);
    verifier.execute_transaction(&tx0).expect("first tx fits");
    verifier.execute_transaction(&tx1).expect("second tx is fully refunded canonically");

    let evm_gas_available = BLOCK_GAS_LIMIT - verifier.evm_gas_used;
    let canonical_gas_available = BLOCK_GAS_LIMIT - verifier.gas_used;
    assert!(evm_gas_available < canonical_gas_available);

    let tx2_gas_limit = evm_gas_available + 1;
    assert!(
        tx2_gas_limit <= canonical_gas_available,
        "malicious tx should fit canonical gas but exceed pre-refund gas"
    );
    let tx2 = legacy_tx_with_gas(2, target, tx2_gas_limit);

    let err = verifier
        .execute_transaction(&tx2)
        .expect_err("verifier must reject pre-refund gas overuse even if refunds hide it");
    assert_gas_limit_exceeded(err, tx2_gas_limit, evm_gas_available);
}

#[test]
fn test_verifier_accepts_payload_when_pre_refund_stays_below_limit() {
    const BLOCK_GAS_LIMIT: u64 = 100_000;
    let target = Address::from([0x11; 20]);
    let tx0 = legacy_tx(0, target);
    let tx1 = legacy_tx(1, target);
    let entries = full_refund_for_second_tx(BLOCK_GAS_LIMIT, &tx0, &tx1);

    let mut fixture = JovianExecutorFixture::new(
        DEFAULT_DA_FOOTPRINT_GAS_SCALAR,
        BLOCK_GAS_LIMIT,
        JOVIAN_TIMESTAMP,
    );
    let mut verifier = fixture.verifier(0, entries.clone());
    verifier.execute_transaction(&tx0).expect("first tx fits");
    verifier.execute_transaction(&tx1).expect("second tx is fully refunded canonically");

    let tx2 = legacy_tx_with_gas(2, target, BLOCK_GAS_LIMIT - verifier.evm_gas_used);
    verifier
        .execute_transaction(&tx2)
        .expect("tx declared within the remaining pre-refund budget is accepted");
    let post_exec_recovered = recovered_post_exec(0, entries);
    verifier.execute_transaction(&post_exec_recovered).expect("post-exec tx verifies");
    verifier.finish().expect("verifier finishes accepted boundary block");
}

#[test]
fn test_mismatched_payload_block_number_fails_pre_execution() {
    // build_executor configures BlockEnv with block number 0; a payload anchored to a
    // different block must be rejected before any tx runs.
    let mut fixture = JovianExecutorFixture::default();
    let mut executor = fixture.verifier(42, vec![]);

    let err =
        executor.apply_pre_execution_changes().expect_err("mismatched block number must fail");
    assert_invalid_post_exec(err, "payload block number 42 does not match block number 0");
}

#[test]
fn test_duplicate_payload_index_fails_pre_execution() {
    // Two entries colliding on tx index 3 — the second insert must be flagged at construction
    // and surface as a pre-execution failure.
    let mut fixture = JovianExecutorFixture::default();
    let mut executor = fixture.verifier(
        0,
        vec![SDMGasEntry { index: 3, gas_refund: 10 }, SDMGasEntry { index: 3, gas_refund: 20 }],
    );

    let err = executor
        .apply_pre_execution_changes()
        .expect_err("duplicate payload index must fail pre-execution");
    assert_invalid_post_exec(err, "duplicate post-exec payload entry for tx index 3");
}

#[test]
fn test_verifier_rejects_payload_targeting_non_normal_tx() {
    for (tx_index, is_deposit, is_post_exec, evm_gas_used, expected_reason) in [
        (0, true, false, 21_000, "payload entry targets deposit tx index 0"),
        (4, false, true, 0, "payload entry targets post-exec tx index 4"),
    ] {
        let mut fixture = JovianExecutorFixture::default();
        let executor = fixture.verifier(0, vec![SDMGasEntry { index: tx_index, gas_refund: 1 }]);

        let err = executor
            .verifier_post_exec_refund_for_tx(tx_index, is_deposit, is_post_exec, evm_gas_used)
            .expect_err("payload entries must not target non-normal txs");
        assert_invalid_post_exec(err, expected_reason);
    }
}

#[test]
fn test_verifier_rejects_refund_exceeding_evm_gas() {
    let mut fixture = JovianExecutorFixture::default();
    let executor = fixture.verifier(0, vec![SDMGasEntry { index: 2, gas_refund: 50_000 }]);

    // evm_gas_used < payload refund — a refund that exceeds the tx's EVM-reported cost is
    // impossible under SDM semantics and must be rejected, otherwise canonical_gas_used
    // would underflow to a bogus value via saturating_sub.
    let err = executor
        .verifier_post_exec_refund_for_tx(2, false, false, 40_000)
        .expect_err("refund greater than evm_gas_used must be rejected");
    assert_invalid_post_exec(err, "payload refund 50000 exceeds evm_gas_used 40000 for tx index 2");

    // Boundary: refund == evm_gas_used is permitted (canonical_gas_used ends up at zero).
    let ok = executor
        .verifier_post_exec_refund_for_tx(2, false, false, 50_000)
        .expect("refund equal to evm_gas_used is permitted");
    assert_eq!(ok, 50_000);
}

#[test]
fn test_verifier_returns_zero_when_no_entry_for_tx() {
    // Deposit and post-exec cases guard against the inverse-ordering regression: every
    // block calls this helper for every deposit and for the synthetic 0x7D tx, so the
    // is_deposit / is_post_exec error branches must only fire when a payload entry actually
    // targets that tx index. If those branches are checked before the entry-existence guard,
    // every block fails at its first deposit (and at the synthetic tx).
    for (label, tx_index, is_deposit, is_post_exec) in [
        ("normal tx with no payload entry", 3, false, false),
        ("deposit tx with no payload entry", 3, true, false),
        ("post-exec tx with no payload entry", 3, false, true),
    ] {
        let mut fixture = JovianExecutorFixture::default();
        let executor = fixture.verifier(0, vec![SDMGasEntry { index: 7, gas_refund: 42 }]);

        let refund = executor
            .verifier_post_exec_refund_for_tx(tx_index, is_deposit, is_post_exec, 21_000)
            .unwrap_or_else(|err| panic!("{label}: expected no refund, got error: {err:?}"));
        assert_eq!(refund, 0, "{label}");
    }
}

#[test]
fn test_finish_reports_all_unconsumed_post_exec_entries() {
    let mut fixture = JovianExecutorFixture::default();
    let executor = fixture.verifier(
        0,
        vec![SDMGasEntry { index: 2, gas_refund: 7 }, SDMGasEntry { index: 5, gas_refund: 11 }],
    );

    let Err(err) = executor.finish() else {
        panic!("unconsumed verifier entries must fail");
    };
    assert_invalid_post_exec(err, "2 unconsumed post-exec payload entries for tx indexes [2, 5]");
}

#[test]
fn test_finish_rejects_verify_block_missing_post_exec_tx() {
    const BLOCK_GAS_LIMIT: u64 = 100_000;
    let target = Address::from([0x11; 20]);
    let tx0 = legacy_tx(0, target);
    let tx1 = legacy_tx(1, target);
    // Refund the second tx so its verifier entry is consumed during normal settlement.
    let entries = full_refund_for_second_tx(BLOCK_GAS_LIMIT, &tx0, &tx1);

    let mut fixture = JovianExecutorFixture::new(
        DEFAULT_DA_FOOTPRINT_GAS_SCALAR,
        BLOCK_GAS_LIMIT,
        JOVIAN_TIMESTAMP,
    );
    let mut verifier = fixture.verifier(0, entries);
    verifier.execute_transaction(&tx0).expect("first tx executes");
    verifier.execute_transaction(&tx1).expect("refunded tx consumes its verifier entry");
    assert!(
        verifier.post_exec.remaining_verifier_indexes().is_empty(),
        "the refunded tx must already have drained every verifier entry",
    );

    // The 0x7D was never executed, so the unconsumed-entries check cannot catch this.
    let Err(err) = verifier.finish() else {
        panic!("a Verify block that applies refunds but omits the 0x7D must be rejected");
    };
    assert_invalid_post_exec(err, "post-exec payload present but block carries no post-exec tx");
}

#[test]
fn test_disabled_mode_rejects_post_exec_tx() {
    let mut fixture = JovianExecutorFixture::default();
    // build_executor leaves post_exec_mode at the default (Disabled).
    let mut executor = fixture.executor();
    assert!(matches!(executor.post_exec, PostExecState::Disabled));

    let tx = recovered_post_exec(0, vec![]);
    let err = executor.execute_transaction(&tx).expect_err("0x7D tx in Disabled mode must fail");
    assert_invalid_post_exec(
        err,
        "unexpected post-exec tx at index 0: SDM not active for this block",
    );
}
