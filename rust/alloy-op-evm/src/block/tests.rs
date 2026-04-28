use alloc::{string::ToString, vec};
use alloy_consensus::{SignableTransaction, TxLegacy, transaction::Recovered};
use alloy_eips::eip2718::WithEncoded;
use alloy_evm::{EvmEnv, ToTxEnv};
use alloy_hardforks::ForkCondition;
use alloy_op_hardforks::OpHardfork;
use alloy_primitives::{Address, Signature, U256, uint};
use op_alloy::consensus::OpTxEnvelope;
use op_revm::{
    L1BlockInfo, OpBuilder, OpSpecId, OpTransaction,
    constants::{
        BASE_FEE_SCALAR_OFFSET, ECOTONE_L1_BLOB_BASE_FEE_SLOT, ECOTONE_L1_FEE_SCALARS_SLOT,
        L1_BASE_FEE_SLOT, L1_BLOCK_CONTRACT, OPERATOR_FEE_SCALARS_SLOT,
    },
};
use revm::{
    Context, MainContext,
    context::{BlockEnv, CfgEnv},
    database::{CacheDB, EmptyDB, InMemoryDB, State},
    inspector::NoOpInspector,
    primitives::HashMap,
    state::AccountInfo,
};

use crate::OpEvm;

use super::*;

/// Wraps a `TxLegacy` in an `OpTxEnvelope::Legacy` recovered with a zero signer.
fn recovered_legacy(tx: TxLegacy) -> Recovered<OpTxEnvelope> {
    Recovered::new_unchecked(
        OpTxEnvelope::Legacy(tx.into_signed(Signature::new(
            Default::default(),
            Default::default(),
            Default::default(),
        ))),
        Address::ZERO,
    )
}

/// Build the standard verifier payload (version 1) used by every test.
fn post_exec_payload(block_number: u64, gas_refund_entries: Vec<SDMGasEntry>) -> PostExecPayload {
    PostExecPayload { version: 1, block_number, gas_refund_entries }
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

fn build_executor<'a>(
    db: &'a mut State<InMemoryDB>,
    receipt_builder: &'a OpAlloyReceiptBuilder,
    op_chain_hardforks: &'a OpChainHardforks,
    gas_limit: u64,
    jovian_timestamp: u64,
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
            timestamp: U256::from(jovian_timestamp),
            gas_limit,
            ..Default::default()
        })
        .modify_cfg_chained(|cfg| cfg.spec = OpSpecId::JOVIAN);

    let evm = OpEvm::new(ctx.build_op_with_inspector(NoOpInspector {}), true);

    OpBlockExecutor::new(evm, OpBlockExecutionCtx::default(), op_chain_hardforks, receipt_builder)
}

struct SDMExecutorFixture {
    db: State<InMemoryDB>,
    receipt_builder: OpAlloyReceiptBuilder,
    op_chain_hardforks: OpChainHardforks,
    gas_limit: u64,
    jovian_timestamp: u64,
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
        }
    }

    fn executor(&mut self) -> SDMTestExecutor<'_> {
        build_executor(
            &mut self.db,
            &self.receipt_builder,
            &self.op_chain_hardforks,
            self.gas_limit,
            self.jovian_timestamp,
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

mod sdm {
    use super::*;
    use alloy_consensus::Sealable;
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
        recovered_legacy(TxLegacy {
            nonce,
            gas_limit: 50_000,
            to: alloy_primitives::TxKind::Call(to),
            ..Default::default()
        })
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
        assert_eq!(account.original_info.balance, U256::from(15));

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

    // End-to-end executor coverage for SDM: a producer emits refund entries and appends a
    // post-exec tx, then a verifier replays the same tx stream and consumes the payload.
    #[test]
    fn test_post_exec_producer_verifier_roundtrip() {
        let target = Address::from([0x11; 20]);
        let user_txs = vec![legacy_tx(0, target), legacy_tx(1, target)];

        let mut producer_fixture = SDMExecutorFixture::default();
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

        let entries = producer.take_post_exec_entries();
        assert!(!entries.is_empty(), "producer should emit at least one SDM refund entry");
        assert_eq!(entries[0].index, 1, "the second tx reuses block-warmed addresses");
        assert!(entries[0].gas_refund > 0);

        let post_exec_recovered = recovered_post_exec(0, entries.clone());
        assert_eq!(producer.execute_transaction(&post_exec_recovered).unwrap().tx_gas_used(), 0);
        let (_, produced) = producer.finish().expect("producer finishes block");

        let mut verifier_fixture = SDMExecutorFixture::default();
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
}
