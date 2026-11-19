use super::*;
use alloy_eips::{eip2935::HISTORY_STORAGE_ADDRESS, eip4788::BEACON_ROOTS_ADDRESS};
use alloy_evm::block::{BlockExecutor, BlockExecutorFactory};
use alloy_hardforks::ForkCondition;
use alloy_op_hardforks::{OpChainHardforks, OpHardfork};
use revm::{
    Database as RevmDatabase,
    context::{Block, JournalTr},
    context_interface::Transaction,
    database::State,
    interpreter::{InstructionResult, InterpreterAction, interpreter_types::LoopControl},
    state::Bytecode,
};

#[derive(Debug, PartialEq, Eq)]
struct ObservedCall {
    caller: Address,
    target: Address,
    data: Bytes,
    number: U256,
}

#[derive(Debug, Default)]
struct SystemInspector {
    calls: Vec<ObservedCall>,
    ends: usize,
    steps: usize,
    stop: bool,
    stopped: bool,
}

impl<CTX: ContextTr<Journal: JournalExt>> Inspector<CTX> for SystemInspector {
    fn call(&mut self, context: &mut CTX, inputs: &mut CallInputs) -> Option<CallOutcome> {
        self.calls.push(ObservedCall {
            caller: inputs.caller,
            target: inputs.target_address,
            data: context.tx().input().clone(),
            number: context.block().number(),
        });
        None
    }

    fn call_end(&mut self, _: &mut CTX, _: &CallInputs, _: &mut CallOutcome) {
        self.ends += 1;
    }

    fn step(&mut self, interp: &mut Interpreter, _: &mut CTX) {
        self.steps += 1;
        if self.stop || self.stopped {
            self.stopped = true;
            interp
                .bytecode
                .set_action(InterpreterAction::new_halt(InstructionResult::OutOfGas, interp.gas));
        }
    }
}

fn system_env() -> EvmEnv<OpSpecId> {
    EvmEnv::new(
        CfgEnv::new_with_spec(OpSpecId::JOVIAN).with_chain_id(901),
        BlockEnv {
            number: U256::from(7),
            timestamp: U256::from(1_746_806_402u64),
            gas_limit: 1_000_000,
            ..Default::default()
        },
    )
}

fn insert_code(db: &mut InMemoryDB, target: Address, code: &[u8]) {
    let code = Bytes::copy_from_slice(code);
    db.insert_account_info(
        target,
        AccountInfo {
            code_hash: alloy_primitives::keccak256(&code),
            code: Some(Bytecode::new_raw(code)),
            ..Default::default()
        },
    );
}

#[test_case::test_case(false; "inspection disabled")]
#[test_case::test_case(true; "inspection enabled")]
fn system_calls_preserve_state_and_context_with_runtime_inspection(inspect: bool) {
    let caller = Address::with_last_byte(0xcc);
    let target = Address::with_last_byte(0x44);
    let data = B256::repeat_byte(0x12);
    let mut db = InMemoryDB::default();
    // Store CALLER, CALLDATALOAD(0) and NUMBER in slots 0, 1 and 2.
    insert_code(
        &mut db,
        target,
        &[0x33, 0x60, 0, 0x55, 0x60, 0, 0x35, 0x60, 1, 0x55, 0x43, 0x60, 2, 0x55, 0],
    );
    let factory = OpEvmFactory::<OpTx>::default();
    let mut ordinary = factory.create_evm(db.clone(), system_env());
    let expected = ordinary.transact_system_call(caller, target, data.into()).unwrap();
    let mut evm = factory.create_evm_with_inspector(db, system_env(), SystemInspector::default());
    evm.set_inspector_enabled(inspect);
    let result = evm.transact_system_call(caller, target, data.into()).unwrap();
    assert_eq!(result, expected, "observing must preserve the complete result and state");
    let storage = &result.state[&target].storage;
    assert_eq!(storage[&U256::ZERO].present_value(), U256::from_be_slice(caller.as_slice()));
    assert_eq!(storage[&U256::from(1)].present_value(), U256::from_be_bytes(data.0));
    assert_eq!(storage[&U256::from(2)].present_value(), U256::from(7));
    assert_eq!(evm.cfg_env().chain_id, 901);
    assert_eq!(evm.block(), ordinary.block());
    let inspector = evm.components().1;
    if inspect {
        assert_eq!(
            inspector.calls,
            vec![ObservedCall { caller, target, data: data.into(), number: U256::from(7) }]
        );
        assert_eq!(inspector.ends, 1);
        assert!(inspector.steps > 0, "real system opcodes must reach the inspector");
    } else {
        assert!(inspector.calls.is_empty());
        assert_eq!(inspector.ends, 0);
        assert_eq!(inspector.steps, 0);
    }
}

#[test]
fn system_call_stop_is_latched_before_storage_writes() {
    let target = Address::with_last_byte(0x44);
    let mut db = InMemoryDB::default();
    insert_code(&mut db, target, &[0x60, 1, 0x60, 0, 0x55, 0]);
    let mut evm = OpEvmFactory::<OpTx>::default().create_evm_with_inspector(
        db,
        system_env(),
        SystemInspector { stop: true, ..Default::default() },
    );
    for _ in 0..2 {
        let result = evm.transact_system_call(Address::ZERO, target, Bytes::new()).unwrap();
        assert!(result.result.is_halt());
        assert!(
            result
                .state
                .values()
                .all(|account| account.storage.values().all(|slot| !slot.is_changed()))
        );
        assert!(evm.components().1.stopped);
        // Clearing the external trigger cannot renew the same retained inspector.
        evm.components_mut().1.stop = false;
    }
    assert_eq!(evm.components().1.steps, 2);
    assert_eq!(evm.components().1.ends, 2);
}

#[derive(Debug, thiserror::Error)]
#[error("synthetic storage failure")]
struct StorageFailure;

impl DBErrorMarker for StorageFailure {}

#[derive(Debug)]
struct FailingStorage {
    inner: InMemoryDB,
    fail: bool,
}

impl RevmDatabase for FailingStorage {
    type Error = StorageFailure;

    fn basic(&mut self, address: Address) -> Result<Option<AccountInfo>, Self::Error> {
        Ok(self.inner.basic(address).expect("infallible fixture"))
    }

    fn code_by_hash(&mut self, hash: B256) -> Result<Bytecode, Self::Error> {
        Ok(self.inner.code_by_hash(hash).expect("infallible fixture"))
    }

    fn storage(&mut self, address: Address, index: U256) -> Result<U256, Self::Error> {
        if self.fail && index == U256::from(2) {
            return Err(StorageFailure);
        }
        Ok(self.inner.storage(address, index).expect("infallible fixture"))
    }

    fn block_hash(&mut self, number: u64) -> Result<B256, Self::Error> {
        Ok(self.inner.block_hash(number).expect("infallible fixture"))
    }
}

#[test]
fn inspected_system_error_finalizes_partial_journal_and_retains_inspector() {
    let target = Address::with_last_byte(0x44);
    let mut db = InMemoryDB::default();
    // SSTORE(0, 1), then the failing SLOAD(2).
    insert_code(&mut db, target, &[0x60, 1, 0x60, 0, 0x55, 0x60, 2, 0x54, 0]);
    let factory = OpEvmFactory::<OpTx>::default();
    let mut expected = factory.create_evm(db.clone(), system_env());
    let expected = expected.transact_system_call(Address::ZERO, target, Bytes::new()).unwrap();
    let mut evm = factory.create_evm_with_inspector(
        FailingStorage { inner: db, fail: true },
        system_env(),
        SystemInspector::default(),
    );
    assert!(matches!(
        evm.transact_system_call(Address::ZERO, target, Bytes::new()),
        Err(EVMError::Database(StorageFailure))
    ));
    assert!(evm.inner.0.ctx.journaled_state.evm_state().is_empty());
    assert_eq!(evm.components().1.calls.len(), 1);
    assert!(evm.components().1.steps > 0, "the failure followed a real partial write");
    assert_eq!(evm.db_mut().inner.storage(target, U256::ZERO).unwrap(), U256::ZERO);
    evm.db_mut().fail = false;
    let result = evm.transact_system_call(Address::ZERO, target, Bytes::new()).unwrap();
    assert_eq!(result, expected, "no failed-call journal may contaminate the next call");
    assert_eq!(evm.components().1.calls.len(), 2, "the original inspector is retained");
}

#[test]
fn system_calls_preserve_live_refund_policy_across_ordinary_and_post_exec_transactions() {
    let caller = Address::ZERO;
    let target = Address::with_last_byte(0x44);
    let failing = Address::with_last_byte(0x45);
    let mut db = InMemoryDB::default();
    db.insert_account_info(
        caller,
        AccountInfo { balance: U256::from(1_000_000_000u64), ..Default::default() },
    );
    insert_code(&mut db, failing, &[0x60, 2, 0x54, 0]);
    let mut evm = OpEvmFactory::<OpTx, TestRefundPolicyFactory>::new(TestRefundPolicyFactory {
        initial_committed: 3,
        refund_total: 11,
    })
    .create_evm_with_inspector(
        FailingStorage { inner: db, fail: true },
        system_env(),
        SystemInspector::default(),
    );
    let checkpoint = evm.refund_checkpoint();
    for (index, committed) in [(0, 4), (1, 5)] {
        let before_system = evm.refund_policy().committed;
        evm.transact_system_call(caller, target, Bytes::new()).unwrap();
        assert_eq!(evm.refund_policy().committed, before_system);
        assert_eq!(evm.refund_policy().refund_total, 11);
        evm.begin_post_exec_tx(post_exec::PostExecTxContext {
            tx_index: index,
            kind: post_exec::PostExecTxKind::Normal,
        });
        evm.transact_raw(legacy_op_tx(0, caller, target, 100_000)).unwrap();
        assert_eq!(evm.take_last_post_exec_tx_result().refund_total, 11);
        assert_eq!(evm.refund_policy().committed, committed);
        if index == 0 {
            assert!(matches!(
                evm.transact_system_call(caller, failing, Bytes::new()),
                Err(EVMError::Database(StorageFailure))
            ));
            assert_eq!(evm.refund_policy().committed, committed);
            assert_eq!(evm.refund_policy().refund_total, 11);
            assert!(evm.refund_policy().current_kind.is_none());
            evm.db_mut().fail = false;
        }
    }
    let result = evm.transact_raw(post_exec_op_tx()).unwrap();
    assert!(result.result.is_success());
    assert_eq!(result.result.tx_gas_used(), 0);
    assert!(result.state.is_empty());
    assert_eq!(evm.refund_policy().committed, 5);
    assert_eq!(
        evm.components().1.calls.len(),
        5,
        "system, failed and ordinary calls share one live inspector"
    );
    evm.revert_refund_checkpoint(checkpoint).unwrap();
    assert_eq!(evm.refund_policy().committed, 3, "the original policy checkpoint stays valid");
}

#[test_case::test_case(false, false; "uninspected pre-execution")]
#[test_case::test_case(true, false; "inspected pre-execution")]
#[test_case::test_case(true, true; "guard-only pre-execution")]
fn op_block_pre_execution_propagates_the_selected_system_inspector(inspect: bool, stop: bool) {
    use crate::block::{OpAlloyReceiptBuilder, OpBlockExecutionCtx, OpBlockExecutorFactory};
    let mut db = InMemoryDB::default();
    for target in [HISTORY_STORAGE_ADDRESS, BEACON_ROOTS_ADDRESS] {
        // Store the real SystemCaller's supplied parent hash/root in slot 0.
        insert_code(&mut db, target, &[0x60, 0, 0x35, 0x60, 0, 0x55, 0]);
    }
    let spec = OpChainHardforks::new(
        OpHardfork::op_mainnet()
            .into_iter()
            .chain([(OpHardfork::Jovian, ForkCondition::Timestamp(1_746_806_402))]),
    );
    let factory = OpBlockExecutorFactory::new(
        OpAlloyReceiptBuilder::default(),
        spec,
        OpEvmFactory::<OpTx>::default(),
    );
    let mut state = State::builder().with_database(db).build();
    let mut evm = factory.evm_factory().create_evm_with_inspector(
        &mut state,
        system_env(),
        SystemInspector { stop, ..Default::default() },
    );
    evm.set_inspector_enabled(inspect);
    let parent = B256::repeat_byte(0x12);
    let beacon = B256::repeat_byte(0x34);
    let mut executor = factory.create_executor(
        evm,
        OpBlockExecutionCtx {
            parent_hash: parent,
            parent_beacon_block_root: Some(beacon),
            ..Default::default()
        },
    );
    executor.apply_pre_execution_changes().expect("actual OP system pre-execution");
    let inspector = executor.evm().components().1;
    if inspect {
        assert_eq!(
            inspector.calls.iter().map(|call| call.target).collect::<Vec<_>>(),
            vec![HISTORY_STORAGE_ADDRESS, BEACON_ROOTS_ADDRESS]
        );
        assert_eq!(
            inspector.calls.iter().map(|call| call.data.clone()).collect::<Vec<_>>(),
            vec![Bytes::from(parent), Bytes::from(beacon)]
        );
        assert_eq!(inspector.ends, 2);
        assert!(inspector.steps > 0);
        assert_eq!(inspector.stopped, stop);
    } else {
        assert!(inspector.calls.is_empty());
        assert_eq!(inspector.steps, 0);
    }
    drop(executor);
    for (target, value) in [(HISTORY_STORAGE_ADDRESS, parent), (BEACON_ROOTS_ADDRESS, beacon)] {
        assert_eq!(
            state.storage(target, U256::ZERO).unwrap(),
            if stop { U256::ZERO } else { U256::from_be_bytes(value.0) }
        );
    }
}
