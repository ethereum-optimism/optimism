use alloc::{string::ToString, vec};
use alloy_consensus::{Sealable, Sealed, SignableTransaction, TxLegacy, transaction::Recovered};
use alloy_eips::eip2718::WithEncoded;
use alloy_evm::{EvmEnv, ToTxEnv};
use alloy_hardforks::ForkCondition;
use alloy_op_hardforks::{OpHardfork, OpHardforks};
use alloy_primitives::{Address, B256, Bytes, Signature, TxKind, U256, keccak256, uint};
use op_alloy::consensus::{
    OpTxEnvelope, PostExecPayload, SDMGasEntry, TxDeposit, build_post_exec_tx,
};
use op_revm::{
    L1BlockInfo, OpBuilder, OpSpecId, OpTransaction,
    constants::{
        BASE_FEE_SCALAR_OFFSET, ECOTONE_L1_BLOB_BASE_FEE_SLOT, ECOTONE_L1_FEE_SCALARS_SLOT,
        L1_BASE_FEE_SLOT, L1_BLOCK_CONTRACT, L1_FEE_RECIPIENT, OPERATOR_FEE_SCALARS_SLOT,
    },
};
use revm::{
    Context, MainContext,
    context::{BlockEnv, CfgEnv},
    context_interface::ContextTr,
    database::{CacheDB, EmptyDB, InMemoryDB, State},
    inspector::{JournalExt, NoOpInspector},
    interpreter::{CallInputs, CallOutcome, CreateInputs, CreateOutcome, Interpreter},
    primitives::HashMap,
    state::{Account, AccountInfo, Bytecode, EvmState},
};

use crate::{
    OpEvm,
    post_exec::{PostExecExecutedTx, PostExecRefundInspector, PostExecTxContext},
};

use super::*;

/// Wraps a `TxLegacy` in an `OpTxEnvelope::Legacy` recovered with a zero signer.
fn recovered_legacy(tx: TxLegacy) -> Recovered<OpTxEnvelope> {
    recovered_legacy_from(Address::ZERO, tx)
}

/// Wraps a `TxLegacy` in an `OpTxEnvelope::Legacy` recovered with the given signer.
fn recovered_legacy_from(sender: Address, tx: TxLegacy) -> Recovered<OpTxEnvelope> {
    Recovered::new_unchecked(
        OpTxEnvelope::Legacy(tx.into_signed(Signature::new(
            Default::default(),
            Default::default(),
            Default::default(),
        ))),
        sender,
    )
}

#[test]
fn test_with_encoded() {
    let executor_factory = OpBlockExecutorFactory::new(
        OpAlloyReceiptBuilder::default(),
        OpChainHardforks::op_mainnet(),
        OpEvmFactory::<crate::OpTx>::default(),
    );
    let mut db = State::builder().with_database(CacheDB::<EmptyDB>::default()).build();
    let evm = executor_factory.evm_factory.create_evm(&mut db, EvmEnv::default());
    let mut executor = executor_factory.create_executor(evm, OpBlockExecutionCtx::default());
    let tx = recovered_legacy(TxLegacy::default());
    let tx_with_encoded = WithEncoded::new(tx.encoded_2718().into(), tx.clone());

    // make sure we can use both `WithEncoded` and transaction itself as inputs.
    let _ = executor.execute_transaction(&tx);
    let _ = executor.execute_transaction(&tx_with_encoded);
}

fn prepare_jovian_db(da_footprint_gas_scalar: u16) -> State<InMemoryDB> {
    const L1_BASE_FEE: U256 = uint!(1_U256);
    const L1_BLOB_BASE_FEE: U256 = uint!(2_U256);
    const L1_BASE_FEE_SCALAR: u64 = 3;
    const L1_BLOB_BASE_FEE_SCALAR: u64 = 4;
    const L1_FEE_SCALARS: U256 = U256::from_limbs([
        0,
        (L1_BASE_FEE_SCALAR << (64 - BASE_FEE_SCALAR_OFFSET * 2)) | L1_BLOB_BASE_FEE_SCALAR,
        0,
        0,
    ]);
    const OPERATOR_FEE_SCALAR: u8 = 5;
    const OPERATOR_FEE_CONST: u8 = 6;
    let da_footprint_gas_scalar_bytes = da_footprint_gas_scalar.to_be_bytes();
    let mut operator_fee_and_da_footprint = [0u8; 32];
    operator_fee_and_da_footprint[31] = OPERATOR_FEE_CONST;
    operator_fee_and_da_footprint[23] = OPERATOR_FEE_SCALAR;
    operator_fee_and_da_footprint[19] = da_footprint_gas_scalar_bytes[1];
    operator_fee_and_da_footprint[18] = da_footprint_gas_scalar_bytes[0];
    let operator_fee_and_da_footprint_u256 = U256::from_be_bytes(operator_fee_and_da_footprint);

    let mut db = State::builder().with_database(InMemoryDB::default()).build();

    db.insert_account_with_storage(
        L1_BLOCK_CONTRACT,
        Default::default(),
        HashMap::from_iter([
            (L1_BASE_FEE_SLOT, L1_BASE_FEE),
            (ECOTONE_L1_FEE_SCALARS_SLOT, L1_FEE_SCALARS),
            (ECOTONE_L1_BLOB_BASE_FEE_SLOT, L1_BLOB_BASE_FEE),
            (OPERATOR_FEE_SCALARS_SLOT, operator_fee_and_da_footprint_u256),
        ]),
    );

    db.insert_account(
        Address::ZERO,
        AccountInfo { balance: U256::from(400_000_000), ..Default::default() },
    );

    db
}

type JovianTestExecutor<'a> = OpBlockExecutor<
    OpEvm<
        &'a mut State<InMemoryDB>,
        NoOpInspector,
        op_revm::precompiles::OpPrecompiles,
        crate::OpTx,
    >,
    &'a OpAlloyReceiptBuilder,
    &'a OpChainHardforks,
>;

const DEFAULT_DA_FOOTPRINT_GAS_SCALAR: u16 = 7;
const DEFAULT_GAS_LIMIT: u64 = 100_000;
const JOVIAN_TIMESTAMP: u64 = 1_746_806_402;

/// Whether [`build_executor`] attaches an inspector to the EVM — a self-documenting alternative to
/// a bare `bool` at the call sites.
enum Inspect {
    Enabled,
    /// Models real block building and validation, which construct the EVM without an inspector
    /// (`create_evm` passes `false`). The inspected `inspect_tx` path is then reached *only* via
    /// SDM Produce's `begin_post_exec_tx`, exactly as in production.
    Disabled,
}

#[allow(clippy::too_many_arguments)]
fn build_executor<'a>(
    db: &'a mut State<InMemoryDB>,
    receipt_builder: &'a OpAlloyReceiptBuilder,
    op_chain_hardforks: &'a OpChainHardforks,
    gas_limit: u64,
    block_timestamp: u64,
    parent_timestamp: Option<u64>,
    base_fee: u64,
    beneficiary: Address,
    inspect: Inspect,
) -> JovianTestExecutor<'a> {
    let ctx = Context::mainnet()
        .with_tx(crate::OpTx(OpTransaction::builder().build_fill()))
        .with_cfg(CfgEnv::new_with_spec(OpSpecId::BEDROCK))
        .with_chain(L1BlockInfo::default())
        .with_db(db)
        .with_chain(L1BlockInfo {
            operator_fee_scalar: Some(U256::from(2)),
            operator_fee_constant: Some(U256::from(50)),
            ..Default::default()
        })
        .with_block(BlockEnv {
            timestamp: U256::from(block_timestamp),
            gas_limit,
            basefee: base_fee,
            beneficiary,
            ..Default::default()
        })
        .modify_cfg_chained(|cfg| cfg.spec = OpSpecId::JOVIAN);

    let evm = OpEvm::new(
        ctx.build_op_with_inspector(NoOpInspector {}),
        matches!(inspect, Inspect::Enabled),
    );

    // Like production call sites, the activation-block flag is computed where the parent
    // timestamp is available and left `false` where it isn't.
    let no_user_tx_activation_block = parent_timestamp.is_some_and(|parent_timestamp| {
        op_chain_hardforks.is_no_user_tx_activation_block(parent_timestamp, block_timestamp)
    });

    OpBlockExecutor::new(
        evm,
        OpBlockExecutionCtx { no_user_tx_activation_block, ..Default::default() },
        op_chain_hardforks,
        receipt_builder,
    )
}

struct JovianExecutorFixture {
    db: State<InMemoryDB>,
    receipt_builder: OpAlloyReceiptBuilder,
    op_chain_hardforks: OpChainHardforks,
    gas_limit: u64,
    jovian_timestamp: u64,
    parent_timestamp: Option<u64>,
    base_fee: u64,
    beneficiary: Address,
}

impl JovianExecutorFixture {
    fn new(da_footprint_gas_scalar: u16, gas_limit: u64, jovian_timestamp: u64) -> Self {
        Self {
            db: prepare_jovian_db(da_footprint_gas_scalar),
            receipt_builder: OpAlloyReceiptBuilder::default(),
            op_chain_hardforks: OpChainHardforks::new(
                OpHardfork::op_mainnet()
                    .into_iter()
                    .chain(vec![(OpHardfork::Jovian, ForkCondition::Timestamp(jovian_timestamp))]),
            ),
            gas_limit,
            jovian_timestamp,
            // These Jovian execution tests run normal (non-activation) blocks; leaving the parent
            // timestamp unset skips the fork-activation guard, matching op-reth's import path.
            parent_timestamp: None,
            // Default to a zero base fee; settlement tests opt into a non-zero one.
            base_fee: 0,
            // Default beneficiary is the zero address; coinbase-warmth tests opt into a distinct
            // one so the block beneficiary is separable from the (also-zero) default tx sender.
            beneficiary: Address::ZERO,
        }
    }

    fn executor(&mut self) -> JovianTestExecutor<'_> {
        build_executor(
            &mut self.db,
            &self.receipt_builder,
            &self.op_chain_hardforks,
            self.gas_limit,
            self.jovian_timestamp,
            self.parent_timestamp,
            self.base_fee,
            self.beneficiary,
            Inspect::Enabled,
        )
    }

    fn executor_with_post_exec_mode(
        &mut self,
        post_exec_mode: PostExecMode,
    ) -> JovianTestExecutor<'_> {
        let mut executor = self.executor();
        executor.set_post_exec_mode(post_exec_mode);
        executor
    }

    fn verifier(&mut self, block_number: u64, entries: Vec<SDMGasEntry>) -> JovianTestExecutor<'_> {
        self.executor_with_post_exec_mode(PostExecMode::Verify(PostExecPayload {
            version: 1,
            block_number,
            gas_refund_entries: entries,
        }))
    }
}

impl Default for JovianExecutorFixture {
    fn default() -> Self {
        Self::new(DEFAULT_DA_FOOTPRINT_GAS_SCALAR, DEFAULT_GAS_LIMIT, JOVIAN_TIMESTAMP)
    }
}

#[test]
fn test_jovian_da_footprint_estimation() {
    let mut fixture = JovianExecutorFixture::default();
    let mut executor = fixture.executor();
    let tx = recovered_legacy(TxLegacy { gas_limit: DEFAULT_GAS_LIMIT, ..Default::default() });
    let tx_env = tx.to_tx_env();

    let expected_da_footprint = executor.jovian_da_footprint_estimation(&tx_env, &tx).unwrap();

    executor.execute_transaction(&tx).expect("legacy tx executes");
    assert_eq!(executor.da_footprint_used, expected_da_footprint);
}

#[test]
fn test_jovian_da_footprint_estimation_out_of_gas() {
    const GAS_LIMIT: u64 = 100;

    let mut fixture =
        JovianExecutorFixture::new(DEFAULT_DA_FOOTPRINT_GAS_SCALAR, GAS_LIMIT, JOVIAN_TIMESTAMP);
    let mut executor = fixture.executor();
    let tx = recovered_legacy(TxLegacy { gas_limit: GAS_LIMIT, ..Default::default() });
    let tx_env = tx.to_tx_env();

    let expected_da_footprint = executor.jovian_da_footprint_estimation(&tx_env, &tx).unwrap();

    let err = executor.execute_transaction(&tx).expect_err("must reject when DA exceeds limit");
    match err {
        BlockExecutionError::Validation(BlockValidationError::Other(err)) => {
            assert_eq!(
                err.to_string(),
                OpBlockExecutionError::TransactionDaFootprintAboveGasLimit {
                    transaction_da_footprint: expected_da_footprint,
                    available_block_da_footprint: GAS_LIMIT,
                }
                .to_string(),
            );
        }
        _ => panic!("expected TransactionDaFootprintAboveGasLimit error"),
    }
}

#[test]
fn test_jovian_da_footprint_estimation_maxed_out_da_footprint() {
    const DA_FOOTPRINT_GAS_SCALAR: u16 = 2000;
    const GAS_LIMIT: u64 = 200_000;

    let mut fixture =
        JovianExecutorFixture::new(DA_FOOTPRINT_GAS_SCALAR, GAS_LIMIT, JOVIAN_TIMESTAMP);
    let mut executor = fixture.executor();
    let tx = recovered_legacy(TxLegacy { gas_limit: GAS_LIMIT, ..Default::default() });
    let tx_env = tx.to_tx_env();

    let expected_da_footprint = executor.jovian_da_footprint_estimation(&tx_env, &tx).unwrap();
    let gas_used_tx =
        executor.execute_transaction(&tx).expect("failed to execute transaction").tx_gas_used();

    // The legacy gas used must stay below the DA-derived footprint so the latter dominates.
    assert!(gas_used_tx < expected_da_footprint);

    // After Jovian, `blob_gas_used` reports the DA footprint when it exceeds the legacy gas used.
    let (_, result) = executor.finish().expect("failed to finish executor");
    assert_eq!(result.blob_gas_used, expected_da_footprint);
    assert_eq!(result.gas_used, gas_used_tx);
    assert!(result.blob_gas_used > result.gas_used);
}

/// Asserts that `err` is a `TransactionGasLimitMoreThanAvailableBlockGas` with the expected fields.
fn assert_gas_limit_exceeded(
    err: BlockExecutionError,
    expected_tx_gas_limit: u64,
    expected_available: u64,
) {
    match err {
        BlockExecutionError::Validation(
            BlockValidationError::TransactionGasLimitMoreThanAvailableBlockGas {
                transaction_gas_limit,
                block_available_gas,
            },
        ) => {
            assert_eq!(transaction_gas_limit, expected_tx_gas_limit);
            assert_eq!(block_available_gas, expected_available);
        }
        other => panic!("expected TransactionGasLimitMoreThanAvailableBlockGas, got: {other:?}"),
    }
}

// With the stock null refund policy `evm_gas_used` equals `gas_used`, so a tx over the block gas
// limit is rejected with the full block gas limit as available gas.
#[test]
fn test_pre_refund_gas_limit_never_binds_with_sdm_off() {
    const BLOCK_GAS_LIMIT: u64 = 100_000;
    let mut fixture = JovianExecutorFixture::new(
        DEFAULT_DA_FOOTPRINT_GAS_SCALAR,
        BLOCK_GAS_LIMIT,
        JOVIAN_TIMESTAMP,
    );
    let mut executor = fixture.executor();

    let tx = recovered_legacy(TxLegacy { gas_limit: BLOCK_GAS_LIMIT + 1, ..Default::default() });
    let err = executor.execute_transaction(&tx).expect_err("tx over the block gas limit");

    assert_gas_limit_exceeded(err, BLOCK_GAS_LIMIT + 1, BLOCK_GAS_LIMIT);
}

/// A deposit transaction emulating the L1-attributes / network-upgrade deposits that a
/// fork-activation block legitimately contains. Detection is parent-timestamp based, so the
/// calldata contents are irrelevant here.
fn recovered_deposit() -> Recovered<OpTxEnvelope> {
    // A depositor distinct from `Address::ZERO` (the signer of the user legacy txs) so the deposit
    // doesn't bump the user's nonce.
    let deposit = TxDeposit {
        source_hash: B256::ZERO,
        from: Address::with_last_byte(1),
        to: TxKind::Call(L1_BLOCK_CONTRACT),
        mint: 0,
        value: U256::ZERO,
        gas_limit: 50_000,
        is_system_transaction: false,
        input: Bytes::new(),
    };
    Recovered::new_unchecked(
        OpTxEnvelope::Deposit(Sealed::new_unchecked(deposit, B256::ZERO)),
        Address::with_last_byte(1),
    )
}

const KARST_TIMESTAMP: u64 = JOVIAN_TIMESTAMP + 1_000;

/// Builds a chain scheduling every fork at or after Jovian at a distinct, increasing timestamp,
/// returned alongside the `(fork, activation_timestamp)` schedule.
///
/// Driven by [`OpHardfork::forks_from`], so a future hardfork variant is scheduled — and, via the
/// rejection test's loop over the returned schedule, exercised — automatically. `KARST_TIMESTAMP`
/// (`JOVIAN_TIMESTAMP + 1_000`) is the schedule's second entry, used by the single-fork tests.
///
/// `OpChainHardforks` indexes by `OpHardfork::idx()`, so the fork list must hold exactly one entry
/// per fork in canonical order. We keep `op_mainnet()`'s pre-Jovian forks and schedule everything
/// from Jovian onward ourselves.
fn no_user_tx_activation_hardforks() -> (OpChainHardforks, Vec<(OpHardfork, u64)>) {
    let mut forks: Vec<(OpHardfork, ForkCondition)> = OpHardfork::op_mainnet()
        .into_iter()
        .filter(|(fork, _)| fork.idx() < OpHardfork::Jovian.idx())
        .collect();
    let mut schedule = Vec::new();
    for (i, fork) in OpHardfork::Jovian.forks_from().enumerate() {
        let timestamp = JOVIAN_TIMESTAMP + i as u64 * 1_000;
        forks.push((fork, ForkCondition::Timestamp(timestamp)));
        schedule.push((fork, timestamp));
    }
    (OpChainHardforks::new(forks), schedule)
}

#[test]
fn test_no_user_tx_activation_block_rejects_user_tx() {
    // Loops over every fork >= Jovian. Forwards-compatible: adding a hardfork variant schedules
    // and exercises it here automatically, without editing this test.
    let (hardforks, schedule) = no_user_tx_activation_hardforks();
    for (fork, fork_timestamp) in schedule {
        let mut db = prepare_jovian_db(0);
        let receipt_builder = OpAlloyReceiptBuilder::default();
        let mut executor = build_executor(
            &mut db,
            &receipt_builder,
            &hardforks,
            DEFAULT_GAS_LIMIT,
            fork_timestamp,
            Some(fork_timestamp - 1),
            0,
            Address::ZERO,
            Inspect::Enabled,
        );
        assert!(
            executor.ctx.no_user_tx_activation_block,
            "{fork:?} activation block should be flagged"
        );

        let user_tx = recovered_legacy(TxLegacy { gas_limit: 21_000, ..Default::default() });
        let err = executor
            .execute_transaction(&user_tx)
            .expect_err("user tx must be rejected on a fork-activation block");
        match err {
            BlockExecutionError::Validation(BlockValidationError::Other(inner)) => assert!(
                matches!(
                    inner.downcast_ref::<OpBlockExecutionError>(),
                    Some(OpBlockExecutionError::UnexpectedNonDepositTxInForkActivationBlock)
                ),
                "expected UnexpectedNonDepositTxInForkActivationBlock for {fork:?}, got {inner}"
            ),
            other => panic!("expected a validation error for {fork:?}, got {other:?}"),
        }
    }
}

#[test]
fn test_fork_activation_block_accepts_deposits_only() {
    let mut db = prepare_jovian_db(0);
    let receipt_builder = OpAlloyReceiptBuilder::default();
    let (hardforks, _) = no_user_tx_activation_hardforks();
    let mut executor = build_executor(
        &mut db,
        &receipt_builder,
        &hardforks,
        DEFAULT_GAS_LIMIT,
        KARST_TIMESTAMP,
        Some(KARST_TIMESTAMP - 1),
        0,
        Address::ZERO,
        Inspect::Enabled,
    );
    assert!(executor.ctx.no_user_tx_activation_block);

    // Deposits (L1-attributes + network-upgrade automatic deposits) are accepted.
    executor
        .execute_transaction(&recovered_deposit())
        .expect("deposit executes on activation block");

    let (_, result) = executor.finish().expect("activation block finishes");
    // With no user transactions the DA footprint stays at zero.
    assert_eq!(result.blob_gas_used, 0);
}

#[test]
fn test_normal_post_activation_block_accepts_user_tx() {
    // Parent already in Karst -> this is NOT an activation block, so user txs are allowed.
    let mut db = prepare_jovian_db(DEFAULT_DA_FOOTPRINT_GAS_SCALAR);
    let receipt_builder = OpAlloyReceiptBuilder::default();
    let (hardforks, _) = no_user_tx_activation_hardforks();
    let mut executor = build_executor(
        &mut db,
        &receipt_builder,
        &hardforks,
        DEFAULT_GAS_LIMIT,
        KARST_TIMESTAMP + 2,
        Some(KARST_TIMESTAMP + 1),
        0,
        Address::ZERO,
        Inspect::Enabled,
    );
    assert!(!executor.ctx.no_user_tx_activation_block);

    let user_tx = recovered_legacy(TxLegacy { gas_limit: DEFAULT_GAS_LIMIT, ..Default::default() });
    executor.execute_transaction(&user_tx).expect("user tx accepted on a normal Karst block");
}

#[test]
fn test_non_activation_karst_block_not_rejected() {
    // False-trigger guard: a Karst block whose parent is also in Karst is NOT an activation block.
    let mut db = prepare_jovian_db(DEFAULT_DA_FOOTPRINT_GAS_SCALAR);
    let receipt_builder = OpAlloyReceiptBuilder::default();
    let (hardforks, _) = no_user_tx_activation_hardforks();
    let mut executor = build_executor(
        &mut db,
        &receipt_builder,
        &hardforks,
        DEFAULT_GAS_LIMIT,
        KARST_TIMESTAMP + 100,
        Some(KARST_TIMESTAMP + 50),
        0,
        Address::ZERO,
        Inspect::Enabled,
    );
    assert!(!executor.ctx.no_user_tx_activation_block);

    let user_tx = recovered_legacy(TxLegacy { gas_limit: DEFAULT_GAS_LIMIT, ..Default::default() });
    executor
        .execute_transaction(&user_tx)
        .expect("user tx accepted on a non-activation Karst block");
}

#[test]
fn test_none_parent_timestamp_skips_check() {
    // With no parent timestamp (op-reth import path), the guard is skipped even though the
    // block/parent would otherwise make this the Karst activation block.
    let mut db = prepare_jovian_db(0);
    let receipt_builder = OpAlloyReceiptBuilder::default();
    let (hardforks, _) = no_user_tx_activation_hardforks();
    let mut executor = build_executor(
        &mut db,
        &receipt_builder,
        &hardforks,
        DEFAULT_GAS_LIMIT,
        KARST_TIMESTAMP,
        None,
        0,
        Address::ZERO,
        Inspect::Enabled,
    );
    assert!(!executor.ctx.no_user_tx_activation_block);

    let user_tx = recovered_legacy(TxLegacy { gas_limit: DEFAULT_GAS_LIMIT, ..Default::default() });
    executor
        .execute_transaction(&user_tx)
        .expect("check skipped when the parent timestamp is unavailable");
}

const OBSERVER_TEST_CONTRACT: Address = Address::with_last_byte(0x42);

/// Bytecode that executes CREATE, CALL, and SELFDESTRUCT. Every opcode also drives `step`.
const OBSERVER_TEST_BYTECODE: &[u8] = &[
    0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0xf0, 0x50, // CREATE; POP
    0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00, // CALL output/input offsets
    0x60, 0x00, 0x60, 0x00, 0x5a, 0xf1, 0x50, // value, target, GAS, CALL, POP
    0x60, 0x00, 0xff, // SELFDESTRUCT to the zero address
];

fn prepare_observer_db() -> State<InMemoryDB> {
    let mut db = prepare_jovian_db(0);
    let code = Bytes::from_static(OBSERVER_TEST_BYTECODE);
    db.insert_account(
        OBSERVER_TEST_CONTRACT,
        AccountInfo {
            code_hash: keccak256(&code),
            code: Some(Bytecode::new_raw(code)),
            ..Default::default()
        },
    );
    db
}

type PolicyTestExecutor<'a, R> = OpBlockExecutor<
    OpEvm<
        &'a mut State<InMemoryDB>,
        NoOpInspector,
        op_revm::precompiles::OpPrecompiles,
        crate::OpTx,
        R,
    >,
    &'a OpAlloyReceiptBuilder,
    &'a OpChainHardforks,
>;

fn build_policy_executor<'a, R>(
    db: &'a mut State<InMemoryDB>,
    receipt_builder: &'a OpAlloyReceiptBuilder,
    hardforks: &'a OpChainHardforks,
) -> PolicyTestExecutor<'a, R>
where
    R: Default + PostExecRefundInspector,
{
    build_policy_executor_with(db, receipt_builder, hardforks, 0, Address::ZERO, Inspect::Enabled)
}

/// [`build_policy_executor`] with an explicit base fee, beneficiary, and inspector setting, for
/// tests whose assertion depends on fees actually moving ETH or on how the EVM is constructed.
fn build_policy_executor_with<'a, R>(
    db: &'a mut State<InMemoryDB>,
    receipt_builder: &'a OpAlloyReceiptBuilder,
    hardforks: &'a OpChainHardforks,
    base_fee: u64,
    beneficiary: Address,
    inspect: Inspect,
) -> PolicyTestExecutor<'a, R>
where
    R: Default + PostExecRefundInspector,
{
    let ctx = Context::mainnet()
        .with_tx(crate::OpTx(OpTransaction::builder().build_fill()))
        .with_cfg(CfgEnv::new_with_spec(OpSpecId::BEDROCK))
        .with_chain(L1BlockInfo::default())
        .with_db(db)
        .with_block(BlockEnv {
            timestamp: U256::from(JOVIAN_TIMESTAMP),
            gas_limit: 500_000,
            basefee: base_fee,
            beneficiary,
            ..Default::default()
        })
        .modify_cfg_chained(|cfg| cfg.spec = OpSpecId::JOVIAN);

    let evm: OpEvm<_, _, _, crate::OpTx, R> = OpEvm::new(
        ctx.build_op_with_inspector(NoOpInspector {}),
        matches!(inspect, Inspect::Enabled),
    );
    OpBlockExecutor::new(
        evm,
        OpBlockExecutionCtx { post_exec_mode: PostExecMode::Produce, ..Default::default() },
        hardforks,
        receipt_builder,
    )
}

fn observer_test_tx() -> Recovered<OpTxEnvelope> {
    recovered_legacy(TxLegacy {
        to: TxKind::Call(OBSERVER_TEST_CONTRACT),
        gas_limit: 200_000,
        ..Default::default()
    })
}

#[derive(Debug, Clone, Default)]
struct HookObservingPolicy {
    observed_hooks: u64,
}

impl PostExecRefundInspector for HookObservingPolicy {
    type Snapshot = u64;

    fn begin_tx(&mut self, _ctx: PostExecTxContext) {
        self.observed_hooks = 0;
    }

    fn note_account_touch(&mut self, _address: Address) {}

    fn finish_tx(&mut self) -> PostExecExecutedTx {
        PostExecExecutedTx { refund_total: self.observed_hooks, refund_events: Vec::new() }
    }

    fn inspect_step<CTX>(&mut self, _interp: &mut Interpreter, _context: &mut CTX)
    where
        CTX: ContextTr<Journal: JournalExt>,
    {
        self.observed_hooks |= 1;
    }

    fn inspect_call<CTX>(&mut self, _context: &mut CTX, _inputs: &mut CallInputs)
    where
        CTX: ContextTr<Journal: JournalExt>,
    {
        self.observed_hooks |= 2;
    }

    fn inspect_call_end<CTX>(
        &mut self,
        _context: &mut CTX,
        _inputs: &CallInputs,
        _outcome: &CallOutcome,
    ) where
        CTX: ContextTr<Journal: JournalExt>,
    {
        self.observed_hooks |= 16;
    }

    fn inspect_create<CTX>(&mut self, _context: &mut CTX, _inputs: &mut CreateInputs)
    where
        CTX: ContextTr<Journal: JournalExt>,
    {
        self.observed_hooks |= 4;
    }

    fn inspect_create_end<CTX>(
        &mut self,
        _context: &mut CTX,
        _inputs: &CreateInputs,
        _outcome: &CreateOutcome,
    ) where
        CTX: ContextTr<Journal: JournalExt>,
    {
        self.observed_hooks |= 32;
    }

    fn inspect_selfdestruct(&mut self, _contract: Address, _target: Address, _value: U256) {
        self.observed_hooks |= 8;
    }

    fn snapshot(&self) -> Self::Snapshot {
        self.observed_hooks
    }

    fn restore(&mut self, snapshot: Self::Snapshot) {
        self.observed_hooks = snapshot;
    }
}

#[test]
fn post_exec_composite_dispatches_every_observer_hook() {
    let mut db = prepare_observer_db();
    let receipt_builder = OpAlloyReceiptBuilder::default();
    let hardforks = OpChainHardforks::op_mainnet();
    let mut executor =
        build_policy_executor::<HookObservingPolicy>(&mut db, &receipt_builder, &hardforks);

    executor.execute_transaction(&observer_test_tx()).expect("observer workload executes");

    assert_eq!(
        executor.post_exec_entries(),
        &[op_alloy::consensus::SDMGasEntry { index: 0, gas_refund: 63 }],
        "step, call, call_end, create, create_end, and selfdestruct must all reach the refund policy"
    );
}

#[derive(Debug, Clone, Default)]
struct ScriptedRefundPolicy {
    executed_candidates: u64,
}

impl PostExecRefundInspector for ScriptedRefundPolicy {
    type Snapshot = u64;

    fn begin_tx(&mut self, _ctx: PostExecTxContext) {}

    fn note_account_touch(&mut self, _address: Address) {}

    fn finish_tx(&mut self) -> PostExecExecutedTx {
        self.executed_candidates += 1;
        PostExecExecutedTx { refund_total: self.executed_candidates, refund_events: Vec::new() }
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
        self.executed_candidates
    }

    fn restore(&mut self, snapshot: Self::Snapshot) {
        self.executed_candidates = snapshot;
    }
}

#[test]
fn declined_candidate_restores_refund_policy_snapshot() {
    let mut db = prepare_observer_db();
    let receipt_builder = OpAlloyReceiptBuilder::default();
    let hardforks = OpChainHardforks::op_mainnet();
    let mut executor =
        build_policy_executor::<ScriptedRefundPolicy>(&mut db, &receipt_builder, &hardforks);
    let tx = observer_test_tx();

    let declined = executor
        .execute_transaction_with_commit_condition(&tx, |_| CommitChanges::No)
        .expect("declined candidate executes");
    assert!(declined.is_none());

    executor.execute_transaction(&tx).expect("committed candidate executes");
    assert_eq!(
        executor.post_exec_entries(),
        &[op_alloy::consensus::SDMGasEntry { index: 0, gas_refund: 1 }],
        "a declined candidate must not leak policy state into the committed transaction"
    );
}

mod structural_tests;
