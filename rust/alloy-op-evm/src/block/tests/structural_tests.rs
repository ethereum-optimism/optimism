use super::*;
use crate::post_exec::NullRefundPolicy;

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

    fn inspect_call_end<CTX>(
        &mut self,
        _context: &mut CTX,
        _inputs: &CallInputs,
        _outcome: &CallOutcome,
    ) where
        CTX: ContextTr<Journal: JournalExt>,
    {
    }

    fn inspect_create<CTX>(&mut self, _context: &mut CTX, _inputs: &mut CreateInputs)
    where
        CTX: ContextTr<Journal: JournalExt>,
    {
    }

    fn inspect_create_end<CTX>(
        &mut self,
        _context: &mut CTX,
        _inputs: &CreateInputs,
        _outcome: &CreateOutcome,
    ) where
        CTX: ContextTr<Journal: JournalExt>,
    {
    }

    fn inspect_selfdestruct(&mut self, _contract: Address, _target: Address, _value: U256) {}

    fn snapshot(&self) -> Self::Snapshot {}

    fn restore(&mut self, _snapshot: Self::Snapshot) {}
}

#[test]
fn canonicalized_result_gas_applies_refund_exactly_once() {
    let mut db = prepare_observer_db();
    let receipt_builder = OpAlloyReceiptBuilder::default();
    let hardforks = OpChainHardforks::op_mainnet();
    let mut executor =
        build_policy_executor::<FixedRefundPolicy>(&mut db, &receipt_builder, &hardforks);

    let output = executor
        .execute_transaction_without_commit(&observer_test_tx())
        .expect("refunded workload executes");

    let refund = output.post_exec.as_ref().expect("refund adjustment present").refund;
    assert_eq!(refund, 1);
    let gas = output.inner.result.result.gas();
    // The EIP-7623 floor must stay below the refunded gas, or `tx_gas_used()` would pin at
    // the floor and mask a double-subtracted refund.
    assert!(
        output.evm_gas_used.saturating_sub(2 * refund) > gas.floor_gas(),
        "workload too cheap to distinguish a double-subtracted refund from the floor"
    );
    assert_eq!(output.canonical_gas_used, output.evm_gas_used - refund);

    // The exposed ExecutionResult must agree with the canonical gas the block accounts:
    // the refund applied exactly once, not twice.
    assert_eq!(
        gas.tx_gas_used(),
        output.canonical_gas_used,
        "canonicalized result gas must equal canonical_gas_used"
    );
    // The refund must lower `total_gas_spent`, not inflate the EVM's own refund counter —
    // this workload earns no EVM refund, so any value here is SDM leakage.
    assert_eq!(gas.inner_refunded(), 0, "post-exec refund folded into the EVM refund counter");

    let canonical_gas_used = output.canonical_gas_used;
    executor.commit_transaction(output);
    assert_eq!(executor.gas_used, canonical_gas_used, "block must accumulate canonical gas");
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

/// Every account that can hold ETH over the block audited by
/// [`fixed_policy_settlement_conserves_total_eth_supply`].
fn eth_holding_universe(beneficiary: Address, target: Address) -> [Address; 7] {
    [
        Address::ZERO, // the transaction sender
        beneficiary,
        target,
        L1_BLOCK_CONTRACT,
        L1_FEE_RECIPIENT,
        BASE_FEE_RECIPIENT,
        OPERATOR_FEE_RECIPIENT,
    ]
}

fn sum_balances(db: &mut State<InMemoryDB>, addrs: &[Address]) -> U256 {
    addrs.iter().fold(U256::ZERO, |acc, addr| {
        let balance = revm::Database::basic(db, *addr)
            .expect("load account")
            .map(|info| info.balance)
            .unwrap_or_default();
        acc + balance
    })
}

/// Runs an identical two-transaction block under refund policy `R`, appending the produced
/// post-exec (`0x7D`) transaction when the policy produced any entries. Returns the total ETH
/// balance over `universe` after the block finishes, the sender's balance alone, and the entries.
///
/// Settlement is applied per normal transaction as it executes (see `is_producing` in
/// `block/mod.rs`), *not* by the trailing `0x7D` transaction — which only records the entries for
/// consensus. So the policy, not the presence of the `0x7D` tx, is what toggles settlement.
fn run_block_and_total<R>(
    beneficiary: Address,
    target: Address,
    universe: &[Address],
) -> (U256, U256, Vec<SDMGasEntry>)
where
    R: Default + PostExecRefundInspector,
{
    const BASE_FEE: u64 = 7;
    // Priced above basefee so the priority-fee share that settlement moves between the sender and
    // the beneficiary is non-trivial.
    const GAS_PRICE: u128 = 100;

    let mut db = prepare_observer_db();
    let receipt_builder = OpAlloyReceiptBuilder::default();
    let hardforks = OpChainHardforks::op_mainnet();

    let entries = {
        let mut producer = build_policy_executor_with::<R>(
            &mut db,
            &receipt_builder,
            &hardforks,
            BASE_FEE,
            beneficiary,
            Inspect::Enabled,
        );

        producer
            .execute_transaction(&legacy_tx_with_price(0, target, 50_000, GAS_PRICE))
            .expect("tx0 executes");
        producer
            .execute_transaction(&legacy_tx_with_price(1, target, 50_000, GAS_PRICE))
            .expect("tx1 executes");

        let entries = producer.take_post_exec_entries();
        if !entries.is_empty() {
            let post_exec = recovered_post_exec(0, entries.clone());
            producer.execute_transaction(&post_exec).expect("producer executes post-exec tx");
        }
        // Dropping the EVM releases its borrow of `db` so the audit can read balances.
        let (evm, _result) = producer.finish().expect("producer finishes the block");
        drop(evm);
        entries
    };

    (sum_balances(&mut db, universe), sum_balances(&mut db, &[Address::ZERO]), entries)
}

// State-level ETH-supply audit of post-exec settlement. The sibling conservation tests pin the
// settlement *delta computation*; this one pins the same conservation on applied account state.
//
// The identical two-tx block runs twice — once under a refunding policy, once under
// `NullRefundPolicy` — and every account that can hold ETH is summed. Settlement only moves ETH
// between the sender and the fee recipients, so the total must match the unsettled run exactly:
// no ETH is minted or burned. Comparing two runs cancels block-level effects (e.g. the coinbase
// reward) so the assertion isolates the settlement.
//
// The audit is policy-independent — it needs only a policy that yields a non-zero refund, which
// `FixedRefundPolicy` provides.
#[test]
fn fixed_policy_settlement_conserves_total_eth_supply() {
    let beneficiary = Address::from([0x99; 20]);
    let target = Address::from([0x77; 20]);
    let universe = eth_holding_universe(beneficiary, target);

    let (total_settled, sender_settled, entries) =
        run_block_and_total::<FixedRefundPolicy>(beneficiary, target, &universe);
    let (total_unsettled, sender_unsettled, null_entries) =
        run_block_and_total::<NullRefundPolicy>(beneficiary, target, &universe);

    assert!(
        entries.iter().any(|e| e.gas_refund > 0),
        "the block under audit must actually settle a non-zero refund",
    );
    assert!(null_entries.is_empty(), "the control block must settle nothing");
    // Guards the audit against vacuity: settlement must have moved ETH *somewhere*, otherwise the
    // conservation assertion below would hold trivially for a settlement that did nothing.
    assert_ne!(
        sender_settled, sender_unsettled,
        "settlement must credit the refunding sender, otherwise the audit proves nothing",
    );
    assert_eq!(
        total_settled, total_unsettled,
        "SDM post-exec settlement minted or burned ETH relative to the unsettled block",
    );
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

    fn inspect_call_end<CTX>(
        &mut self,
        _context: &mut CTX,
        _inputs: &CallInputs,
        _outcome: &CallOutcome,
    ) where
        CTX: ContextTr<Journal: JournalExt>,
    {
    }

    fn inspect_create<CTX>(&mut self, _context: &mut CTX, _inputs: &mut CreateInputs)
    where
        CTX: ContextTr<Journal: JournalExt>,
    {
    }

    fn inspect_create_end<CTX>(
        &mut self,
        _context: &mut CTX,
        _inputs: &CreateInputs,
        _outcome: &CreateOutcome,
    ) where
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

/// Post-exec (`0x7D`) transactions do not consume DA footprint.
///
/// End-to-end pin on the observable behaviour, not on any single guard: the exclusion is
/// over-determined (the `0x7D` returns early with `blob_gas_used: 0`, the accumulator skips
/// `is_post_exec`, and `encoded_tx_da_footprint` returns zero for the type byte), so this test
/// survives the removal of any one of them. It exists because op-geth used to recompute the
/// footprint independently and act as a cross-check; it is no longer a supported verifier.
#[test]
fn test_post_exec_tx_does_not_accrue_da_footprint() {
    let mut fixture = JovianExecutorFixture::default();
    let mut producer = fixture.executor_with_post_exec_mode(PostExecMode::Produce);

    let user_tx = recovered_legacy(TxLegacy { gas_limit: DEFAULT_GAS_LIMIT, ..Default::default() });
    producer.execute_transaction(&user_tx).expect("producer executes user tx");

    let footprint_before = producer.da_footprint_used;
    assert!(footprint_before > 0, "the user tx must accrue a footprint for this test to bite");

    let post_exec = recovered_post_exec(0, vec![SDMGasEntry { index: 0, gas_refund: 7 }]);

    // The minimum-size floor ensures a counted 0x7D would have a non-zero footprint, so the
    // assertion below is about the exclusion and not about a 0x7D that is simply too small to
    // register.
    let counted_footprint =
        op_revm::estimate_tx_compressed_size(post_exec.tx().encoded_2718().as_ref())
            .saturating_div(1_000_000)
            .saturating_mul(DEFAULT_DA_FOOTPRINT_GAS_SCALAR.into());
    assert!(counted_footprint > 0, "a counted 0x7D would have added a non-zero footprint");

    producer.execute_transaction(&post_exec).expect("producer executes the 0x7D");
    assert_eq!(
        producer.da_footprint_used, footprint_before,
        "the 0x7D must not accrue DA footprint",
    );

    let (_, result) = producer.finish().expect("producer finishes block");
    assert_eq!(result.blob_gas_used, footprint_before);
    assert_eq!(result.receipts.len(), 2, "the 0x7D still gets a receipt");
}

/// Regression coverage for state leaked by a dropped transaction.
///
/// The first three tests cover a warm-set leak in op-revm's `catch_error`. `catch_error` must
/// discard the journal on a non-deposit tx error, as upstream revm's `EthHandler::catch_error`
/// does. Were it not to, then on the SDM Produce execution path (`alloy-op-evm` routes to
/// `inspect_tx`, which skips `finalize()` on error) a failed candidate tx would leave its sender
/// EIP-2929-warm in the shared journal, so a later tx executed on the same EVM would be mischarged
/// — diverging from a validator that never executed the failed tx. Those tests pin that the leak
/// does not occur, and that the current production (SDM-disabled) configuration is unaffected
/// either way.
///
/// The journal-warmth path is policy-independent: Produce mode routes through `inspect_tx` whatever
/// the policy, and the assertion is on `B`'s EIP-2929 gas, not on any refund.
/// [`FixedRefundPolicy`] serves only to make the block carry a `0x7D` payload, as a real SDM block
/// does. op-revm's `catch_error_tests.rs` keeps the unit matrix for that invariant; the first two
/// tests here are the end-to-end builder-vs-validator pin.
///
/// The final two tests cover a separate leak in the producer policy's block-scoped state. They use
/// a stateful policy, assert on its snapshot rather than probe transaction gas, and specifically
/// pin this crate's candidate-rollback wrapper; op-revm's unit matrix does not cover that
/// invariant.
mod warm_set_leak {
    use super::*;
    use alloc::collections::BTreeSet;
    use revm::context::result::InvalidTransaction;

    /// Sender of the failing tx `A` — loaded+warmed during validation before the nonce check
    /// rejects it. Its leaked warmth is the bug.
    const LEAK_ADDR: Address = Address::new([0xAA; 20]);
    /// Sender of the probe tx `B`.
    const PROBE_SENDER: Address = Address::new([0xBB; 20]);
    /// Unique policy-state marker used to distinguish snapshot restoration from clearing state.
    const POLICY_SENTINEL: Address = Address::new([0xDD; 20]);
    /// Probe contract: `PUSH20 <LEAK_ADDR>; BALANCE; POP; STOP` — a Berlin account access on `A`'s
    /// sender, charged 100 (warm) or 2600 (cold).
    const PROBE_CONTRACT: Address = Address::new([0xCC; 20]);

    fn nonce_too_low_db() -> State<InMemoryDB> {
        // `A`'s sender: nonce 5 so `A` (nonce 0) is rejected `NonceTooLow` *after* the sender is
        // loaded+warmed, leaving no other state mutation (crisp signal).
        db_with_leak_account(AccountInfo {
            nonce: 5,
            balance: U256::from(1_000_000_000u64),
            ..Default::default()
        })
    }

    /// Like [`nonce_too_low_db`] but `A`'s sender carries code, so `A` is rejected by EIP-3607
    /// (sender-has-code) instead of by the nonce check — still only after it is loaded+warmed.
    fn contract_sender_db() -> State<InMemoryDB> {
        let code = Bytecode::new_raw(vec![0x00u8].into()); // STOP
        // nonce 0 so the nonce check passes and EIP-3607 is the sole rejection reason.
        db_with_leak_account(AccountInfo {
            code_hash: code.hash_slow(),
            code: Some(code),
            balance: U256::from(1_000_000_000u64),
            ..Default::default()
        })
    }

    /// Seeds the shared warm-leak accounts; `A`'s sender (`LEAK_ADDR`) is supplied by the caller so
    /// each test can choose the validation error `A` fails with.
    fn db_with_leak_account(leak_account: AccountInfo) -> State<InMemoryDB> {
        let mut db = prepare_jovian_db(DEFAULT_DA_FOOTPRINT_GAS_SCALAR);
        db.insert_account(LEAK_ADDR, leak_account);
        // `B`'s sender: funded.
        db.insert_account(
            PROBE_SENDER,
            AccountInfo {
                balance: U256::from(1_000_000_000_000_000_000u128),
                ..Default::default()
            },
        );
        // Probe contract bytecode: `PUSH20 <LEAK_ADDR>; BALANCE; POP; STOP`.
        let mut code = vec![0x73u8]; // PUSH20
        code.extend_from_slice(LEAK_ADDR.as_slice());
        code.extend_from_slice(&[0x31, 0x50, 0x00]); // BALANCE, POP, STOP
        let bytecode = Bytecode::new_raw(code.into());
        db.insert_account(
            PROBE_CONTRACT,
            AccountInfo {
                code_hash: bytecode.hash_slow(),
                code: Some(bytecode),
                ..Default::default()
            },
        );
        db
    }

    fn legacy_with_sender(
        sender: Address,
        nonce: u64,
        to: Address,
        gas_limit: u64,
    ) -> Recovered<OpTxEnvelope> {
        recovered_legacy_from(
            sender,
            TxLegacy { nonce, gas_limit, gas_price: 0, to: TxKind::Call(to), ..Default::default() },
        )
    }

    /// Activates every fork through Lagoon at `JOVIAN_TIMESTAMP`. Lagoon gates SDM, so the Produce
    /// path exercised here runs on a genuinely SDM-active fork rather than being forced on a
    /// pre-SDM one.
    fn hardforks() -> OpChainHardforks {
        let forks = OpHardfork::op_mainnet()
            .into_iter()
            .filter(|(fork, _)| fork.idx() < OpHardfork::Jovian.idx())
            .chain(
                OpHardfork::Jovian
                    .forks_from()
                    .take_while(|fork| fork.idx() <= OpHardfork::Lagoon.idx())
                    .map(|fork| (fork, ForkCondition::Timestamp(JOVIAN_TIMESTAMP))),
            );
        OpChainHardforks::new(forks)
    }

    fn probe_tx() -> Recovered<OpTxEnvelope> {
        legacy_with_sender(PROBE_SENDER, 0, PROBE_CONTRACT, 200_000)
    }

    /// Canonical gas charged to probe tx `B` (BALANCE on `LEAK_ADDR`), optionally preceded by a
    /// failing tx `A` from `LEAK_ADDR` that is attempted then skipped on the same executor —
    /// mirroring the builder's skip-and-continue loop.
    fn probe_b_canonical_gas(mode: PostExecMode, include_failing_a: bool) -> u64 {
        let mut db = nonce_too_low_db();
        let receipt_builder = OpAlloyReceiptBuilder::default();
        let op_chain_hardforks = hardforks();
        let mut executor = build_policy_executor_with::<FixedRefundPolicy>(
            &mut db,
            &receipt_builder,
            &op_chain_hardforks,
            0,
            Address::ZERO,
            Inspect::Disabled,
        );
        executor.set_post_exec_mode(mode);

        if include_failing_a {
            let tx_a = legacy_with_sender(LEAK_ADDR, 0, PROBE_SENDER, 50_000);
            executor
                .execute_transaction(&tx_a)
                .expect_err("tx A must fail NonceTooLow and be skipped");
        }

        executor.execute_transaction(&probe_tx()).expect("probe tx B executes").tx_gas_used()
    }

    /// Drives the builder-vs-validator consensus scenario: a builder in SDM `Produce` mode attempts
    /// `failing_a` (which must error and be skipped), then includes probe tx `B` and the `0x7D`
    /// refund payload; a validator in `Verify` mode re-executes only `B` and the payload, never
    /// `A`. Asserts the sealed-block gas matches — i.e. the skipped tx left no warm residue for the
    /// builder to bill `B` for.
    fn assert_no_builder_validator_divergence(
        make_db: impl Fn() -> State<InMemoryDB>,
        failing_a: Recovered<OpTxEnvelope>,
        a_error_context: &str,
    ) {
        let receipt_builder = OpAlloyReceiptBuilder::default();
        let op_chain_hardforks = hardforks();

        // ---- Builder (SDM Produce): A is attempted, fails, and is skipped; B is included.
        let mut producer_db = make_db();
        let mut producer = build_policy_executor_with::<FixedRefundPolicy>(
            &mut producer_db,
            &receipt_builder,
            &op_chain_hardforks,
            0,
            Address::ZERO,
            Inspect::Disabled,
        );
        producer.execute_transaction(&failing_a).expect_err(a_error_context);
        producer.execute_transaction(&probe_tx()).expect("probe tx B executes");
        let entries = producer.take_post_exec_entries();
        let post_exec_tx = recovered_post_exec(0, entries.clone());
        producer.execute_transaction(&post_exec_tx).expect("producer appends 0x7D tx");
        let (_, produced) = producer.finish().expect("producer finishes block");

        // ---- Validator (SDM Verify of the sealed block): only B and the 0x7D tx; never A.
        let mut verifier_db = make_db();
        let mut verifier = build_policy_executor_with::<FixedRefundPolicy>(
            &mut verifier_db,
            &receipt_builder,
            &op_chain_hardforks,
            0,
            Address::ZERO,
            Inspect::Disabled,
        );
        verifier.set_post_exec_mode(PostExecMode::Verify(PostExecPayload {
            version: 1,
            block_number: 0,
            gas_refund_entries: entries,
        }));
        verifier.execute_transaction(&probe_tx()).expect("validator executes B");
        verifier.execute_transaction(&post_exec_tx).expect("validator consumes 0x7D tx");
        let (_, verified) = verifier.finish().expect("validator finishes block");

        assert_eq!(
            produced.gas_used,
            verified.gas_used,
            "builder built B against a leaked-warm sender (from skipped tx A) but the validator \
             re-executes B cold: {}-gas builder-vs-validator divergence -> block rejected. The \
             failed tx's journal must be discarded on the error path",
            produced.gas_used.abs_diff(verified.gas_used),
        );
    }

    /// The faithful builder-vs-validator consensus test.
    ///
    /// A *builder* in SDM Produce mode runs `[A (fails, skipped), B]` on one EVM and seals a block
    /// containing only `B` plus the `0x7D` refund payload. A *validator* re-executes that block in
    /// Verify mode — it never runs `A`. Were `OpHandler::catch_error` not to discard the failed
    /// tx's journal, the builder would charge `B` warm (100) for the access to `A`'s leaked-warm
    /// sender while the validator charges it cold (2600), so `produced.gas_used !=
    /// verified.gas_used` (a 2500-gas divergence) and the validator rejects the block — a chain
    /// halt.
    ///
    /// Note: the SDM refund the builder emits for `B` is *trusted* by the validator (it only
    /// bounds-checks the payload), so it is applied identically on both sides and does NOT
    /// contribute to this divergence.
    #[test]
    fn skipped_failed_tx_in_sdm_produce_does_not_diverge_builder_vs_validator() {
        // `A`'s sender has nonce 5, so `A` (nonce 0) is rejected `NonceTooLow` after being
        // loaded+warmed during validation.
        assert_no_builder_validator_divergence(
            nonce_too_low_db,
            legacy_with_sender(LEAK_ADDR, 0, PROBE_SENDER, 50_000),
            "tx A must fail NonceTooLow and be skipped",
        );
    }

    /// EIP-3607 variant of the divergence test: `A` is rejected because its sender carries code (a
    /// contract may not originate a tx), not because of a stale nonce. revm still loads and warms
    /// the sender before that check, so a different validation error must discard the same warmth —
    /// otherwise the builder over-warms `B` and diverges from the validator exactly as in the
    /// `NonceTooLow` case.
    #[test]
    fn skipped_eip3607_failed_tx_in_sdm_produce_does_not_diverge_builder_vs_validator() {
        assert_no_builder_validator_divergence(
            contract_sender_db,
            legacy_with_sender(LEAK_ADDR, 0, PROBE_SENDER, 50_000),
            "tx A must fail EIP-3607 (contract sender) and be skipped",
        );
    }

    /// With SDM disabled (production config: Karst pre-Lagoon, opt-in OFF) execution uses the
    /// `transact` path, whose `finalize()` runs even on error and wipes the failed tx's journal —
    /// so a skipped failing tx cannot affect `B` regardless of whether `OpHandler::catch_error`
    /// discards the journal. This guards today's chains' safety.
    #[test]
    fn skipped_failed_tx_does_not_affect_next_tx_when_sdm_disabled() {
        let with_failed_a = probe_b_canonical_gas(PostExecMode::Disabled, true);
        let without = probe_b_canonical_gas(PostExecMode::Disabled, false);
        assert_eq!(
            without, with_failed_a,
            "with SDM disabled (prod config) a skipped failing tx must not affect the next tx",
        );
    }

    // Leak B: unlike the journal-warmth leak above, this rides the producer *policy's* block-scoped
    // state. `FixedRefundPolicy` is stateless, so it cannot detect it; use a policy whose only
    // mutation is in `note_account_touch` — the `transact_raw` error-path call
    // (`note_post_exec_account_touch`) a dropped tx triggers.
    #[derive(Debug, Clone, Default)]
    struct FeeVaultTouchPolicy {
        touched: BTreeSet<Address>,
    }

    impl PostExecRefundInspector for FeeVaultTouchPolicy {
        type Snapshot = BTreeSet<Address>;

        fn begin_tx(&mut self, _ctx: PostExecTxContext) {}

        fn note_account_touch(&mut self, address: Address) {
            self.touched.insert(address);
        }

        fn finish_tx(&mut self) -> PostExecExecutedTx {
            PostExecExecutedTx::default()
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

        fn inspect_call_end<CTX>(
            &mut self,
            _context: &mut CTX,
            _inputs: &CallInputs,
            _outcome: &CallOutcome,
        ) where
            CTX: ContextTr<Journal: JournalExt>,
        {
        }

        fn inspect_create<CTX>(&mut self, _context: &mut CTX, _inputs: &mut CreateInputs)
        where
            CTX: ContextTr<Journal: JournalExt>,
        {
        }

        fn inspect_create_end<CTX>(
            &mut self,
            _context: &mut CTX,
            _inputs: &CreateInputs,
            _outcome: &CreateOutcome,
        ) where
            CTX: ContextTr<Journal: JournalExt>,
        {
        }

        fn inspect_selfdestruct(&mut self, _contract: Address, _target: Address, _value: U256) {}

        fn snapshot(&self) -> Self::Snapshot {
            self.touched.clone()
        }

        fn restore(&mut self, snapshot: Self::Snapshot) {
            self.touched = snapshot;
        }
    }

    fn assert_invalid_transaction(err: BlockExecutionError, expected: InvalidTransaction) {
        match err {
            BlockExecutionError::Validation(BlockValidationError::InvalidTx { error, .. }) => {
                assert_eq!(error.as_invalid_tx_err(), Some(&expected));
            }
            other => panic!("expected invalid transaction {expected:?}, got: {other:?}"),
        }
    }

    /// A dropped state-invalid tx must restore a stateful producer policy to its per-candidate
    /// snapshot. The sentinel proves the wrapper restores rather than clears prior block-scoped
    /// state; the successful probe afterward proves this fixture exercises fee-vault touch
    /// tracking. Removing the `Err`-branch restore leaves the failed tx's touches alongside the
    /// sentinel and fails the exact-snapshot assertion.
    fn assert_dropped_tx_leaves_no_policy_touch(
        make_db: impl Fn() -> State<InMemoryDB>,
        expected_error: InvalidTransaction,
        a_error_context: &str,
    ) {
        let mut db = make_db();
        let receipt_builder = OpAlloyReceiptBuilder::default();
        let op_chain_hardforks = hardforks();
        let mut executor = build_policy_executor_with::<FeeVaultTouchPolicy>(
            &mut db,
            &receipt_builder,
            &op_chain_hardforks,
            0,
            Address::ZERO,
            Inspect::Disabled,
        );

        let sentinel_snapshot = BTreeSet::from([POLICY_SENTINEL]);
        executor.seed_refund_snapshot(sentinel_snapshot.clone());
        assert_eq!(executor.refund_snapshot(), sentinel_snapshot);

        let err = executor
            .execute_transaction(&legacy_with_sender(LEAK_ADDR, 0, PROBE_SENDER, 50_000))
            .expect_err(a_error_context);
        assert_invalid_transaction(err, expected_error);
        assert_eq!(
            executor.refund_snapshot(),
            sentinel_snapshot,
            "a dropped failing tx did not restore the producer policy's per-candidate snapshot",
        );

        executor.execute_transaction(&probe_tx()).expect("probe tx B executes");
        let expected_committed_snapshot = BTreeSet::from([
            POLICY_SENTINEL,
            L1_FEE_RECIPIENT,
            BASE_FEE_RECIPIENT,
            OPERATOR_FEE_RECIPIENT,
        ]);
        assert_eq!(
            executor.refund_snapshot(),
            expected_committed_snapshot,
            "a committed tx must exercise and retain the fee-vault touch pathway",
        );
    }

    #[test]
    fn dropped_nonce_too_low_tx_leaves_no_fee_vault_touch_in_policy() {
        assert_dropped_tx_leaves_no_policy_touch(
            nonce_too_low_db,
            InvalidTransaction::NonceTooLow { tx: 0, state: 5 },
            "tx A must fail NonceTooLow and be skipped",
        );
    }

    #[test]
    fn dropped_eip3607_tx_leaves_no_fee_vault_touch_in_policy() {
        assert_dropped_tx_leaves_no_policy_touch(
            contract_sender_db,
            InvalidTransaction::RejectCallerWithCode,
            "tx A must fail EIP-3607 (contract sender) and be skipped",
        );
    }
}
