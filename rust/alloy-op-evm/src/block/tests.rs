use alloc::{string::ToString, vec};
use alloy_consensus::{Sealed, SignableTransaction, TxLegacy, transaction::Recovered};
use alloy_eips::eip2718::WithEncoded;
use alloy_evm::{EvmEnv, ToTxEnv};
use alloy_hardforks::ForkCondition;
use alloy_op_hardforks::{OpHardfork, OpHardforks};
use alloy_primitives::{Address, B256, Bytes, Signature, TxKind, U256, uint};
use op_alloy::consensus::{OpTxEnvelope, TxDeposit};
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
    database::{CacheDB, EmptyDB, InMemoryDB, State},
    inspector::NoOpInspector,
    primitives::HashMap,
    state::{AccountInfo, Bytecode},
};

use crate::OpEvm;

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

/// Build the standard verifier payload (version 1) used by every test.
fn post_exec_payload(block_number: u64, gas_refund_entries: Vec<SDMGasEntry>) -> PostExecPayload {
    PostExecPayload { version: 1, block_number, gas_refund_entries }
}

/// Runtime of a contract that reads (warms) its own storage slot 0: `PUSH1 0x00; SLOAD; POP; STOP`.
///
/// Two transactions that *call* this contract warm the same storage slot across the block, so the
/// second tx earns a genuine, non-intrinsic SLOAD warming rebate — the kind of cross-tx warming SDM
/// exists to rebate, as opposed to the intrinsic sender/`to`/fee-vault touches a plain value
/// transfer makes (which are correctly never rebated).
const WARMING_CONTRACT_CODE: [u8; 5] = [0x60, 0x00, 0x54, 0x50, 0x00];

const WARMING_CONTRACT: Address = Address::new([0x77; 20]);

fn warming_contract_account() -> AccountInfo {
    let code = Bytecode::new_raw(Bytes::from_static(&WARMING_CONTRACT_CODE));
    AccountInfo {
        code_hash: alloy_primitives::keccak256(WARMING_CONTRACT_CODE),
        code: Some(code),
        ..Default::default()
    }
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

type SDMTestExecutor<'a> = OpBlockExecutor<
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
) -> SDMTestExecutor<'a> {
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

    let evm = OpEvm::new(ctx.build_op_with_inspector(NoOpInspector {}), true);

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

struct SDMExecutorFixture {
    db: State<InMemoryDB>,
    receipt_builder: OpAlloyReceiptBuilder,
    op_chain_hardforks: OpChainHardforks,
    gas_limit: u64,
    jovian_timestamp: u64,
    parent_timestamp: Option<u64>,
    base_fee: u64,
    beneficiary: Address,
}

impl SDMExecutorFixture {
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
            // SDM/post-exec tests run normal (non-activation) blocks; leaving the parent timestamp
            // unset skips the fork-activation guard, matching op-reth's parentless import path.
            parent_timestamp: None,
            // Default to a zero base fee; settlement tests opt into a non-zero one.
            base_fee: 0,
            // Default beneficiary is the zero address; coinbase-warmth tests opt into a distinct
            // one so the block beneficiary is separable from the (also-zero) default tx sender.
            beneficiary: Address::ZERO,
        }
    }

    fn executor(&mut self) -> SDMTestExecutor<'_> {
        build_executor(
            &mut self.db,
            &self.receipt_builder,
            &self.op_chain_hardforks,
            self.gas_limit,
            self.jovian_timestamp,
            self.parent_timestamp,
            self.base_fee,
            self.beneficiary,
        )
    }

    fn executor_with_post_exec_mode(
        &mut self,
        post_exec_mode: PostExecMode,
    ) -> SDMTestExecutor<'_> {
        let mut executor = self.executor();
        executor.set_post_exec_mode(post_exec_mode);
        executor
    }

    /// Shorthand for an executor in `Verify` mode against `post_exec_payload(block, entries)`.
    fn verifier(&mut self, block_number: u64, entries: Vec<SDMGasEntry>) -> SDMTestExecutor<'_> {
        self.executor_with_post_exec_mode(PostExecMode::Verify(post_exec_payload(
            block_number,
            entries,
        )))
    }
}

impl Default for SDMExecutorFixture {
    fn default() -> Self {
        Self::new(DEFAULT_DA_FOOTPRINT_GAS_SCALAR, DEFAULT_GAS_LIMIT, JOVIAN_TIMESTAMP)
    }
}

#[test]
fn test_jovian_da_footprint_estimation() {
    let mut fixture = SDMExecutorFixture::default();
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
        SDMExecutorFixture::new(DEFAULT_DA_FOOTPRINT_GAS_SCALAR, GAS_LIMIT, JOVIAN_TIMESTAMP);
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

    let mut fixture = SDMExecutorFixture::new(DA_FOOTPRINT_GAS_SCALAR, GAS_LIMIT, JOVIAN_TIMESTAMP);
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

// SDM-off regression: with refunds disabled `evm_gas_used` equals `gas_used`, so a tx over the
// block gas limit is rejected with the full block gas limit as available gas.
#[test]
fn test_pre_refund_gas_limit_never_binds_with_sdm_off() {
    const BLOCK_GAS_LIMIT: u64 = 100_000;
    let mut fixture =
        SDMExecutorFixture::new(DEFAULT_DA_FOOTPRINT_GAS_SCALAR, BLOCK_GAS_LIMIT, JOVIAN_TIMESTAMP);
    let mut executor = fixture.executor();

    let tx = recovered_legacy(TxLegacy { gas_limit: BLOCK_GAS_LIMIT + 1, ..Default::default() });
    let err = executor.execute_transaction(&tx).expect_err("tx over the block gas limit");

    assert_gas_limit_exceeded(err, BLOCK_GAS_LIMIT + 1, BLOCK_GAS_LIMIT);
}

// SDM refunds lower canonical gas but must not increase the real compute admitted into a block:
// after a refund, a tx that fits canonical `gas_used` while exceeding the pre-refund `evm_gas_used`
// budget must still be rejected.
#[test]
fn test_pre_refund_gas_limit_counts_sdm_refunded_gas() {
    const BLOCK_GAS_LIMIT: u64 = 100_000;
    // Both txs call a contract that SLOADs a shared slot, so tx1 earns a genuine cross-tx warming
    // rebate and canonical gas falls below pre-refund EVM gas.
    let target = WARMING_CONTRACT;
    let tx0 = recovered_legacy(TxLegacy {
        nonce: 0,
        gas_limit: 50_000,
        to: alloy_primitives::TxKind::Call(target),
        ..Default::default()
    });
    let tx1 = recovered_legacy(TxLegacy {
        nonce: 1,
        gas_limit: 50_000,
        to: alloy_primitives::TxKind::Call(target),
        ..Default::default()
    });

    let mut fixture =
        SDMExecutorFixture::new(DEFAULT_DA_FOOTPRINT_GAS_SCALAR, BLOCK_GAS_LIMIT, JOVIAN_TIMESTAMP);
    fixture.db.insert_account(target, warming_contract_account());
    let mut executor = fixture.executor_with_post_exec_mode(PostExecMode::Produce);
    executor.execute_transaction(&tx0).expect("first tx fits");
    executor.execute_transaction(&tx1).expect("second tx fits and receives a refund");

    assert!(executor.evm_gas_used > executor.gas_used, "expected SDM to refund canonical gas");
    let evm_gas_available = BLOCK_GAS_LIMIT - executor.evm_gas_used;
    let canonical_gas_available = BLOCK_GAS_LIMIT - executor.gas_used;
    assert!(evm_gas_available < canonical_gas_available);

    let tx2_gas_limit = evm_gas_available + 1;
    assert!(
        tx2_gas_limit <= canonical_gas_available,
        "test tx should fit canonical gas but exceed pre-refund gas"
    );
    let tx2 = recovered_legacy(TxLegacy {
        nonce: 2,
        gas_limit: tx2_gas_limit,
        to: alloy_primitives::TxKind::Call(target),
        ..Default::default()
    });

    let err = executor
        .execute_transaction(&tx2)
        .expect_err("tx exceeding pre-refund block gas must be rejected");
    assert_gas_limit_exceeded(err, tx2_gas_limit, evm_gas_available);
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
    );
    assert!(!executor.ctx.no_user_tx_activation_block);

    let user_tx = recovered_legacy(TxLegacy { gas_limit: DEFAULT_GAS_LIMIT, ..Default::default() });
    executor
        .execute_transaction(&user_tx)
        .expect("check skipped when the parent timestamp is unavailable");
}

mod sdm {
    use super::*;
    use alloy_consensus::{Sealable, TxEip7702};
    use alloy_eips::eip7702::{Authorization, SignedAuthorization};
    use op_alloy::consensus::build_post_exec_tx;

    /// Builds a recovered post-exec (0x7D) tx with a zero signer.
    fn recovered_post_exec(
        block_number: u64,
        entries: Vec<SDMGasEntry>,
    ) -> Recovered<OpTxEnvelope> {
        Recovered::new_unchecked(
            OpTxEnvelope::PostExec(build_post_exec_tx(block_number, entries).seal_slow()),
            Address::ZERO,
        )
    }

    fn legacy_tx(nonce: u64, to: Address) -> Recovered<OpTxEnvelope> {
        legacy_tx_with_gas(nonce, to, 50_000)
    }

    fn legacy_tx_with_gas(nonce: u64, to: Address, gas_limit: u64) -> Recovered<OpTxEnvelope> {
        recovered_legacy(TxLegacy {
            nonce,
            gas_limit,
            to: alloy_primitives::TxKind::Call(to),
            ..Default::default()
        })
    }

    /// A legacy tx with an explicit gas price, so post-exec settlement deltas (which scale with
    /// `effective_gas_price - basefee`) are non-trivial.
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
            to: alloy_primitives::TxKind::Call(to),
            ..Default::default()
        })
    }

    fn legacy_tx_from(sender: Address, nonce: u64, to: Address) -> Recovered<OpTxEnvelope> {
        recovered_legacy_from(
            sender,
            TxLegacy {
                nonce,
                gas_limit: 50_000,
                to: alloy_primitives::TxKind::Call(to),
                ..Default::default()
            },
        )
    }

    /// A top-level CREATE tx with empty init code (deploys an empty contract).
    fn create_tx(nonce: u64) -> Recovered<OpTxEnvelope> {
        recovered_legacy(TxLegacy {
            nonce,
            gas_limit: 100_000,
            to: alloy_primitives::TxKind::Create,
            ..Default::default()
        })
    }

    fn full_refund_for_second_tx(
        block_gas_limit: u64,
        tx0: &Recovered<OpTxEnvelope>,
        tx1: &Recovered<OpTxEnvelope>,
    ) -> Vec<SDMGasEntry> {
        let mut fixture = SDMExecutorFixture::new(
            DEFAULT_DA_FOOTPRINT_GAS_SCALAR,
            block_gas_limit,
            JOVIAN_TIMESTAMP,
        );
        let mut probe = fixture.executor();
        probe.execute_transaction(tx0).expect("probe first tx");
        let tx1_evm_gas_used =
            probe.execute_transaction(tx1).expect("probe second tx").tx_gas_used();

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
        let bundle_account = bundle
            .account(&BASE_FEE_RECIPIENT)
            .expect("bundle must contain the base fee recipient");
        assert_eq!(bundle_account.original_info.as_ref().unwrap().balance, U256::from(10));
        assert_eq!(bundle_account.info.as_ref().unwrap().balance, U256::from(12));
    }

    // "Where the rebate money comes from": settling a refund credits the sender exactly the sum of
    // three recipient debits (beneficiary priority fee, base-fee vault, operator-fee vault), so
    // total ETH supply is conserved — no value is minted. Pins each per-recipient share against the
    // spec formula and the conservation identity.
    #[test]
    fn test_post_exec_settlement_deltas_conserve_value() {
        const BASE_FEE: u64 = 7;
        const GAS_PRICE: u128 = 100;
        const EVM_GAS_USED: u64 = 50_000;
        const REFUND: u64 = 1_000;

        let mut fixture = SDMExecutorFixture { base_fee: BASE_FEE, ..Default::default() };
        let mut executor = fixture.executor();

        let tx = legacy_tx_with_price(0, Address::from([0x11; 20]), DEFAULT_GAS_LIMIT, GAS_PRICE);
        let deltas = executor
            .post_exec_settlement_deltas(&tx, EVM_GAS_USED, REFUND, false, false)
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

    // Settlement only moves money for refunding standard txs. Deposits, the post-exec tx itself,
    // and zero-refund txs produce no balance deltas, regardless of gas price / basefee.
    #[test]
    fn test_post_exec_settlement_deltas_skip_non_refunding_txs() {
        let mut fixture = SDMExecutorFixture { base_fee: 7, ..Default::default() };
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
                executor.post_exec_settlement_deltas(&tx, 50_000, 1_000, true, false).unwrap()
            ),
            "deposits never settle a refund",
        );
        // The post-exec (0x7D) tx itself never claims.
        assert!(
            is_no_op(
                executor.post_exec_settlement_deltas(&tx, 50_000, 1_000, false, true).unwrap()
            ),
            "the post-exec tx never settles a refund",
        );
        // Zero refund: nothing to settle.
        assert!(
            is_no_op(executor.post_exec_settlement_deltas(&tx, 50_000, 0, false, false).unwrap()),
            "a zero refund produces no settlement",
        );
    }

    // A settlement debit larger than a recipient's balance invalidates the block via
    // PostExecSettlementUnderflow rather than silently saturating — this is the guard against a
    // malformed/adversarial payload minting ETH out of an underfunded vault.
    #[test]
    fn test_post_exec_settlement_underflow_is_rejected() {
        let mut fixture = SDMExecutorFixture::default();
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

    // A transaction's own `to` is intrinsically warm under EIP-2929 (pre-added to the accessed set,
    // billed warm, never cold), so it must never earn a warming rebate even when an earlier tx
    // warmed it. Regression test: `collect_intrinsic_warmth` now marks `caller` and the Call
    // `target` warm, so a later tx no longer claims a rebate for its own `to`/sender.
    #[test]
    fn test_intrinsic_warm_to_address_is_not_rebated() {
        let target = Address::from([0x11; 20]);
        // Two txs to the same `to`. The first warms `target`; the second touches it only as its own
        // (intrinsically warm) `to`, so it must not claim a rebate for it.
        let user_txs = vec![legacy_tx(0, target), legacy_tx(1, target)];

        let mut fixture = SDMExecutorFixture::default();
        let mut producer = fixture.executor_with_post_exec_mode(PostExecMode::Produce);
        for tx in &user_txs {
            producer.execute_transaction(tx).expect("producer executes user tx");
        }

        // No tx may claim a warming rebate for `target`: it is the `to` of every tx, hence
        // intrinsically warm for each. (Fee-recipient touches, a separate concern, target other
        // addresses and are unaffected by this assertion.)
        let claimed_own_to =
            producer.warming_events_by_tx.iter().flatten().find(|event| event.address == target);
        assert!(
            claimed_own_to.is_none(),
            "a tx claimed a warming rebate for its own intrinsically-warm `to` ({target}): {:#?}",
            producer.warming_events_by_tx,
        );
    }

    // Companion to the `to` case: a tx's own `sender` is also intrinsically warm under EIP-2929, so
    // a later tx from the same sender must not claim a warming rebate for it. Uses a non-zero
    // sender distinct from the block beneficiary (separately excluded) and distinct recipients,
    // so the shared sender is the only warmed account under test.
    #[test]
    fn test_intrinsic_warm_sender_is_not_rebated() {
        let sender = Address::from([0x55; 20]);
        let txs = vec![
            legacy_tx_from(sender, 0, Address::from([0xa1; 20])),
            legacy_tx_from(sender, 1, Address::from([0xa2; 20])),
        ];

        let mut fixture = SDMExecutorFixture::default();
        // Fund the sender so it can cover the operator fee charged on each tx.
        fixture.db.insert_account(
            sender,
            AccountInfo { balance: U256::from(400_000_000), ..Default::default() },
        );
        let mut producer = fixture.executor_with_post_exec_mode(PostExecMode::Produce);
        for tx in &txs {
            producer.execute_transaction(tx).expect("producer executes user tx");
        }

        let claimed_own_sender =
            producer.warming_events_by_tx.iter().flatten().find(|event| event.address == sender);
        assert!(
            claimed_own_sender.is_none(),
            "a tx claimed a warming rebate for its own intrinsically-warm sender ({sender}): {:#?}",
            producer.warming_events_by_tx,
        );
    }

    // A top-level CREATE's created-contract address is the tx's intrinsic "to" under EIP-2929 —
    // pre-warmed, billed warm. Regression test: a CREATE tx whose created address an earlier tx
    // warmed must not claim a rebate for it.
    #[test]
    fn test_intrinsic_warm_created_address_is_not_rebated() {
        // A block gas limit large enough to fit the dummy tx plus the CREATE.
        const BLOCK_GAS_LIMIT: u64 = 1_000_000;

        // Probe the deterministic created address by running the same (dummy tx, CREATE) sequence.
        let created = {
            let mut probe_fixture = SDMExecutorFixture::new(
                DEFAULT_DA_FOOTPRINT_GAS_SCALAR,
                BLOCK_GAS_LIMIT,
                JOVIAN_TIMESTAMP,
            );
            let mut probe = probe_fixture.executor_with_post_exec_mode(PostExecMode::Produce);
            probe
                .execute_transaction(&legacy_tx(0, Address::from([0x33; 20])))
                .expect("probe dummy tx");
            let mut created = None;
            probe
                .execute_transaction_with_result_closure(&create_tx(1), |res| {
                    if let ExecutionResult::Success {
                        output: Output::Create(_, Some(addr)), ..
                    } = &res.result().result
                    {
                        created = Some(*addr);
                    }
                })
                .expect("probe create tx");
            created.expect("create produced a contract address")
        };

        // Real run: tx0 warms `created` (as its `to`); tx1 creates at `created`.
        let mut fixture = SDMExecutorFixture::new(
            DEFAULT_DA_FOOTPRINT_GAS_SCALAR,
            BLOCK_GAS_LIMIT,
            JOVIAN_TIMESTAMP,
        );
        let mut producer = fixture.executor_with_post_exec_mode(PostExecMode::Produce);
        producer.execute_transaction(&legacy_tx(0, created)).expect("tx0 warms the create address");
        producer.execute_transaction(&create_tx(1)).expect("tx1 creates at the warmed address");

        let claimed_created =
            producer.warming_events_by_tx.iter().flatten().find(|event| event.address == created);
        assert!(
            claimed_created.is_none(),
            "a CREATE tx claimed a warming rebate for its own created address ({created}): {:#?}",
            producer.warming_events_by_tx,
        );
    }

    // End-to-end companion to the inspector-level settlement test: the OP fee vaults (L1/base-fee/
    // operator-fee recipients) are warmed by the protocol's per-tx fee settlement write in
    // `transact_raw`, not by a user opcode access, so no cold EIP-2929 access is ever paid for
    // them. Regression test: a second non-deposit tx must not claim a warming rebate for them
    // just because the first tx's settlement warmed them.
    #[test]
    fn test_fee_recipient_settlement_touch_is_not_rebated() {
        // Two plain transfers to distinct fresh recipients: the only accounts tx1 re-touches that
        // tx0 warmed are the fee vaults, via each tx's settlement write.
        let txs =
            vec![legacy_tx(0, Address::from([0xc1; 20])), legacy_tx(1, Address::from([0xc2; 20]))];

        let mut fixture = SDMExecutorFixture::default();
        let mut producer = fixture.executor_with_post_exec_mode(PostExecMode::Produce);
        for tx in &txs {
            producer.execute_transaction(tx).expect("producer executes user tx");
        }

        let fee_recipients = [L1_FEE_RECIPIENT, BASE_FEE_RECIPIENT, OPERATOR_FEE_RECIPIENT];
        let claimed: Vec<Address> = producer
            .warming_events_by_tx
            .iter()
            .flatten()
            .map(|event| event.address)
            .filter(|address| fee_recipients.contains(address))
            .collect();
        assert!(
            claimed.is_empty(),
            "fee recipients were rebated for a settlement-only touch (no cold access paid): {claimed:?}",
        );
    }

    // End-to-end executor coverage for SDM: a producer emits refund entries and appends a
    // post-exec tx, then a verifier replays the same tx stream and consumes the payload.
    #[test]
    fn test_post_exec_producer_verifier_roundtrip() {
        // Both txs call a contract that SLOADs a shared slot, so tx1 earns a genuine cross-tx
        // (non-intrinsic) warming rebate.
        let target = WARMING_CONTRACT;
        let user_txs = vec![legacy_tx(0, target), legacy_tx(1, target)];

        let mut producer_fixture = SDMExecutorFixture::default();
        producer_fixture.db.insert_account(target, warming_contract_account());
        let mut producer = producer_fixture.executor_with_post_exec_mode(PostExecMode::Produce);
        let first_user_gas = producer
            .execute_transaction(&user_txs[0])
            .expect("producer executes first user tx")
            .tx_gas_used();
        let second_user_gas = producer
            .execute_transaction(&user_txs[1])
            .expect("producer executes second user tx")
            .tx_gas_used();
        assert!(second_user_gas < first_user_gas, "second user tx should receive an SDM refund");

        let snapshot = producer.post_exec_entries().to_vec();
        assert!(!snapshot.is_empty(), "snapshot should expose produced SDM entries");
        assert_eq!(producer.post_exec_entries(), snapshot.as_slice(), "snapshot must not drain");

        let entries = producer.take_post_exec_entries();
        assert_eq!(entries, snapshot, "take should return the same entries observed by snapshot");
        assert!(producer.post_exec_entries().is_empty(), "take should drain produced entries");
        assert!(!entries.is_empty(), "producer should emit at least one SDM refund entry");
        assert_eq!(entries[0].index, 1, "the second tx reuses block-warmed addresses");
        assert!(entries[0].gas_refund > 0);

        let post_exec_recovered = recovered_post_exec(0, entries.clone());
        assert_eq!(producer.execute_transaction(&post_exec_recovered).unwrap().tx_gas_used(), 0);
        let (_, produced) = producer.finish().expect("producer finishes block");

        let mut verifier_fixture = SDMExecutorFixture::default();
        verifier_fixture.db.insert_account(target, warming_contract_account());
        let mut verifier = verifier_fixture.verifier(0, entries);
        for tx in &user_txs {
            verifier.execute_transaction(tx).expect("verifier executes user tx");
        }
        assert_eq!(verifier.execute_transaction(&post_exec_recovered).unwrap().tx_gas_used(), 0);
        let (_, verified) = verifier.finish().expect("verifier consumes all entries");

        assert_eq!(verified.gas_used, produced.gas_used);
        assert_eq!(verified.blob_gas_used, produced.blob_gas_used);
        assert_eq!(verified.receipts, produced.receipts);
        assert_eq!(verified.receipts.len(), user_txs.len() + 1);
    }

    // Demonstrates the accounting the pre-refund cap relies on: under SDM refunds, canonical
    // `gas_used` falls below `evm_gas_used`, so capping on `evm_gas_used` (not `gas_used`) keeps
    // tracking the real compute performed.
    #[test]
    fn test_evm_gas_used_tracks_pre_refund_gas_under_sdm() {
        let target = WARMING_CONTRACT;
        let user_txs = vec![legacy_tx(0, target), legacy_tx(1, target)];

        let mut fixture = SDMExecutorFixture::default();
        fixture.db.insert_account(target, warming_contract_account());
        let mut producer = fixture.executor_with_post_exec_mode(PostExecMode::Produce);
        for tx in &user_txs {
            producer.execute_transaction(tx).expect("producer executes user tx");
        }

        // The second tx reuses block-warmed addresses and earns an SDM refund, so canonical gas is
        // strictly less than the pre-refund EVM gas spent.
        assert!(!producer.post_exec_entries().is_empty(), "expected an SDM refund to be produced");
        assert!(
            producer.evm_gas_used > producer.gas_used,
            "pre-refund evm_gas_used ({}) must exceed canonical gas_used ({}) once refunds apply",
            producer.evm_gas_used,
            producer.gas_used,
        );
        // The gap is exactly the total refund.
        let total_refund: u64 = producer.post_exec_entries().iter().map(|e| e.gas_refund).sum();
        assert_eq!(producer.evm_gas_used - producer.gas_used, total_refund);
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

        let mut fixture = SDMExecutorFixture::new(
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

        let mut fixture = SDMExecutorFixture::new(
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
        let mut fixture = SDMExecutorFixture::default();
        let mut executor = fixture.verifier(42, vec![]);

        let err =
            executor.apply_pre_execution_changes().expect_err("mismatched block number must fail");
        assert_invalid_post_exec(err, "payload block number 42 does not match block number 0");
    }

    #[test]
    fn test_duplicate_payload_index_fails_pre_execution() {
        // Two entries colliding on tx index 3 — the second insert must be flagged at construction
        // and surface as a pre-execution failure.
        let mut fixture = SDMExecutorFixture::default();
        let mut executor = fixture.verifier(
            0,
            vec![
                SDMGasEntry { index: 3, gas_refund: 10 },
                SDMGasEntry { index: 3, gas_refund: 20 },
            ],
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
            let mut fixture = SDMExecutorFixture::default();
            let executor =
                fixture.verifier(0, vec![SDMGasEntry { index: tx_index, gas_refund: 1 }]);

            let err = executor
                .verifier_post_exec_refund_for_tx(tx_index, is_deposit, is_post_exec, evm_gas_used)
                .expect_err("payload entries must not target non-normal txs");
            assert_invalid_post_exec(err, expected_reason);
        }
    }

    #[test]
    fn test_verifier_rejects_refund_exceeding_evm_gas() {
        let mut fixture = SDMExecutorFixture::default();
        let executor = fixture.verifier(0, vec![SDMGasEntry { index: 2, gas_refund: 50_000 }]);

        // evm_gas_used < payload refund — a refund that exceeds the tx's EVM-reported cost is
        // impossible under SDM semantics and must be rejected, otherwise canonical_gas_used
        // would underflow to a bogus value via saturating_sub.
        let err = executor
            .verifier_post_exec_refund_for_tx(2, false, false, 40_000)
            .expect_err("refund greater than evm_gas_used must be rejected");
        assert_invalid_post_exec(
            err,
            "payload refund 50000 exceeds evm_gas_used 40000 for tx index 2",
        );

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
            let mut fixture = SDMExecutorFixture::default();
            let executor = fixture.verifier(0, vec![SDMGasEntry { index: 7, gas_refund: 42 }]);

            let refund = executor
                .verifier_post_exec_refund_for_tx(tx_index, is_deposit, is_post_exec, 21_000)
                .unwrap_or_else(|err| panic!("{label}: expected no refund, got error: {err:?}"));
            assert_eq!(refund, 0, "{label}");
        }
    }

    #[test]
    fn test_finish_reports_all_unconsumed_post_exec_entries() {
        let mut fixture = SDMExecutorFixture::default();
        let executor = fixture.verifier(
            0,
            vec![SDMGasEntry { index: 2, gas_refund: 7 }, SDMGasEntry { index: 5, gas_refund: 11 }],
        );

        let Err(err) = executor.finish() else {
            panic!("unconsumed verifier entries must fail");
        };
        assert_invalid_post_exec(
            err,
            "2 unconsumed post-exec payload entries for tx indexes [2, 5]",
        );
    }

    /// Followers running with SDM disabled must reject any block that carries a post-exec
    /// 0x7D tx. Silently short-circuiting the tx (which is what the pre-guard code did) would
    /// let a producer ship a payload with arbitrary refund entries that no follower validates,
    /// and the two nodes' states would diverge without anyone noticing.
    #[test]
    fn test_disabled_mode_rejects_post_exec_tx() {
        let mut fixture = SDMExecutorFixture::default();
        // build_executor leaves post_exec_mode at the default (Disabled).
        let mut executor = fixture.executor();
        assert!(matches!(executor.post_exec, PostExecState::Disabled));

        let tx = recovered_post_exec(0, vec![]);
        let err =
            executor.execute_transaction(&tx).expect_err("0x7D tx in Disabled mode must fail");
        assert_invalid_post_exec(
            err,
            "unexpected post-exec tx at index 0: SDM not active for this block",
        );
    }

    // A candidate that is executed but declined (`CommitChanges::No`) must not leave behind
    // block-scoped warming: without rollback, an uncommitted candidate would grant a later
    // committed tx a phantom refund, diverging the producer's payload from derivation
    // (ethereum-optimism/optimism#21354).
    #[test]
    fn test_declined_candidate_does_not_warm_later_committed_tx() {
        // Route warming through a probe contract that BALANCE-touches a *third* account: a tx's own
        // `to` (here `probe`) is intrinsically warm and never rebated, so we observe warming on
        // `warmed`, which each call reaches via a genuine cold EIP-2929 access.
        let probe = Address::from([0x11; 20]);
        let warmed = Address::from([0x22; 20]);

        let mut fixture = SDMExecutorFixture::default();
        fixture.db.insert_account(probe, balance_probe_account(&[warmed]));
        let mut producer = fixture.executor_with_post_exec_mode(PostExecMode::Produce);

        // Execute a candidate that warms `warmed` but decline to commit it, as the payload builder
        // does when a candidate exceeds a limit or is reverted-and-excluded.
        let outcome = producer
            .execute_transaction_with_commit_condition(&legacy_tx(0, probe), |_| CommitChanges::No)
            .expect("declined candidate still executes");
        assert!(outcome.is_none(), "candidate must not be committed");
        assert!(
            producer.post_exec_entries().is_empty(),
            "a declined candidate emits no SDM refund entries",
        );

        // The first committed tx touching `warmed` must be its first toucher: the declined
        // candidate's warming was rolled back, so no refund.
        producer.execute_transaction(&legacy_tx(0, probe)).expect("first committed tx executes");
        assert!(
            producer.post_exec_entries().is_empty(),
            "an uncommitted candidate must not warm a later committed tx (no phantom SDM refund)",
        );

        // Sanity: committed warmth still accumulates — a second committed tx re-warms the
        // now-committed address and earns a refund.
        producer.execute_transaction(&legacy_tx(1, probe)).expect("second committed tx executes");
        let entries = producer.post_exec_entries();
        assert_eq!(entries.len(), 1, "second committed tx re-warms the block-warmed address");
        assert_eq!(entries[0].index, 1, "refund is attributed to the second committed tx");
        assert!(entries[0].gas_refund > 0);
    }

    /// Bytecode that runs `BALANCE` against each address in turn, warming it through a genuine
    /// cold EIP-2929 access: `PUSH20 <addr>; BALANCE; POP` per address, then `STOP`. Unlike a
    /// plain transfer (which only touches the intrinsically-warm sender/`to`), calling this
    /// contract lets a test warm arbitrary accounts the way a real opcode access would.
    fn balance_probe_account(addrs: &[Address]) -> AccountInfo {
        let mut code = Vec::new();
        for addr in addrs {
            code.push(0x73); // PUSH20
            code.extend_from_slice(addr.as_slice());
            code.push(0x31); // BALANCE
            code.push(0x50); // POP
        }
        code.push(0x00); // STOP
        let raw = Bytes::from(code);
        let code_hash = alloy_primitives::keccak256(&raw);
        AccountInfo { code_hash, code: Some(Bytecode::new_raw(raw)), ..Default::default() }
    }

    /// Builds a recovered EIP-7702 (set-code) tx signed with the zero signer, carrying a single
    /// authorization. Used to exercise the authority intrinsic-warm path.
    fn recovered_7702(
        nonce: u64,
        to: Address,
        auth: SignedAuthorization,
    ) -> Recovered<OpTxEnvelope> {
        let tx = TxEip7702 {
            chain_id: 1,
            nonce,
            gas_limit: 200_000,
            max_fee_per_gas: 0,
            max_priority_fee_per_gas: 0,
            to,
            value: U256::ZERO,
            access_list: Default::default(),
            authorization_list: vec![auth],
            input: Bytes::new(),
        };
        Recovered::new_unchecked(
            OpTxEnvelope::Eip7702(tx.into_signed(Signature::new(
                Default::default(),
                Default::default(),
                false,
            ))),
            Address::ZERO,
        )
    }

    /// Sums the ETH balance of each address (absent accounts count as zero).
    fn sum_balances(db: &mut State<InMemoryDB>, addrs: &[Address]) -> U256 {
        addrs.iter().fold(U256::ZERO, |acc, addr| {
            let balance = revm::Database::basic(db, *addr)
                .expect("load account")
                .map(|info| info.balance)
                .unwrap_or_default();
            acc + balance
        })
    }

    /// Returns every warming rebate event the producer attributed, flattened across txs.
    fn all_warming_events(producer: &SDMTestExecutor<'_>) -> Vec<WarmingRefundEvent> {
        producer.warming_events_by_tx.iter().flatten().copied().collect()
    }

    // The block beneficiary (coinbase) is in every transaction's EIP-2929 intrinsically-warm set,
    // billed warm and never cold, so it must never earn a warming rebate — even after an earlier
    // tx warmed it. Regression test for the `collect_intrinsic_warmth` beneficiary insert: a
    // control account warmed the same way (a genuine cold BALANCE) still earns its rebate, proving
    // the assertion isn't vacuous.
    #[test]
    fn test_intrinsic_warm_beneficiary_is_not_rebated() {
        const BLOCK_GAS_LIMIT: u64 = 1_000_000;
        let beneficiary = Address::from([0x99; 20]);
        let control = Address::from([0xcc; 20]);
        let probe = Address::from([0x88; 20]);

        let mut fixture = SDMExecutorFixture::new(
            DEFAULT_DA_FOOTPRINT_GAS_SCALAR,
            BLOCK_GAS_LIMIT,
            JOVIAN_TIMESTAMP,
        );
        fixture.beneficiary = beneficiary;
        fixture.db.insert_account(probe, balance_probe_account(&[beneficiary, control]));

        let mut producer = fixture.executor_with_post_exec_mode(PostExecMode::Produce);
        producer
            .execute_transaction(&legacy_tx(0, probe))
            .expect("tx0 warms beneficiary + control");
        producer.execute_transaction(&legacy_tx(1, probe)).expect("tx1 re-touches both");

        let events = all_warming_events(&producer);
        assert!(
            !events.iter().any(|e| e.address == beneficiary),
            "the block beneficiary was rebated despite being intrinsically warm: {events:#?}",
        );
        assert!(
            events.iter().any(|e| e.address == control && e.amount == 2_500),
            "a control account warmed across txs must still be rebated: {events:#?}",
        );
    }

    // A precompile address is added to every transaction's intrinsically-warm set (it is in the
    // journal's precompile set), billed warm and never cold, so it must never earn a warming
    // rebate. Regression test for the `collect_intrinsic_warmth` precompile-set extension.
    #[test]
    fn test_intrinsic_warm_precompile_is_not_rebated() {
        const BLOCK_GAS_LIMIT: u64 = 1_000_000;
        let precompile = Address::with_last_byte(1); // ecrecover — always in the precompile set.
        let control = Address::from([0xcd; 20]);
        let probe = Address::from([0x8a; 20]);

        let mut fixture = SDMExecutorFixture::new(
            DEFAULT_DA_FOOTPRINT_GAS_SCALAR,
            BLOCK_GAS_LIMIT,
            JOVIAN_TIMESTAMP,
        );
        fixture.db.insert_account(probe, balance_probe_account(&[precompile, control]));

        let mut producer = fixture.executor_with_post_exec_mode(PostExecMode::Produce);
        producer.execute_transaction(&legacy_tx(0, probe)).expect("tx0 warms precompile + control");
        producer.execute_transaction(&legacy_tx(1, probe)).expect("tx1 re-touches both");

        let events = all_warming_events(&producer);
        assert!(
            !events.iter().any(|e| e.address == precompile),
            "a precompile was rebated despite being intrinsically warm: {events:#?}",
        );
        assert!(
            events.iter().any(|e| e.address == control && e.amount == 2_500),
            "a control account warmed across txs must still be rebated: {events:#?}",
        );
    }

    // An EIP-7702 authorization authority is pre-warmed for the transaction that lists it (added to
    // the accessed set at tx start), so that tx must not earn a warming rebate for it — even though
    // an earlier tx warmed the same account through a genuine cold access. Regression test for the
    // `collect_intrinsic_warmth` authorization-list loop.
    #[test]
    fn test_intrinsic_warm_7702_authority_is_not_rebated() {
        const BLOCK_GAS_LIMIT: u64 = 2_000_000;
        let delegate = Address::from([0xde; 20]);
        let auth = Authorization { chain_id: U256::ZERO, address: delegate, nonce: 0 }
            .into_signed(Signature::test_signature());
        let authority = auth.recover_authority().expect("authority recovers");
        let control = Address::from([0xce; 20]);
        let probe = Address::from([0x8b; 20]);
        // Guard against an accidental address collision making the assertion vacuous.
        assert!(![Address::ZERO, delegate, control, probe].contains(&authority));

        let mut fixture = SDMExecutorFixture::new(
            DEFAULT_DA_FOOTPRINT_GAS_SCALAR,
            BLOCK_GAS_LIMIT,
            JOVIAN_TIMESTAMP,
        );
        fixture.db.insert_account(probe, balance_probe_account(&[authority, control]));

        let mut producer = fixture.executor_with_post_exec_mode(PostExecMode::Produce);
        // tx0: a normal tx warms `authority` and `control` via genuine cold BALANCE accesses.
        producer.execute_transaction(&legacy_tx(0, probe)).expect("tx0 warms authority + control");
        // tx1: a 7702 tx listing `authority`, which makes `authority` intrinsically warm for it.
        producer
            .execute_transaction(&recovered_7702(1, probe, auth))
            .expect("tx1 (7702) re-touches both");

        let events = all_warming_events(&producer);
        assert!(
            !events.iter().any(|e| e.address == authority),
            "a 7702 authority was rebated despite being intrinsically warm: {events:#?}",
        );
        assert!(
            events.iter().any(|e| e.address == control && e.amount == 2_500),
            "a control account warmed across txs must still be rebated: {events:#?}",
        );
    }

    /// The set of accounts that can hold ETH in the SDM fixture: the sender, the block
    /// beneficiary, the three OP fee vaults, the `L1Block` predeploy and the called contract.
    fn eth_holding_universe(beneficiary: Address, target: Address) -> [Address; 7] {
        [
            Address::ZERO,
            beneficiary,
            target,
            L1_BLOCK_CONTRACT,
            L1_FEE_RECIPIENT,
            BASE_FEE_RECIPIENT,
            OPERATOR_FEE_RECIPIENT,
        ]
    }

    /// Runs a two-tx warming block in Produce mode and, when `settle` is set, appends the produced
    /// post-exec (0x7D) tx so its SDM settlement is applied. Returns the total ETH balance over
    /// `universe` after the block finishes, plus the produced refund entries.
    fn run_block_and_total(
        beneficiary: Address,
        target: Address,
        settle: bool,
        universe: &[Address],
    ) -> (U256, Vec<SDMGasEntry>) {
        let mut fixture =
            SDMExecutorFixture::new(DEFAULT_DA_FOOTPRINT_GAS_SCALAR, 1_000_000, JOVIAN_TIMESTAMP);
        fixture.beneficiary = beneficiary;
        fixture.base_fee = 7;
        fixture.db.insert_account(target, warming_contract_account());

        // Two txs sharing a warmed SLOAD slot, priced above basefee so the priority-fee share that
        // settlement moves between the sender and the beneficiary is non-trivial.
        let tx0 = legacy_tx_with_price(0, target, 50_000, 100);
        let tx1 = legacy_tx_with_price(1, target, 50_000, 100);
        let entries = {
            let mut producer = fixture.executor_with_post_exec_mode(PostExecMode::Produce);
            producer.execute_transaction(&tx0).expect("tx0 executes");
            producer.execute_transaction(&tx1).expect("tx1 earns a refund");
            let entries = producer.take_post_exec_entries();
            if settle {
                let post_exec = recovered_post_exec(0, entries.clone());
                producer.execute_transaction(&post_exec).expect("post-exec settles the refund");
            }
            // Dropping the EVM releases its borrow of `fixture.db` so the audit can read balances.
            let (evm, _result) = producer.finish().expect("producer finishes the block");
            drop(evm);
            entries
        };
        (sum_balances(&mut fixture.db, universe), entries)
    }

    // State-level ETH-supply audit of post-exec settlement ("audit total balances before and after
    // refund blocks"). The per-delta test pins that one settlement credit equals the sum of its
    // debits; this pins the same conservation on real account state. We run the identical two-tx
    // block twice — once applying the SDM refund settlement, once not — and sum every account that
    // can hold ETH. Settlement only moves ETH between the sender and the fee recipients, so the
    // total must match the un-settled run exactly: no ETH is minted or burned ("no infinite mint").
    // Comparing the two runs cancels block-level effects (e.g. the test harness' coinbase reward)
    // so the assertion isolates the settlement.
    #[test]
    fn test_post_exec_settlement_conserves_total_eth_supply() {
        let beneficiary = Address::from([0x99; 20]);
        let target = WARMING_CONTRACT;
        let universe = eth_holding_universe(beneficiary, target);

        let (total_settled, entries) = run_block_and_total(beneficiary, target, true, &universe);
        let (total_unsettled, _) = run_block_and_total(beneficiary, target, false, &universe);

        assert!(
            entries.iter().any(|e| e.gas_refund > 0),
            "the block under audit must actually settle a non-zero refund",
        );
        assert_eq!(
            total_settled, total_unsettled,
            "SDM post-exec settlement minted or burned ETH relative to the unsettled block",
        );
    }
}
