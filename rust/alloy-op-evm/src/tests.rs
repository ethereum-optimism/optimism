use alloc::vec;
use alloy_consensus::{SignableTransaction, TxLegacy};
use alloy_evm::{
    EvmInternals, FromRecoveredTx,
    evm::EvmFactoryExt,
    precompiles::{Precompile, PrecompileInput},
};
use alloy_primitives::{B256, Signature, TxKind, U256};
use core::convert::Infallible;
use op_revm::{
    OpTransactionError,
    precompiles::{bls12_381, bn254_pair},
};
use revm::{
    context::CfgEnv,
    context_interface::{
        ContextTr,
        result::{EVMError, InvalidTransaction},
    },
    database::{CacheDB, EmptyDB, InMemoryDB},
    inspector::JournalExt,
    interpreter::{CallInputs, CallOutcome, CreateInputs, CreateOutcome, Interpreter},
    precompile::PrecompileHalt,
    primitives::eip7825::TX_GAS_LIMIT_CAP,
    state::AccountInfo,
};
use revm_inspectors::tracing::{TracingInspector, TracingInspectorConfig};

use super::*;

/// Runtime of a contract that reads (warms) storage slot 0: `PUSH1 0x00; SLOAD; POP; STOP`.
#[derive(Debug, Default)]
struct TestRefundPolicy {
    current_kind: Option<post_exec::PostExecTxKind>,
    committed: u64,
}

impl post_exec::PostExecRefundInspector for TestRefundPolicy {
    type Snapshot = u64;

    fn begin_tx(&mut self, ctx: post_exec::PostExecTxContext) {
        self.current_kind = Some(ctx.kind);
    }

    fn note_account_touch(&mut self, _address: Address) {}

    fn finish_tx(&mut self) -> post_exec::PostExecExecutedTx {
        let refund_total = if self.current_kind.take() == Some(post_exec::PostExecTxKind::Normal) {
            self.committed += 1;
            7
        } else {
            0
        };
        post_exec::PostExecExecutedTx { refund_total, refund_events: Vec::new() }
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
        self.committed
    }

    fn restore(&mut self, snapshot: Self::Snapshot) {
        self.committed = snapshot;
    }
}

fn legacy_op_tx(nonce: u64, caller: Address, target: Address, gas_limit: u64) -> OpTx {
    let tx = TxLegacy { nonce, gas_limit, to: TxKind::Call(target), ..Default::default() }
        .into_signed(Signature::new(Default::default(), Default::default(), Default::default()));

    OpTx::from_recovered_tx(&tx, caller)
}

fn post_exec_op_tx() -> OpTx {
    let tx = op_alloy::consensus::post_exec::build_post_exec_tx(1, vec![]);
    OpTx::from_recovered_tx(&tx, Address::ZERO)
}

/// Post-exec transactions only appear in Lagoon-active blocks, so the replay env under test
/// runs with the Lagoon spec.
fn lagoon_env_on_chain_901() -> EvmEnv<OpSpecId> {
    EvmEnv::new(CfgEnv::new_with_spec(OpSpecId::LAGOON).with_chain_id(901), BlockEnv::default())
}

fn lagoon_test_evm_on_chain_901<DB: Database>(db: DB) -> OpEvm<DB, NoOpInspector, PrecompilesMap> {
    OpEvmFactory::<OpTx>::default().create_evm(db, lagoon_env_on_chain_901())
}

#[test_case::test_case(false; "without inspector")]
#[test_case::test_case(true; "with inspector")]
fn transact_raw_short_circuits_post_exec_tx(inspect: bool) {
    let mut evm = lagoon_test_evm_on_chain_901(EmptyDB::default());
    evm.set_inspector_enabled(inspect);

    let result = evm.transact_raw(post_exec_op_tx()).expect("post-exec tx short-circuits");

    assert!(matches!(result.result, revm::context::result::ExecutionResult::Success { .. }));
    assert_eq!(result.result.tx_gas_used(), 0);
    assert!(result.state.is_empty());
}

/// Documents the failure the post-exec short-circuit avoids: `TxEnv::default()` carries
/// `chain_id: Some(1)`, which revm rejects on any non-mainnet chain.
#[test]
fn transact_raw_rejects_mainnet_chain_id_on_other_chain() {
    let mut evm = lagoon_test_evm_on_chain_901(EmptyDB::default());

    let stale_chain_id_tx = OpTx(OpTransaction {
        base: TxEnv { gas_limit: 100_000, ..Default::default() },
        enveloped_tx: Some(Bytes::new()),
        deposit: Default::default(),
    });
    assert_eq!(stale_chain_id_tx.chain_id(), Some(1));

    let err = evm.transact_raw(stale_chain_id_tx).expect_err("chain id mismatch");
    assert!(format!("{err:?}").contains("InvalidChainId"));
}

/// `debug_trace*` replays each transaction through `Evm::transact` with a tracing inspector
/// attached — the exact path that failed with `InvalidChainId` before the post-exec
/// short-circuit. A post-exec tx must yield a successful, empty trace instead of aborting the
/// whole block trace.
#[test]
fn tracing_inspector_traces_post_exec_tx_as_noop() {
    let mut inspector = TracingInspector::new(TracingInspectorConfig::default_geth());
    let mut evm = OpEvmFactory::<OpTx>::default().create_evm_with_inspector(
        EmptyDB::default(),
        lagoon_env_on_chain_901(),
        &mut inspector,
    );

    let result = evm.transact(post_exec_op_tx()).expect("tracing a post-exec tx succeeds");
    assert!(matches!(result.result, revm::context::result::ExecutionResult::Success { .. }));
    assert_eq!(result.result.tx_gas_used(), 0);
    drop(evm);

    let nodes = inspector.traces().nodes();
    assert_eq!(nodes.len(), 1, "only the arena's placeholder root remains");
    assert!(
        nodes[0].children.is_empty() && nodes[0].trace.steps.is_empty(),
        "a no-op replay records no call frames and no steps",
    );
}

/// The `trace_*` namespace iterates a block's transactions through `TxTracer::try_trace_many`,
/// which ends the iteration early — without surfacing an error — when a replay fails. A block
/// carrying a post-exec tx must produce one trace result per transaction, not a silently
/// truncated list.
#[test]
fn try_trace_many_covers_every_tx_in_a_post_exec_block() {
    let mut tracer = OpEvmFactory::<OpTx>::default().create_tracer(
        CacheDB::new(EmptyDB::default()),
        lagoon_env_on_chain_901(),
        TracingInspector::new(TracingInspectorConfig::default_geth()),
    );

    let traced_gas = tracer
        .try_trace_many(
            vec![
                legacy_op_tx(0, Address::with_last_byte(0xEE), Address::with_last_byte(1), 21_000),
                post_exec_op_tx(),
            ],
            |ctx| Ok::<_, EVMError<Infallible, OpTxError>>(ctx.result.tx_gas_used()),
        )
        .collect::<Result<Vec<_>, _>>()
        .expect("both transactions trace");

    assert_eq!(traced_gas, vec![21_000, 0], "the post-exec tx must not truncate the iteration");
}

/// EIP-7825 caps a transaction's gas limit, and a transaction above the cap is rejected before
/// it executes. The deposit exemption in `transact_raw` is scoped to the deposit that carries
/// it and must leave that rule intact for every other transaction.
#[test]
fn non_deposit_above_tx_gas_limit_cap_is_rejected() {
    let caller = Address::ZERO;
    let target = Address::from([0x22; 20]);
    // Karst is Osaka-based, so EIP-7825 is live. The block gas limit is set above the
    // transaction's so that the cap is the only limit the transaction can trip over.
    let mut evm = OpEvmFactory::<OpTx>::default().create_evm(
        EmptyDB::default(),
        EvmEnv::new(
            CfgEnv::new_with_spec(OpSpecId::KARST),
            BlockEnv { gas_limit: 60_000_000, ..Default::default() },
        ),
    );

    let err = evm
        .transact_raw(legacy_op_tx(0, caller, target, TX_GAS_LIMIT_CAP + 1))
        .expect_err("a transaction above the cap must be rejected");
    assert!(
        matches!(
            err,
            EVMError::Transaction(OpTxError(OpTransactionError::Base(
                InvalidTransaction::TxGasLimitGreaterThanCap { .. }
            )))
        ),
        "expected TxGasLimitGreaterThanCap, got {err:?}",
    );
}

/// Runtime that reads 16 distinct cold storage slots (`PUSH1 n; SLOAD; POP` x16, then `STOP`),
/// costing ~33.7k gas — comfortably more than the capped budget in the test below and
/// comfortably less than the uncapped one.
fn cold_sload_burner_runtime() -> Bytes {
    let mut code = Vec::new();
    for slot in 0u8..16 {
        code.extend_from_slice(&[0x60, slot, 0x54, 0x50]);
    }
    code.push(0x00);
    Bytes::from(code)
}

/// Deposits are force-included from L1 and must not be clamped by the EIP-7825 per-transaction
/// gas cap, so `transact_raw` lifts the cap for the duration of a deposit.
///
/// The cap is not enforced by a rejection on this path — deposits skip `validate_env`, which is
/// where `TxGasLimitGreaterThanCap` is raised — so the exemption is only observable in how much
/// gas the first frame actually receives: `initial_gas_and_reservoir` splits the limit at
/// `min(gas_limit, cap)`, and OP does not override the `validate_initial_tx_gas` path that feeds
/// it. This test therefore measures execution, not the error type: the deposit runs a payload
/// that costs more than the capped budget and must still complete.
#[test]
fn deposit_above_tx_gas_limit_cap_receives_the_full_gas_limit() {
    let caller = Address::ZERO;
    let target = Address::from([0x44; 20]);

    // An explicit low cap keeps the burner payload cheap while reproducing the real shape:
    // capped budget = 30_000 - 21_000 intrinsic = 9_000 gas, which the payload exceeds.
    const CAP: u64 = 30_000;
    const DEPOSIT_GAS_LIMIT: u64 = 200_000;

    let mut db = InMemoryDB::default();
    let runtime = cold_sload_burner_runtime();
    db.insert_account_info(
        target,
        AccountInfo {
            code_hash: alloy_primitives::keccak256(&runtime),
            code: Some(revm::bytecode::Bytecode::new_raw(runtime)),
            ..Default::default()
        },
    );

    let mut cfg = CfgEnv::new_with_spec(OpSpecId::KARST);
    cfg.tx_gas_limit_cap = Some(CAP);
    let mut evm = OpEvmFactory::<OpTx>::default()
        .create_evm(db, EvmEnv::new(cfg, BlockEnv { gas_limit: 60_000_000, ..Default::default() }));

    let deposit = OpTx(OpTransaction {
        base: TxEnv {
            gas_limit: DEPOSIT_GAS_LIMIT,
            kind: TxKind::Call(target),
            caller,
            ..Default::default()
        },
        enveloped_tx: None,
        deposit: op_revm::transaction::deposit::DepositTransactionParts::new(
            B256::from([0x11; 32]),
            None,
            false,
        ),
    });

    let result = evm.transact_raw(deposit).expect("a deposit must not be rejected");
    assert!(
        result.result.is_success(),
        "the deposit must receive its full gas limit, not the capped budget; got {:?}",
        result.result,
    );
    assert!(
        result.result.tx_gas_used() > CAP,
        "the payload must actually exceed the cap or this test proves nothing; used {}",
        result.result.tx_gas_used(),
    );

    // The exemption is scoped to the deposit: the previous cap must be back afterwards.
    assert_eq!(evm.inner.0.ctx.cfg.tx_gas_limit_cap, Some(CAP));
}

/// The cap is saved and restored around a deposit as an `Option<Option<u64>>`, so it must
/// round-trip whichever resting state the field is in — including `None`, which is the
/// production shape (the env builder leaves the raw field unset and lets revm derive the
/// effective cap from the spec).
#[test]
fn deposit_cap_exemption_round_trips_every_resting_state() {
    let caller = Address::ZERO;
    let target = Address::from([0x55; 20]);

    for resting in [None, Some(TX_GAS_LIMIT_CAP), Some(u64::MAX)] {
        let mut cfg = CfgEnv::new_with_spec(OpSpecId::KARST);
        cfg.tx_gas_limit_cap = resting;
        let mut evm = OpEvmFactory::<OpTx>::default().create_evm(
            EmptyDB::default(),
            EvmEnv::new(cfg, BlockEnv { gas_limit: 60_000_000, ..Default::default() }),
        );

        let deposit = OpTx(OpTransaction {
            base: TxEnv {
                gas_limit: 100_000,
                kind: TxKind::Call(target),
                caller,
                ..Default::default()
            },
            enveloped_tx: None,
            deposit: op_revm::transaction::deposit::DepositTransactionParts::new(
                B256::from([0x22; 32]),
                None,
                false,
            ),
        });
        evm.transact_raw(deposit).expect("deposit executes");

        assert_eq!(
            evm.inner.0.ctx.cfg.tx_gas_limit_cap, resting,
            "cap must be restored to its resting state {resting:?}",
        );
    }
}

#[test]
fn op_evm_factory_uses_configured_refund_policy_and_snapshot() {
    let caller = Address::ZERO;
    let target = Address::from([0x33; 20]);
    let mut db = InMemoryDB::default();
    db.insert_account_info(
        caller,
        AccountInfo { balance: U256::from(1_000_000_000u64), ..Default::default() },
    );

    let mut evm = OpEvmFactory::<OpTx, TestRefundPolicy>::default().create_evm(
        db,
        EvmEnv::new(
            CfgEnv::new_with_spec(OpSpecId::JOVIAN),
            BlockEnv { gas_limit: 1_000_000, ..Default::default() },
        ),
    );
    evm.begin_post_exec_tx(post_exec::PostExecTxContext {
        tx_index: 0,
        kind: post_exec::PostExecTxKind::Normal,
    });
    evm.transact_raw(legacy_op_tx(0, caller, target, 100_000)).expect("tx executes");
    assert_eq!(evm.take_last_post_exec_tx_result().refund_total, 7);
    assert_eq!(evm.refund_snapshot(), 1);
    evm.seed_refund_snapshot(9);
    assert_eq!(evm.refund_snapshot(), 9);
}

#[test]
fn test_precompiles_jovian_fail() {
    let mut evm = OpEvmFactory::<OpTx>::default().create_evm(
        EmptyDB::default(),
        EvmEnv::new(CfgEnv::new_with_spec(OpSpecId::JOVIAN), BlockEnv::default()),
    );

    let (precompiles, ctx) = (&mut evm.inner.0.precompiles, &mut evm.inner.0.ctx);

    let jovian_precompile = precompiles.get(bn254_pair::JOVIAN.address()).unwrap();
    let result = jovian_precompile
        .call(PrecompileInput {
            data: &vec![0; bn254_pair::JOVIAN_MAX_INPUT_SIZE + 1],
            gas: u64::MAX,
            reservoir: 0,
            caller: Address::ZERO,
            value: U256::ZERO,
            is_static: false,
            target_address: Address::ZERO,
            bytecode_address: Address::ZERO,
            internals: EvmInternals::from_context(ctx),
        })
        .unwrap();

    assert!(result.is_halt());
    assert!(matches!(result.halt_reason(), Some(&PrecompileHalt::Bn254PairLength)));

    let jovian_precompile = precompiles.get(bls12_381::JOVIAN_G1_MSM.address()).unwrap();
    let result = jovian_precompile
        .call(PrecompileInput {
            data: &vec![0; bls12_381::JOVIAN_G1_MSM_MAX_INPUT_SIZE + 1],
            gas: u64::MAX,
            reservoir: 0,
            caller: Address::ZERO,
            value: U256::ZERO,
            is_static: false,
            target_address: Address::ZERO,
            bytecode_address: Address::ZERO,
            internals: EvmInternals::from_context(ctx),
        })
        .unwrap();

    assert!(result.is_halt());
    assert!(matches!(
        result.halt_reason(),
        Some(PrecompileHalt::Other(msg)) if msg.contains("G1MSM input length too long")
    ));

    let jovian_precompile = precompiles.get(bls12_381::JOVIAN_G2_MSM.address()).unwrap();
    let result = jovian_precompile
        .call(PrecompileInput {
            data: &vec![0; bls12_381::JOVIAN_G2_MSM_MAX_INPUT_SIZE + 1],
            gas: u64::MAX,
            reservoir: 0,
            caller: Address::ZERO,
            value: U256::ZERO,
            is_static: false,
            target_address: Address::ZERO,
            bytecode_address: Address::ZERO,
            internals: EvmInternals::from_context(ctx),
        })
        .unwrap();

    assert!(result.is_halt());
    assert!(matches!(
        result.halt_reason(),
        Some(PrecompileHalt::Other(msg)) if msg.contains("G2MSM input length too long")
    ));

    let jovian_precompile = precompiles.get(bls12_381::JOVIAN_PAIRING.address()).unwrap();
    let result = jovian_precompile
        .call(PrecompileInput {
            data: &vec![0; bls12_381::JOVIAN_PAIRING_MAX_INPUT_SIZE + 1],
            gas: u64::MAX,
            reservoir: 0,
            caller: Address::ZERO,
            value: U256::ZERO,
            is_static: false,
            target_address: Address::ZERO,
            bytecode_address: Address::ZERO,
            internals: EvmInternals::from_context(ctx),
        })
        .unwrap();

    assert!(result.is_halt());
    assert!(matches!(
        result.halt_reason(),
        Some(PrecompileHalt::Other(msg)) if msg.contains("Pairing input length too long")
    ));
}

#[test]
fn test_precompiles_jovian() {
    let mut evm = OpEvmFactory::<OpTx>::default().create_evm(
        EmptyDB::default(),
        EvmEnv::new(CfgEnv::new_with_spec(OpSpecId::JOVIAN), BlockEnv::default()),
    );
    let (precompiles, ctx) = (&mut evm.inner.0.precompiles, &mut evm.inner.0.ctx);
    let jovian_precompile = precompiles.get(bn254_pair::JOVIAN.address()).unwrap();
    let result = jovian_precompile.call(PrecompileInput {
        data: &vec![0; bn254_pair::JOVIAN_MAX_INPUT_SIZE],
        gas: u64::MAX,
        reservoir: 0,
        caller: Address::ZERO,
        value: U256::ZERO,
        is_static: false,
        target_address: Address::ZERO,
        bytecode_address: Address::ZERO,
        internals: EvmInternals::from_context(ctx),
    });

    assert!(result.is_ok());

    let jovian_precompile = precompiles.get(bls12_381::JOVIAN_G1_MSM.address()).unwrap();
    let result = jovian_precompile.call(PrecompileInput {
        data: &vec![0; bls12_381::JOVIAN_G1_MSM_MAX_INPUT_SIZE],
        gas: u64::MAX,
        reservoir: 0,
        caller: Address::ZERO,
        value: U256::ZERO,
        is_static: false,
        target_address: Address::ZERO,
        bytecode_address: Address::ZERO,
        internals: EvmInternals::from_context(ctx),
    });

    assert!(result.is_ok());

    let jovian_precompile = precompiles.get(bls12_381::JOVIAN_G2_MSM.address()).unwrap();
    let result = jovian_precompile.call(PrecompileInput {
        data: &vec![0; bls12_381::JOVIAN_G2_MSM_MAX_INPUT_SIZE],
        gas: u64::MAX,
        reservoir: 0,
        caller: Address::ZERO,
        value: U256::ZERO,
        is_static: false,
        target_address: Address::ZERO,
        bytecode_address: Address::ZERO,
        internals: EvmInternals::from_context(ctx),
    });

    assert!(result.is_ok());

    let jovian_precompile = precompiles.get(bls12_381::JOVIAN_PAIRING.address()).unwrap();
    let result = jovian_precompile.call(PrecompileInput {
        data: &vec![0; bls12_381::JOVIAN_PAIRING_MAX_INPUT_SIZE],
        gas: u64::MAX,
        reservoir: 0,
        caller: Address::ZERO,
        value: U256::ZERO,
        is_static: false,
        target_address: Address::ZERO,
        bytecode_address: Address::ZERO,
        internals: EvmInternals::from_context(ctx),
    });

    assert!(result.is_ok());
}
