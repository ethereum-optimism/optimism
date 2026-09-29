use super::*;
use crate::post_exec::{ObservedRefundPolicy, ParallelRefundPolicy, TransactionObserver};
use alloc::{collections::BTreeMap, sync::Arc};
use revm::database::{BundleState, states::bundle_state::BundleRetention};

const CONTRACT: Address = Address::repeat_byte(0x42);
const BENEFICIARY: Address = Address::repeat_byte(0x77);
type ConfigurePrecompiles = fn(&mut alloy_evm::precompiles::PrecompilesMap);

#[derive(Default, Debug, Clone, PartialEq, Eq)]
struct PolicyState {
    touched: BTreeMap<Address, u64>,
    evaluations: Vec<(PostExecTxContext, usize)>,
}

#[derive(Default)]
struct Observation {
    touches: Vec<Address>,
    reverted: usize,
}

#[derive(Default)]
struct Observer(Observation);

impl PostExecRefundInspector for Observer {
    type Snapshot = ();
    fn begin_tx(&mut self, _: PostExecTxContext) {
        self.0 = Observation::default();
    }
    fn note_account_touch(&mut self, address: Address) {
        self.0.touches.push(address);
    }
    fn finish_tx(&mut self) -> PostExecExecutedTx {
        PostExecExecutedTx::default()
    }
    fn inspect_step<CTX: ContextTr<Journal: JournalExt>>(
        &mut self,
        _: &mut Interpreter,
        _: &mut CTX,
    ) {
    }
    fn inspect_call<CTX: ContextTr<Journal: JournalExt>>(
        &mut self,
        context: &mut CTX,
        inputs: &mut CallInputs,
    ) {
        self.0.touches.push(inputs.target_address);
        // Additional observer reads must invalidate a speculative attempt just like EVM reads.
        let _ = context.db_mut().basic(CONTRACT);
        let _ = context.db_mut().storage(CONTRACT, U256::ZERO);
    }
    fn inspect_call_end<CTX: ContextTr<Journal: JournalExt>>(
        &mut self,
        _: &mut CTX,
        _: &CallInputs,
        outcome: &CallOutcome,
    ) {
        self.0.reverted += usize::from(outcome.result.result.is_revert());
    }
    fn inspect_create<CTX: ContextTr<Journal: JournalExt>>(
        &mut self,
        _: &mut CTX,
        _: &mut CreateInputs,
    ) {
    }
    fn inspect_create_end<CTX: ContextTr<Journal: JournalExt>>(
        &mut self,
        _: &mut CTX,
        _: &CreateInputs,
        _: &CreateOutcome,
    ) {
    }
    fn inspect_selfdestruct(&mut self, contract: Address, target: Address, _: U256) {
        self.0.touches.extend([contract, target]);
    }
    fn snapshot(&self) {}
    fn restore(&mut self, _: ()) {}
}

impl TransactionObserver for Observer {
    type Observation = Observation;
    fn take_observation(&mut self) -> (Observation, usize) {
        let bytes = self.0.touches.capacity() * core::mem::size_of::<Address>() +
            core::mem::size_of::<Observation>();
        (core::mem::take(&mut self.0), bytes)
    }
}

std::thread_local! {
    // Synthetic policy work is opt-in in the ignored benchmark, on the coordinator only.
    static POLICY_WORK: core::cell::Cell<usize> = const { core::cell::Cell::new(0) };
}

struct Policy;
impl ParallelRefundPolicy for Policy {
    type Observer = Observer;
    type State = PolicyState;
    fn evaluate(
        state: &PolicyState,
        context: PostExecTxContext,
        observation: &Observation,
    ) -> (PostExecExecutedTx, PolicyState) {
        POLICY_WORK.with(|work| {
            let mut digest = B256::ZERO;
            for _ in 0..work.get() {
                digest = alloy_primitives::keccak256(core::hint::black_box(digest));
            }
            core::hint::black_box(digest);
        });
        let mut next = state.clone();
        let refund = observation
            .touches
            .iter()
            .filter(|address| state.touched.contains_key(*address))
            .count() as u64 *
            100;
        for address in &observation.touches {
            next.touched.entry(*address).or_insert(context.tx_index);
        }
        next.evaluations.push((context, observation.reverted));
        (
            PostExecExecutedTx {
                refund_total: if context.kind.claims_refunds() { refund } else { 0 },
                refund_events: Vec::new(),
            },
            next,
        )
    }
}

#[derive(Debug, PartialEq, Eq)]
struct Reference {
    bundle: BundleState,
    state_root: B256,
    receipt_root: B256,
    receipts: Vec<op_alloy::consensus::OpReceiptEnvelope>,
    outputs: Vec<Result<ExecutionResult<op_revm::OpHaltReason>, String>>,
    pre_refund_gas: u64,
    canonical_gas: u64,
    da_gas: u64,
    entries: Vec<SDMGasEntry>,
    policy: PolicyState,
}

fn transfer(sender: u8, nonce: u64, to: Address) -> Recovered<OpTxEnvelope> {
    recovered_legacy_from(
        Address::repeat_byte(sender),
        TxLegacy {
            nonce,
            gas_limit: 150_000,
            gas_price: 10,
            to: TxKind::Call(to),
            value: U256::from(sender),
            ..Default::default()
        },
    )
}

fn database(code: &[u8]) -> State<InMemoryDB> {
    let mut db = prepare_jovian_db(1);
    db.transition_state = Some(Default::default());
    for sender in 1..=8 {
        db.insert_account(
            Address::repeat_byte(sender),
            AccountInfo::default().with_balance(U256::from(1_000_000_000u64)),
        );
    }
    db.insert_account(
        CONTRACT,
        AccountInfo::default().with_code(Bytecode::new_raw(Bytes::copy_from_slice(code))),
    );
    db
}

// These fixtures put the entire parent state in the cache over an empty backing database.
// Include unchanged accounts and storage as well as the committed transaction changes.
fn state_root<DB: revm::Database>(db: &State<DB>) -> B256 {
    alloy_trie::root::state_root_unhashed(db.cache.trie_account().into_iter().map(
        |(address, account)| {
            let storage_root = alloy_trie::root::storage_root_unhashed(
                account
                    .storage
                    .iter()
                    .filter(|(_, value)| !value.is_zero())
                    .map(|(slot, value)| (B256::from(*slot), *value)),
            );
            (
                address,
                alloy_trie::TrieAccount {
                    nonce: account.info.nonce,
                    balance: account.info.balance,
                    code_hash: account.info.code_hash,
                    storage_root,
                },
            )
        },
    ))
}

fn run(
    config: ParallelExecutionConfig,
    txs: &[Recovered<OpTxEnvelope>],
    code: &[u8],
    reject: Option<usize>,
    beneficiary: Address,
) -> Reference {
    let mut db = database(code);
    run_on(&mut db, config, txs, reject, beneficiary, PolicyState::default())
}

fn run_on(
    db: &mut State<InMemoryDB>,
    config: ParallelExecutionConfig,
    txs: &[Recovered<OpTxEnvelope>],
    reject: Option<usize>,
    beneficiary: Address,
    seed: PolicyState,
) -> Reference {
    run_on_with_statistics(db, config, txs, reject, beneficiary, seed, None).0
}

#[derive(Clone, Debug)]
struct ParentReader(Arc<InMemoryDB>, Arc<std::sync::atomic::AtomicU64>);
impl revm::Database for ParentReader {
    type Error = reth_optimism_parallel::SpeculationError;
    fn basic(&mut self, address: Address) -> Result<Option<AccountInfo>, Self::Error> {
        self.1.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        Ok(revm::DatabaseRef::basic_ref(&*self.0, address).unwrap())
    }
    fn storage(&mut self, address: Address, slot: U256) -> Result<U256, Self::Error> {
        self.1.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        Ok(revm::DatabaseRef::storage_ref(&*self.0, address, slot).unwrap())
    }
    fn code_by_hash(&mut self, hash: B256) -> Result<Bytecode, Self::Error> {
        self.1.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        Ok(revm::DatabaseRef::code_by_hash_ref(&*self.0, hash).unwrap())
    }
    fn block_hash(&mut self, number: u64) -> Result<B256, Self::Error> {
        self.1.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        Ok(revm::DatabaseRef::block_hash_ref(&*self.0, number).unwrap())
    }
}

fn independent_session(db: &State<InMemoryDB>) -> Arc<reth_optimism_parallel::SnapshotSession> {
    let parent = ParentReader(Arc::new(db.database.clone()), Arc::default());
    Arc::new(reth_optimism_parallel::SnapshotSession::new(
        Arc::new(move || {
            Ok(Box::new(parent.clone())
                as Box<dyn revm::Database<Error = reth_optimism_parallel::SpeculationError>>)
        }),
        64 * 1024 * 1024,
    ))
}

fn run_on_with_statistics(
    db: &mut State<InMemoryDB>,
    config: ParallelExecutionConfig,
    txs: &[Recovered<OpTxEnvelope>],
    reject: Option<usize>,
    beneficiary: Address,
    seed: PolicyState,
    configure_precompiles: Option<ConfigurePrecompiles>,
) -> (Reference, reth_optimism_parallel::ParallelStatistics) {
    let mode = config.mode;
    let direct = config.state_reads == reth_optimism_parallel::StateReads::Auto;
    let expect_workers = config.max_read_bytes > 4096 &&
        config.max_output_bytes > 4096 &&
        config.max_speculative_gas >= txs[0].gas_limit() &&
        (mode == ExecutionMode::Shadow ||
            (config.max_in_flight > 1 &&
                txs.iter().any(|tx| tx.signer() != txs[0].signer())));
    let mut factory = OpBlockExecutorFactory::new(
        OpAlloyReceiptBuilder::default(),
        OpChainHardforks::new([
            (OpHardfork::Bedrock, ForkCondition::Block(0)),
            (OpHardfork::Regolith, ForkCondition::Timestamp(0)),
            (OpHardfork::Jovian, ForkCondition::Timestamp(0)),
        ]),
        OpEvmFactory::<crate::OpTx, ObservedRefundPolicy<Policy>>::default(),
    );
    if mode != ExecutionMode::Sequential {
        factory = factory.with_parallel_runtime(Arc::new(ParallelRuntime::new(config).unwrap()));
    }
    let runtime = factory.parallel_runtime().cloned();
    let session = direct.then(|| independent_session(db));
    let mut evm = factory.evm_factory.create_evm(
        &mut *db,
        EvmEnv::new(
            CfgEnv::new_with_spec(OpSpecId::JOVIAN),
            BlockEnv {
                timestamp: U256::from(JOVIAN_TIMESTAMP),
                number: U256::from(1),
                gas_limit: 10_000_000,
                basefee: 2,
                beneficiary,
                ..Default::default()
            },
        ),
    );
    if let Some(configure) = configure_precompiles {
        configure(evm.precompiles_mut());
    }
    let candidates = txs
        .iter()
        .map(|tx| ParallelCandidate {
            hash: tx.trie_hash(),
            transaction: crate::OpTx::from_recovered_tx(tx.inner(), tx.signer()),
        })
        .collect();
    let mut executor = factory.create_executor_with_snapshot(
        evm,
        OpBlockExecutionCtx {
            post_exec_mode: PostExecMode::Produce,
            parallel_candidates: candidates,
            ..Default::default()
        },
        session,
    );
    executor.seed_refund_snapshot(seed);
    let mut outputs = Vec::new();
    for (index, tx) in txs.iter().enumerate() {
        let mut execution = None;
        let result = executor.execute_transaction_with_commit_condition(tx, |output| {
            execution = Some(output.inner.result.result.clone());
            if reject == Some(index) { CommitChanges::No } else { CommitChanges::Yes }
        });
        outputs.push(match result {
            Ok(_) => Ok(execution.unwrap()),
            Err(error) => Err(error.to_string()),
        });
    }
    if mode == ExecutionMode::Shadow {
        assert!(executor.parallel.is_some(), "shadow mismatch disabled the runtime");
    }
    let statistics = runtime.as_ref().map(|runtime| runtime.statistics()).unwrap_or_default();
    if direct {
        assert_eq!(statistics.broker_reads, 0);
    } else {
        assert_eq!(statistics.direct_reads, 0);
    }
    if let Some(runtime) = runtime.filter(|_| expect_workers) {
        let statistics = runtime.statistics();
        assert!(statistics.completed > 0, "test must execute successful speculative work");
        if mode == ExecutionMode::Parallel && configure_precompiles.is_none() {
            assert!(statistics.reused > 0, "test must reuse validated work");
        }
        if mode == ExecutionMode::Shadow && configure_precompiles.is_none() {
            assert!(statistics.shadow_matches > 0, "test must compare worker results");
        }
        assert_eq!(statistics.shadow_mismatches, 0);
    }
    let mut reference = Reference {
        bundle: BundleState::default(),
        state_root: B256::ZERO,
        receipt_root: alloy_consensus::proofs::calculate_receipt_root(&executor.receipts),
        receipts: executor.receipts.clone(),
        outputs,
        pre_refund_gas: executor.evm_gas_used,
        canonical_gas: executor.gas_used,
        da_gas: executor.da_footprint_used,
        entries: executor.post_exec_entries().to_vec(),
        policy: executor.refund_snapshot(),
    };
    drop(executor);
    reference.state_root = state_root(db);
    db.merge_transitions(BundleRetention::Reverts);
    reference.bundle = db.take_bundle();
    (reference, statistics)
}

fn assert_parity(
    txs: &[Recovered<OpTxEnvelope>],
    code: &[u8],
    reject: Option<usize>,
    beneficiary: Address,
) -> Reference {
    let reference = run(ParallelExecutionConfig::default(), txs, code, reject, beneficiary);
    for workers in [1, 2, 4, 8] {
        for mode in [ExecutionMode::Shadow, ExecutionMode::Parallel] {
            for (state_reads, scheduler) in [
                (reth_optimism_parallel::StateReads::Broker, ExecutionScheduler::Window),
                (reth_optimism_parallel::StateReads::Auto, ExecutionScheduler::Window),
                (reth_optimism_parallel::StateReads::Broker, ExecutionScheduler::Rolling),
                (reth_optimism_parallel::StateReads::Auto, ExecutionScheduler::Rolling),
            ] {
                let config = ParallelExecutionConfig {
                    mode,
                    workers,
                    state_reads,
                    scheduler,
                    ..Default::default()
                };
                assert_eq!(
                    run(config, txs, code, reject, beneficiary),
                    reference,
                    "{mode:?}, {state_reads:?}, {scheduler:?}, workers={workers}"
                );
            }
        }
    }
    reference
}

#[test]
fn conflicts_nonce_chains_reverts_and_policy_reads_match_reference() {
    // Increment a shared storage slot. Every stale observation/read must retry once in order.
    let code = [0x60, 0, 0x54, 0x60, 1, 0x01, 0x60, 0, 0x55, 0x00];
    let txs = vec![
        transfer(1, 0, CONTRACT),
        transfer(2, 0, CONTRACT),
        transfer(1, 1, CONTRACT),
        transfer(3, 0, CONTRACT),
    ];
    let result = assert_parity(&txs, &code, None, BENEFICIARY);
    assert_eq!(result.policy.evaluations.len(), 4);
    assert!(result.canonical_gas < result.pre_refund_gas);
    let reverted = assert_parity(
        &txs,
        &[0x60, 0x2a, 0x60, 0, 0x52, 0x60, 0x20, 0x60, 0, 0xfd],
        None,
        BENEFICIARY,
    );
    assert_eq!(reverted.policy.evaluations.iter().map(|(_, n)| n).sum::<usize>(), 4);
}

#[test]
fn rejected_and_failed_attempts_do_not_advance_policy() {
    let txs = vec![
        transfer(1, 0, CONTRACT),
        transfer(1, 1, CONTRACT),
        transfer(2, 0, CONTRACT),
        transfer(3, 0, CONTRACT),
    ];
    let result = assert_parity(&txs, &[0], Some(0), BENEFICIARY);
    assert!(result.outputs[1].is_err(), "descendant of declined tx remains invalid");
    assert_eq!(result.policy.evaluations.len(), 2);
    assert_eq!(result.policy.evaluations[0].0.tx_index, 0);
}

#[test]
fn sender_descendants_preserve_other_results_and_speculative_budget() {
    let txs = vec![
        transfer(1, 0, Address::repeat_byte(0x51)),
        transfer(1, 1, Address::repeat_byte(0x52)),
        transfer(2, 0, Address::repeat_byte(0x53)),
    ];
    for reject in [None, Some(0)] {
        let reference = run(ParallelExecutionConfig::default(), &txs, &[0], reject, BENEFICIARY);
        let (actual, statistics) = run_on_with_statistics(
            &mut database(&[0]),
            ParallelExecutionConfig {
                mode: ExecutionMode::Parallel,
                workers: 2,
                max_in_flight: 2,
                max_speculative_gas: 300_000,
                ..Default::default()
            },
            &txs,
            reject,
            BENEFICIARY,
            PolicyState::default(),
            None,
        );
        assert_eq!(actual, reference);
        assert_eq!(
            statistics.attempted, 2,
            "descendant must not consume the second job or its gas budget"
        );
        assert_eq!(statistics.completed, 2);
        assert_eq!(
            statistics.reused, 2,
            "canonical descendant must preserve the other sender's output"
        );
    }
}

#[test_case::test_case(ExecutionScheduler::Window)]
#[test_case::test_case(ExecutionScheduler::Rolling)]
fn single_sender_chain_avoids_worker_round_trips(scheduler: ExecutionScheduler) {
    let txs: Vec<_> = (0..8).map(|nonce| transfer(1, nonce, CONTRACT)).collect();
    let reference = run(ParallelExecutionConfig::default(), &txs, &[0], None, BENEFICIARY);
    let (actual, statistics) = run_on_with_statistics(
        &mut database(&[0]),
        ParallelExecutionConfig {
            mode: ExecutionMode::Parallel,
            scheduler,
            workers: 4,
            ..Default::default()
        },
        &txs,
        None,
        BENEFICIARY,
        PolicyState::default(),
        None,
    );
    assert_eq!(actual, reference);
    assert_eq!(statistics.attempted, 0);
}

#[test_case::test_case(ExecutionScheduler::Window)]
#[test_case::test_case(ExecutionScheduler::Rolling)]
fn fresh_preview_reopens_speculation_after_a_serial_sender_chain(scheduler: ExecutionScheduler) {
    use crate::post_exec::PostExecExecutorExt;
    let txs = vec![
        transfer(1, 0, Address::repeat_byte(0x51)),
        transfer(1, 1, Address::repeat_byte(0x52)),
        transfer(2, 0, Address::repeat_byte(0x53)),
    ];
    let reference = run(ParallelExecutionConfig::default(), &txs, &[0], None, BENEFICIARY);
    let mut db = database(&[0]);
    let runtime = Arc::new(
        ParallelRuntime::new(ParallelExecutionConfig {
            mode: ExecutionMode::Parallel,
            scheduler,
            workers: 2,
            ..Default::default()
        })
        .unwrap(),
    );
    let factory = OpBlockExecutorFactory::new(
        OpAlloyReceiptBuilder::default(),
        OpChainHardforks::op_mainnet(),
        OpEvmFactory::<crate::OpTx, ObservedRefundPolicy<Policy>>::default(),
    )
    .with_parallel_runtime(runtime.clone());
    let session = independent_session(&db);
    let evm = factory.evm_factory.create_evm(
        &mut db,
        EvmEnv::new(
            CfgEnv::new_with_spec(OpSpecId::JOVIAN),
            BlockEnv {
                timestamp: U256::from(JOVIAN_TIMESTAMP),
                number: U256::from(1),
                gas_limit: 10_000_000,
                basefee: 2,
                beneficiary: BENEFICIARY,
                ..Default::default()
            },
        ),
    );
    let mut executor = factory.create_executor_with_snapshot(
        evm,
        OpBlockExecutionCtx { post_exec_mode: PostExecMode::Produce, ..Default::default() },
        Some(session),
    );
    executor.execute_transaction(&txs[0]).unwrap();
    assert_eq!(runtime.statistics().attempted, 0);
    executor.set_parallel_candidates(
        txs[1..]
            .iter()
            .map(|tx| ParallelCandidate {
                hash: tx.trie_hash(),
                transaction: crate::OpTx::from_recovered_tx(tx.inner(), tx.signer()),
            })
            .collect(),
    );
    for tx in &txs[1..] {
        executor.execute_transaction(tx).unwrap();
    }
    assert_eq!(runtime.statistics().reused, 2);
    assert_eq!(executor.refund_snapshot(), reference.policy);
    assert_eq!(executor.receipts, reference.receipts);
    drop(executor);
    assert_eq!(state_root(&db), reference.state_root);
}

#[test]
fn protocol_fee_aliases_and_explicit_vault_calls_match_reference() {
    for beneficiary in [
        BENEFICIARY,
        BASE_FEE_RECIPIENT,
        L1_FEE_RECIPIENT,
        OPERATOR_FEE_RECIPIENT,
        Address::repeat_byte(1),
    ] {
        let txs = vec![
            transfer(1, 0, CONTRACT),
            transfer(2, 0, BASE_FEE_RECIPIENT),
            transfer(1, 1, L1_FEE_RECIPIENT),
            transfer(3, 0, OPERATOR_FEE_RECIPIENT),
        ];
        assert_parity(&txs, &[0], None, beneficiary);
    }
}

#[test]
fn creations_selfdestructs_and_deposits_preserve_boundaries() {
    let mut create = transfer(1, 0, CONTRACT);
    // Initcode returns empty runtime code. Use a fresh envelope to preserve its hash.
    create = recovered_legacy_from(
        create.signer(),
        TxLegacy {
            to: TxKind::Create,
            input: Bytes::from_static(&[0x60, 0, 0x60, 0, 0xf3]),
            gas_limit: 150_000,
            gas_price: 10,
            ..Default::default()
        },
    );
    let deposit = Recovered::new_unchecked(
        OpTxEnvelope::Deposit(
            TxDeposit {
                from: Address::repeat_byte(4),
                to: TxKind::Call(CONTRACT),
                gas_limit: 100_000,
                mint: 100_000,
                ..Default::default()
            }
            .seal_slow(),
        ),
        Address::repeat_byte(4),
    );
    let txs = vec![create, transfer(2, 0, CONTRACT), deposit, transfer(3, 0, CONTRACT)];
    // Selfdestruct to a fee vault, with zero value and touch/lifecycle behavior preserved.
    let mut code = vec![0x73];
    code.extend_from_slice(BASE_FEE_RECIPIENT.as_slice());
    code.push(0xff);
    let result = assert_parity(&txs, &code, None, BENEFICIARY);
    assert_eq!(result.policy.evaluations[2].0.kind, PostExecTxKind::Deposit);
}

#[test_case::test_case(ExecutionScheduler::Window)]
#[test_case::test_case(ExecutionScheduler::Rolling)]
fn resource_exhaustion_falls_back_and_subblocks_carry_policy_state(scheduler: ExecutionScheduler) {
    let txs = vec![transfer(1, 0, CONTRACT), transfer(2, 0, CONTRACT)];
    let reference = run(ParallelExecutionConfig::default(), &txs, &[0], None, BENEFICIARY);
    for config in [
        ParallelExecutionConfig { max_read_bytes: 1, ..Default::default() },
        ParallelExecutionConfig { max_output_bytes: 1, ..Default::default() },
        ParallelExecutionConfig { max_speculative_gas: 1, ..Default::default() },
    ] {
        assert_eq!(
            run(
                ParallelExecutionConfig {
                    mode: ExecutionMode::Parallel,
                    scheduler,
                    workers: 2,
                    ..config
                },
                &txs,
                &[0],
                None,
                BENEFICIARY
            ),
            reference
        );
    }
    let mut sequential = database(&[0]);
    let mut parallel = database(&[0]);
    let mut seed = PolicyState::default();
    for tx in &txs {
        let expected = run_on(
            &mut sequential,
            ParallelExecutionConfig::default(),
            core::slice::from_ref(tx),
            None,
            BENEFICIARY,
            seed.clone(),
        );
        let actual = run_on(
            &mut parallel,
            ParallelExecutionConfig {
                mode: ExecutionMode::Parallel,
                scheduler,
                workers: 2,
                ..Default::default()
            },
            core::slice::from_ref(tx),
            None,
            BENEFICIARY,
            seed,
        );
        assert_eq!(actual, expected);
        seed = expected.policy;
    }
    assert_eq!(seed.evaluations.len(), 2);
}

#[test]
fn preceding_refund_can_fund_the_next_transaction() {
    let txs = vec![transfer(1, 0, CONTRACT), transfer(2, 0, CONTRACT), transfer(2, 1, CONTRACT)];
    // Set the sender's opening balance so only the preceding policy refund makes nonce 1 valid.
    let mut probe = database(&[0]);
    let first = run_on(
        &mut probe,
        ParallelExecutionConfig::default(),
        &txs[..2],
        None,
        BENEFICIARY,
        PolicyState::default(),
    );
    let spent = U256::from(1_000_000_000u64) -
        probe.basic(Address::repeat_byte(2)).unwrap().unwrap().balance;
    let mut info = L1BlockInfo::try_fetch(&mut probe, U256::from(1), OpSpecId::JOVIAN).unwrap();
    let encoded = txs[2].encoded_2718();
    let required = U256::from(txs[2].gas_limit()) * U256::from(txs[2].gas_price().unwrap()) +
        txs[2].value() +
        info.operator_fee_charge(&encoded, U256::from(txs[2].gas_limit()), OpSpecId::JOVIAN) +
        info.calculate_tx_l1_cost(&encoded, OpSpecId::JOVIAN);
    let initial = spent + required;
    let mut expected = None;
    for mode in [ExecutionMode::Sequential, ExecutionMode::Shadow, ExecutionMode::Parallel] {
        let mut db = database(&[0]);
        db.insert_account(Address::repeat_byte(2), AccountInfo::default().with_balance(initial));
        let result = run_on(
            &mut db,
            ParallelExecutionConfig { mode, workers: 4, ..Default::default() },
            &txs,
            None,
            BENEFICIARY,
            PolicyState::default(),
        );
        assert!(
            result.outputs.iter().all(Result::is_ok),
            "refund-funded transaction must be valid: {:?}",
            result.outputs
        );
        if let Some(expected) = &expected {
            assert_eq!(&result, expected);
        } else {
            expected = Some(result);
        }
    }
    assert!(!first.entries.is_empty());
}

#[test]
fn delegate_code_and_precompile_calls_match_reference() {
    let delegate = Address::repeat_byte(0x44);
    let txs = vec![
        transfer(1, 0, delegate),
        transfer(2, 0, delegate),
        transfer(3, 0, Address::with_last_byte(4)),
    ];
    let mut reference = None;
    for mode in [ExecutionMode::Sequential, ExecutionMode::Shadow, ExecutionMode::Parallel] {
        let mut db = database(&[0x60, 1, 0x60, 0, 0x55, 0]);
        db.insert_account(
            delegate,
            AccountInfo::default().with_code(Bytecode::new_eip7702(CONTRACT)),
        );
        let actual = run_on(
            &mut db,
            ParallelExecutionConfig { mode, workers: 2, ..Default::default() },
            &txs,
            None,
            BENEFICIARY,
            PolicyState::default(),
        );
        if let Some(reference) = &reference {
            assert_eq!(&actual, reference);
        } else {
            reference = Some(actual);
        }
    }
}

fn precompile_transactions(target: Address, input: &[u8]) -> Vec<Recovered<OpTxEnvelope>> {
    (1..=3)
        .map(|sender| {
            recovered_legacy_from(
                Address::repeat_byte(sender),
                TxLegacy {
                    gas_limit: 150_000,
                    gas_price: 10 + u128::from(sender),
                    to: TxKind::Call(target),
                    input: Bytes::copy_from_slice(input),
                    ..Default::default()
                },
            )
        })
        .collect()
}

#[test]
fn stock_precompile_success_halt_and_reverted_calls_reuse_worker_results() {
    // CALL identity, discard success, then REVERT with its output buffer.
    let reverted_call = [
        0x60, 0x20, 0x5f, 0x5f, 0x5f, 0x5f, 0x60, 4, 0x62, 1, 0x86, 0xa0, 0xf1, 0x50, 0x60, 0x20,
        0x5f, 0xfd,
    ];
    for (target, input, code) in [
        (Address::with_last_byte(4), b"identity".as_slice(), [0].as_slice()),
        (Address::with_last_byte(2), b"sha256".as_slice(), [0].as_slice()),
        (Address::with_last_byte(8), [].as_slice(), [0].as_slice()),
        (Address::with_last_byte(8), [1].as_slice(), [0].as_slice()),
        (CONTRACT, [].as_slice(), reverted_call.as_slice()),
    ] {
        let txs = precompile_transactions(target, input);
        let reference = run(ParallelExecutionConfig::default(), &txs, code, None, BENEFICIARY);
        for result in &reference.outputs {
            let result = result.as_ref().unwrap();
            if target == CONTRACT {
                assert!(matches!(result, ExecutionResult::Revert { .. }));
            } else if target == Address::with_last_byte(8) && !input.is_empty() {
                assert!(matches!(result, ExecutionResult::Halt { .. }));
            } else {
                assert!(result.is_success());
            }
        }
        for workers in [1, 2, 4] {
            for mode in [ExecutionMode::Parallel, ExecutionMode::Shadow] {
                let (actual, statistics) = run_on_with_statistics(
                    &mut database(code),
                    ParallelExecutionConfig { mode, workers, ..Default::default() },
                    &txs,
                    None,
                    BENEFICIARY,
                    PolicyState::default(),
                    None,
                );
                assert_eq!(actual, reference);
                assert_eq!(statistics.completed, txs.len() as u64);
                if mode == ExecutionMode::Parallel {
                    assert_eq!(statistics.reused, txs.len() as u64);
                } else {
                    assert_eq!(statistics.shadow_matches, txs.len() as u64);
                }
            }
        }
    }
}

#[test]
fn custom_precompiles_and_opaque_wrappers_keep_canonical_behavior() {
    use alloy_evm::precompiles::{DynPrecompile, Precompile, PrecompilesMap};
    use revm::precompile::{PrecompileId, PrecompileOutput};
    fn replace(map: &mut PrecompilesMap) {
        map.map_precompile(&Address::with_last_byte(4), |original| {
            // Same ID, address, warmth and cacheability; deliberately different output.
            DynPrecompile::new(original.precompile_id().clone(), |input| {
                Ok(PrecompileOutput::new(15, Bytes::from_static(b"custom"), input.reservoir))
            })
        });
    }
    fn wrap(map: &mut PrecompilesMap) {
        map.map_precompiles(|_, original| {
            DynPrecompile::new(original.precompile_id().clone(), move |input| original.call(input))
        });
    }
    fn lookup(map: &mut PrecompilesMap) {
        map.set_precompile_lookup(|address: &Address| {
            (*address == CONTRACT).then(|| {
                DynPrecompile::new(PrecompileId::Custom("lookup".into()), |input| {
                    Ok(PrecompileOutput::new(15, Bytes::from_static(b"dynamic"), input.reservoir))
                })
            })
        });
    }
    fn remove(map: &mut PrecompilesMap) {
        map.apply_precompile(&Address::with_last_byte(4), |_| None);
    }
    let configurations: [(Address, ConfigurePrecompiles); 4] = [
        (Address::with_last_byte(4), replace),
        (Address::with_last_byte(4), wrap),
        (CONTRACT, lookup),
        (Address::with_last_byte(4), remove),
    ];
    for (target, configure) in configurations {
        let txs = precompile_transactions(target, b"input");
        let (reference, _) = run_on_with_statistics(
            &mut database(&[0]),
            ParallelExecutionConfig::default(),
            &txs,
            None,
            BENEFICIARY,
            PolicyState::default(),
            Some(configure),
        );
        for mode in [ExecutionMode::Parallel, ExecutionMode::Shadow] {
            let (actual, statistics) = run_on_with_statistics(
                &mut database(&[0]),
                ParallelExecutionConfig { mode, workers: 4, ..Default::default() },
                &txs,
                None,
                BENEFICIARY,
                PolicyState::default(),
                Some(configure),
            );
            assert_eq!(actual, reference);
            assert_eq!(statistics.completed, txs.len() as u64);
            assert_eq!(statistics.reused, 0);
            assert_eq!(statistics.shadow_matches, 0);
            assert_eq!(statistics.shadow_mismatches, 0);
        }
    }
}

#[test]
fn checked_fee_overflow_and_vault_storage_writes_match_reference() {
    let txs = vec![transfer(1, 0, BASE_FEE_RECIPIENT), transfer(2, 0, CONTRACT)];
    let mut reference = None;
    for mode in [ExecutionMode::Sequential, ExecutionMode::Shadow, ExecutionMode::Parallel] {
        let mut db = database(&[0]);
        db.insert_account(BENEFICIARY, AccountInfo::default().with_balance(U256::MAX));
        db.insert_account(
            BASE_FEE_RECIPIENT,
            AccountInfo::default()
                .with_code(Bytecode::new_raw(Bytes::from_static(&[0x60, 1, 0x60, 0, 0x55, 0]))),
        );
        let actual = run_on(
            &mut db,
            ParallelExecutionConfig { mode, workers: 2, ..Default::default() },
            &txs,
            None,
            BENEFICIARY,
            PolicyState::default(),
        );
        if let Some(reference) = &reference {
            assert_eq!(&actual, reference);
        } else {
            reference = Some(actual);
        }
    }
}

#[test]
fn opaque_policy_is_rejected_when_parallel_production_is_selected() {
    let factory = OpBlockExecutorFactory::new(
        OpAlloyReceiptBuilder::default(),
        OpChainHardforks::op_mainnet(),
        OpEvmFactory::<crate::OpTx, ScriptedRefundPolicy>::default(),
    )
    .with_parallel_runtime(Arc::new(
        ParallelRuntime::new(ParallelExecutionConfig {
            mode: ExecutionMode::Parallel,
            workers: 1,
            ..Default::default()
        })
        .unwrap(),
    ));
    let mut db = database(&[0]);
    let evm = factory.evm_factory.create_evm(&mut db, EvmEnv::default());
    let mut executor = factory.create_executor(
        evm,
        OpBlockExecutionCtx { post_exec_mode: PostExecMode::Produce, ..Default::default() },
    );
    assert!(
        executor
            .apply_pre_execution_changes()
            .unwrap_err()
            .to_string()
            .contains("observation/evaluation")
    );
}

#[test_case::test_case(ExecutionScheduler::Window)]
#[test_case::test_case(ExecutionScheduler::Rolling)]
fn shuffled_sender_schedules_match_at_different_window_sizes(scheduler: ExecutionScheduler) {
    for seed in 1..=8u64 {
        let mut rng = seed;
        let mut nonces = [0; 8];
        let txs: Vec<_> = (0..24)
            .map(|_| {
                rng = rng.wrapping_mul(6364136223846793005).wrapping_add(1);
                let index = (rng >> 32) as usize % 8;
                let nonce = nonces[index];
                nonces[index] += 1;
                transfer(index as u8 + 1, nonce, CONTRACT)
            })
            .collect();
        let reference = run(ParallelExecutionConfig::default(), &txs, &[0], None, BENEFICIARY);
        for max_in_flight in [1, 3, 8] {
            let result = run(
                ParallelExecutionConfig {
                    mode: ExecutionMode::Parallel,
                    scheduler,
                    workers: 4,
                    max_in_flight,
                    ..Default::default()
                },
                &txs,
                &[0],
                None,
                BENEFICIARY,
            );
            assert_eq!(result, reference, "schedule {seed}, window {max_in_flight}");
        }
    }
}

#[test]
fn validator_uses_declared_refunds_and_structural_post_exec() {
    let txs = vec![transfer(1, 0, CONTRACT), transfer(2, 0, CONTRACT), transfer(1, 1, CONTRACT)];
    let reference = run(ParallelExecutionConfig::default(), &txs, &[0], None, BENEFICIARY);
    assert!(!reference.entries.is_empty());
    for mode in [ExecutionMode::Sequential, ExecutionMode::Shadow, ExecutionMode::Parallel] {
        let mut db = database(&[0]);
        // A null policy verifies the stateful producer's declared refund decisions.
        let mut factory = OpBlockExecutorFactory::new(
            OpAlloyReceiptBuilder::default(),
            OpChainHardforks::new([
                (OpHardfork::Bedrock, ForkCondition::Block(0)),
                (OpHardfork::Regolith, ForkCondition::Timestamp(0)),
                (OpHardfork::Jovian, ForkCondition::Timestamp(0)),
            ]),
            OpEvmFactory::<crate::OpTx>::default(),
        );
        if mode != ExecutionMode::Sequential {
            factory = factory.with_parallel_runtime(Arc::new(
                ParallelRuntime::new(ParallelExecutionConfig {
                    mode,
                    workers: 4,
                    ..Default::default()
                })
                .unwrap(),
            ));
        }
        let evm = factory.evm_factory.create_evm(
            &mut db,
            EvmEnv::new(
                CfgEnv::new_with_spec(OpSpecId::JOVIAN),
                BlockEnv {
                    timestamp: U256::from(JOVIAN_TIMESTAMP),
                    number: U256::from(1),
                    gas_limit: 10_000_000,
                    basefee: 2,
                    beneficiary: BENEFICIARY,
                    ..Default::default()
                },
            ),
        );
        let candidates = txs
            .iter()
            .map(|tx| ParallelCandidate {
                hash: tx.trie_hash(),
                transaction: crate::OpTx::from_recovered_tx(tx.inner(), tx.signer()),
            })
            .collect();
        let mut executor = factory.create_executor(
            evm,
            OpBlockExecutionCtx {
                post_exec_mode: PostExecMode::Verify(PostExecPayload {
                    version: 1,
                    block_number: 1,
                    gas_refund_entries: reference.entries.clone(),
                }),
                parallel_candidates: candidates,
                ..Default::default()
            },
        );
        for tx in &txs {
            executor.execute_transaction(tx).unwrap();
        }
        assert_eq!(executor.receipts, reference.receipts);
        assert_eq!(
            alloy_consensus::proofs::calculate_receipt_root(&executor.receipts),
            reference.receipt_root
        );
        assert_eq!(executor.evm_gas_used, reference.pre_refund_gas);
        let post_exec = Recovered::new_unchecked(
            OpTxEnvelope::PostExec(build_post_exec_tx(1, reference.entries.clone()).seal_slow()),
            Address::ZERO,
        );
        assert_eq!(executor.execute_transaction(&post_exec).unwrap().tx_gas_used(), 0);
        let (_, result) = executor.finish().unwrap();
        assert_eq!(result.gas_used, reference.canonical_gas);
        assert_eq!(state_root(&db), reference.state_root);
        db.merge_transitions(BundleRetention::Reverts);
        assert_eq!(db.take_bundle(), reference.bundle);
    }
}

#[test]
fn prepared_policy_state_is_private_until_commit() {
    let mut policy = ObservedRefundPolicy::<Policy>::default();
    policy.begin_tx(PostExecTxContext { tx_index: 0, kind: PostExecTxKind::Normal });
    policy.note_account_touch(CONTRACT);
    let _ = policy.finish_tx();
    assert!(policy.snapshot().touched.is_empty());
    policy.commit_tx();
    assert_eq!(policy.snapshot().touched.get(&CONTRACT), Some(&0));
    let committed = policy.snapshot();
    policy.begin_tx(PostExecTxContext { tx_index: 1, kind: PostExecTxKind::Normal });
    policy.note_account_touch(BENEFICIARY);
    let _ = policy.finish_tx();
    policy.restore(committed.clone());
    policy.commit_tx();
    assert_eq!(policy.snapshot(), committed);
}

#[test_case::test_case(ExecutionScheduler::Window)]
#[test_case::test_case(ExecutionScheduler::Rolling)]
fn only_committed_results_reach_state_hooks(scheduler: ExecutionScheduler) {
    use std::sync::Mutex;
    let mut db = database(&[0]);
    let seen = Arc::new(Mutex::new(Vec::new()));
    let hook = seen.clone();
    db.set_state_hook(Some(Box::new(move |state: EvmState| {
        hook.lock().unwrap().push(state);
    })));
    let txs = vec![transfer(1, 0, CONTRACT), transfer(2, 0, CONTRACT), transfer(3, 0, CONTRACT)];
    let result = run_on(
        &mut db,
        ParallelExecutionConfig {
            mode: ExecutionMode::Parallel,
            scheduler,
            workers: 4,
            ..Default::default()
        },
        &txs,
        Some(1),
        BENEFICIARY,
        PolicyState::default(),
    );
    assert_eq!(result.receipts.len(), 2);
    let seen = seen.lock().unwrap();
    assert_eq!(seen.len(), 2);
    assert!(seen.iter().all(|state| !state.contains_key(&Address::repeat_byte(2))));
}

#[test]
fn shadow_compares_pending_policy_state_even_when_refunds_match() {
    let tx = transfer(1, 0, CONTRACT);
    let reference = run(
        ParallelExecutionConfig::default(),
        core::slice::from_ref(&tx),
        &[0],
        None,
        BENEFICIARY,
    );
    let mut db = database(&[0]);
    let runtime = Arc::new(
        ParallelRuntime::new(ParallelExecutionConfig {
            mode: ExecutionMode::Shadow,
            workers: 2,
            ..Default::default()
        })
        .unwrap(),
    );
    let factory = OpBlockExecutorFactory::new(
        OpAlloyReceiptBuilder::default(),
        OpChainHardforks::op_mainnet(),
        OpEvmFactory::<crate::OpTx, ObservedRefundPolicy<Policy>>::default(),
    )
    .with_parallel_runtime(runtime.clone());
    let evm = factory.evm_factory.create_evm(
        &mut db,
        EvmEnv::new(
            CfgEnv::new_with_spec(OpSpecId::JOVIAN),
            BlockEnv {
                timestamp: U256::from(JOVIAN_TIMESTAMP),
                number: U256::from(1),
                gas_limit: 10_000_000,
                basefee: 2,
                beneficiary: BENEFICIARY,
                ..Default::default()
            },
        ),
    );
    let mut executor = factory.create_executor(
        evm,
        OpBlockExecutionCtx { post_exec_mode: PostExecMode::Produce, ..Default::default() },
    );
    let state = executor.parallel.as_mut().unwrap();
    let worker = state.worker.clone();
    state.worker = Arc::new(move |job, db| {
        let mut output = worker(job, db)?;
        // This wrong observation still yields a zero refund on the first transaction, but would
        // change later refunds if its prepared policy state were committed in parallel mode.
        output.raw.observation = Some(crate::post_exec::ParallelObservation::new(
            Observation { touches: vec![Address::repeat_byte(0x99)], reverted: 0 },
            64,
        ));
        Ok(output)
    });
    executor.execute_transaction(&tx).unwrap();
    assert_eq!(runtime.statistics().shadow_mismatches, 1);
    assert!(executor.parallel.is_none());
    assert_eq!(executor.refund_snapshot(), reference.policy);
    assert_eq!(executor.receipts, reference.receipts);
    drop(executor);
    assert_eq!(state_root(&db), reference.state_root);
}

#[test]
fn shadow_matching_cumulative_limit_rejections_are_not_mismatches() {
    let first = transfer(1, 0, Address::repeat_byte(0x51));
    let second = recovered_legacy_from(
        Address::repeat_byte(2),
        TxLegacy {
            gas_limit: 9_999_999,
            gas_price: 10,
            to: TxKind::Call(Address::repeat_byte(0x52)),
            ..Default::default()
        },
    );
    let result = assert_parity(&[first, second], &[0], None, BENEFICIARY);
    assert_eq!(result.receipts.len(), 1);
    assert!(result.outputs[1].is_err());
    assert_eq!(result.policy.evaluations.len(), 1);
}

/// A reproducible smoke benchmark for independent EVM loops, stock pairings and nonce chains,
/// including the policy and ordered settlement. Nonce chains cannot benefit from this scheduler.
#[test]
#[ignore = "manual synthetic execution benchmark"]
fn benchmark_optimistic_execution() {
    benchmark_execution(false);
}

/// Long blocks exercise refill, stragglers and ordered evaluation across many windows.
#[test]
#[ignore = "manual rolling execution benchmark"]
#[cfg(feature = "metrics")]
fn benchmark_rolling_execution() {
    benchmark_execution(true);
}

fn benchmark_execution(rolling: bool) {
    use metrics_util::debugging::{DebugValue, DebuggingRecorder};
    use std::time::Instant;
    let recorder = DebuggingRecorder::new();
    let snapshotter = recorder.snapshotter();
    recorder.install().unwrap();
    let count = if rolling { 64u8 } else { 8u8 };
    // EIP-197 identity pairing vector, also exercised by op-revm's pairing tests.
    const PAIRING_INPUT: &[u8] = &alloy_primitives::hex!(
        "2cf44499d5d27bb186308b7af7af02ac5bc9eeb6a3d147c186b21fb1b76e18da2c0f001f52110ccfe69108924926e45f0b0c868df0e7bde1fe16d3242dc715f61fb19bb476f6b9e44e2a32234da8212f61cd63919354bc06aef31e3cfaff3ebc22606845ff186793914e03e21df544c34ffe2f2f3504de8a79d9159eca2d98d92bd368e28381e8eccb5fa81fc26cf3f048eea9abfdd85d7ed3ab3698d63e4f902fe02e47887507adf0ff1743cbac6ba291e66f59be6bd763950bb16041a0a85e000000000000000000000000000000000000000000000000000000000000000130644e72e131a029b85045b68181585d97816a916871ca8d3c208c16d87cfd451971ff0471b09fa93caaf13cbf443c1aede09cc4328f5a62aad45f40ec133eb4091058a3141822985733cbdddfed0fd8d6c104e9e9eff40bf5abfef9ab163bc72a23af9a5ce2ba2796c1f4e453a370eb0af8c212d9dc9acd8fc02c2e907baea223a8eb0b0996252cb548a4487da97b02422ebc0e834613f954de6c7e0afdc1fc"
    );
    // Count down 3,000 iterations, using no shared storage writes.
    let code = [0x61, 0x0b, 0xb8, 0x5b, 0x60, 1, 0x90, 0x03, 0x80, 0x60, 3, 0x57, 0];
    for workload in [
        "independent",
        "nonce_chain",
        "pairing",
        "storage_shared_warm",
        "storage_independent_warm",
        "storage_shared_cold",
        "storage_independent_cold",
        "storage_conflict_cold",
        "slow_head",
        "slow_tail",
        "costly_policy",
    ] {
        if !rolling && matches!(workload, "slow_head" | "slow_tail" | "costly_policy") {
            continue;
        }
        POLICY_WORK.with(|work| work.set(if workload == "costly_policy" { 256 } else { 0 }));
        let mixed = matches!(workload, "slow_head" | "slow_tail");
        let slow_code = [0x61, 0x75, 0x30, 0x5b, 0x60, 1, 0x90, 0x03, 0x80, 0x60, 3, 0x57, 0];
        let fast_code = [0x60, 0x96, 0x5b, 0x60, 1, 0x90, 0x03, 0x80, 0x60, 2, 0x57, 0];
        let storage = workload.starts_with("storage_");
        let cold = workload.ends_with("_cold");
        let independent = workload.starts_with("storage_independent");
        let mut storage_code = Vec::new();
        for slot in 0..256u16 {
            storage_code.extend([0x61, (slot >> 8) as u8, slot as u8, 0x54, 0x50]);
        }
        if workload == "storage_conflict_cold" {
            storage_code.extend([0x60, 0, 0x54, 0x60, 1, 0x01, 0x60, 0, 0x55]);
        }
        storage_code.push(0);
        let code = if storage {
            storage_code.as_slice()
        } else if mixed {
            &slow_code
        } else {
            &code
        };
        let chained = workload == "nonce_chain";
        let pairing = workload == "pairing";
        let txs: Vec<_> = (0..count)
            .map(|i| {
                recovered_legacy_from(
                    Address::repeat_byte(if chained { 1 } else { i + 1 }),
                    TxLegacy {
                        nonce: if chained { u64::from(i) } else { 0 },
                        gas_limit: if storage || mixed { 1_000_000 } else { 150_000 },
                        gas_price: if pairing { 10 + u128::from(i) } else { 10 },
                        to: TxKind::Call(if pairing {
                            Address::with_last_byte(8)
                        } else if independent {
                            Address::with_last_byte(0x50 + i)
                        } else if mixed && i % 16 != if workload == "slow_head" { 0 } else { 15 } {
                            Address::repeat_byte(0x44)
                        } else {
                            CONTRACT
                        }),
                        input: if pairing {
                            Bytes::from_static(PAIRING_INPUT)
                        } else {
                            Bytes::from(vec![i])
                        },
                        ..Default::default()
                    },
                )
            })
            .collect();
        let mut reference = None;
        for (mode, workers) in [
            (ExecutionMode::Sequential, 1),
            (ExecutionMode::Shadow, 4),
            (ExecutionMode::Parallel, 1),
            (ExecutionMode::Parallel, 2),
            (ExecutionMode::Parallel, 4),
            (ExecutionMode::Parallel, 8),
        ] {
            for (state_reads, scheduler) in if rolling {
                vec![
                    (reth_optimism_parallel::StateReads::Auto, ExecutionScheduler::Window),
                    (reth_optimism_parallel::StateReads::Auto, ExecutionScheduler::Rolling),
                ]
            } else {
                vec![
                    (reth_optimism_parallel::StateReads::Broker, ExecutionScheduler::Window),
                    (reth_optimism_parallel::StateReads::Auto, ExecutionScheduler::Window),
                ]
            } {
                if mode == ExecutionMode::Sequential &&
                    (scheduler == ExecutionScheduler::Rolling ||
                        (!rolling && state_reads == reth_optimism_parallel::StateReads::Auto))
                {
                    continue;
                }
                let runtime = Arc::new(
                    ParallelRuntime::new(ParallelExecutionConfig {
                        mode,
                        workers,
                        state_reads,
                        scheduler,
                        max_speculative_gas: if rolling { 128_000_000 } else { 30_000_000 },
                        ..Default::default()
                    })
                    .unwrap(),
                );
                let factory = OpBlockExecutorFactory::new(
                    OpAlloyReceiptBuilder::default(),
                    OpChainHardforks::op_mainnet(),
                    OpEvmFactory::<crate::OpTx, ObservedRefundPolicy<Policy>>::default(),
                )
                .with_parallel_runtime(
                    (mode != ExecutionMode::Sequential).then(|| runtime.clone()),
                );
                let mut samples = Vec::new();
                let mut snapshot_bytes = 0;
                let mut provider_reads = 0;
                let mut timing: BTreeMap<String, f64> = BTreeMap::new();
                let mut snapshot_versions = 0.0f64;
                for iteration in 0..21 {
                    let mut db = database(code);
                    for sender in 1..=count {
                        db.insert_account(
                            Address::repeat_byte(sender),
                            AccountInfo::default().with_balance(U256::from(if rolling {
                                1_000_000_000_000u64
                            } else {
                                1_000_000_000u64
                            })),
                        );
                    }
                    if mixed {
                        db.insert_account(
                            Address::repeat_byte(0x44),
                            AccountInfo::default()
                                .with_code(Bytecode::new_raw(Bytes::copy_from_slice(&fast_code))),
                        );
                    }
                    if storage {
                        for address in if independent {
                            (0..count)
                                .map(|i| Address::with_last_byte(0x50 + i))
                                .collect::<Vec<_>>()
                        } else {
                            vec![CONTRACT]
                        } {
                            let info = AccountInfo::default()
                                .with_code(Bytecode::new_raw(Bytes::copy_from_slice(code)));
                            let slots: alloy_primitives::map::U256Map<U256> = (0..256)
                                .map(|slot| (U256::from(slot), U256::from(slot + 1)))
                                .collect();
                            if cold {
                                db.cache.accounts.remove(&address);
                                db.database.insert_account_info(address, info);
                                for (key, value) in slots {
                                    db.database
                                        .insert_account_storage(address, key, value)
                                        .unwrap();
                                }
                            } else {
                                db.insert_account_with_storage(address, info, slots);
                            }
                        }
                    }
                    let reader = ParentReader(Arc::new(db.database), Arc::default());
                    let read_count = reader.1.clone();
                    let mut db = State::builder()
                        .with_database(reader.clone())
                        .with_cached_prestate(db.cache)
                        .with_bundle_update()
                        .build();
                    let env = EvmEnv::new(
                        CfgEnv::new_with_spec(OpSpecId::JOVIAN),
                        BlockEnv {
                            timestamp: U256::from(JOVIAN_TIMESTAMP),
                            number: U256::from(1),
                            gas_limit: if rolling { 100_000_000 } else { 10_000_000 },
                            basefee: 2,
                            beneficiary: BENEFICIARY,
                            ..Default::default()
                        },
                    );
                    let candidates = txs
                        .iter()
                        .map(|tx| ParallelCandidate {
                            hash: tx.trie_hash(),
                            transaction: crate::OpTx::from_recovered_tx(tx.inner(), tx.signer()),
                        })
                        .collect();
                    let start = Instant::now();
                    let session = (state_reads == reth_optimism_parallel::StateReads::Auto &&
                        mode != ExecutionMode::Sequential)
                        .then(|| Arc::new(reth_optimism_parallel::SnapshotSession::new(
                            Arc::new(move || Ok(Box::new(reader.clone()) as Box<dyn revm::Database<Error = reth_optimism_parallel::SpeculationError>>)),
                            64 * 1024 * 1024,
                        )));
                    let evm = factory.evm_factory.create_evm(&mut db, env);
                    let mut executor = factory.create_executor_with_snapshot(
                        evm,
                        OpBlockExecutionCtx {
                            post_exec_mode: PostExecMode::Produce,
                            parallel_candidates: candidates,
                            ..Default::default()
                        },
                        session.clone(),
                    );
                    for tx in &txs {
                        executor.execute_transaction(tx).unwrap();
                    }
                    let result = (
                        executor.receipts.clone(),
                        executor.refund_snapshot(),
                        executor.post_exec_entries().to_vec(),
                    );
                    drop(executor);
                    snapshot_bytes = snapshot_bytes
                        .max(session.as_ref().map_or(0, |session| session.estimated_bytes()));
                    db.merge_transitions(BundleRetention::Reverts);
                    let bundle = db.take_bundle();
                    let elapsed = start.elapsed().as_secs_f64();
                    let root_started = Instant::now();
                    let root = state_root(&db);
                    let receipt_root = alloy_consensus::proofs::calculate_receipt_root(&result.0);
                    let root_time = root_started.elapsed().as_secs_f64();
                    let result = (result.0, result.1, result.2, root, receipt_root);
                    provider_reads =
                        provider_reads.max(read_count.load(std::sync::atomic::Ordering::Relaxed));
                    assert!(result.0.iter().all(alloy_consensus::TxReceipt::status));
                    let metrics = snapshotter.snapshot().into_vec();
                    if iteration > 0 {
                        samples.push(elapsed);
                        *timing.entry("synthetic_roots_seconds".to_owned()).or_default() +=
                            root_time;
                        for (key, _, _, value) in metrics {
                            if let DebugValue::Histogram(values) = value {
                                if key.key().name().ends_with("_seconds") {
                                    *timing.entry(key.key().name().to_owned()).or_default() +=
                                        values.iter().map(|value| value.0).sum::<f64>();
                                } else if key.key().name() == "optimism_parallel.snapshot_bytes" {
                                    snapshot_bytes = snapshot_bytes.max(
                                        values
                                            .iter()
                                            .map(|value| value.0 as usize)
                                            .max()
                                            .unwrap_or_default(),
                                    );
                                } else if key.key().name() == "optimism_parallel.snapshot_versions"
                                {
                                    snapshot_versions = snapshot_versions.max(
                                        values.iter().map(|value| value.0).fold(0.0, f64::max),
                                    );
                                }
                            }
                        }
                    }
                    if let Some((expected, state)) = &reference {
                        assert_eq!(&result, expected);
                        assert_eq!(&bundle, state);
                    } else {
                        reference = Some((result, bundle));
                    }
                }
                if !rolling && mode != ExecutionMode::Sequential && !chained {
                    assert_eq!(runtime.statistics().completed, 21 * txs.len() as u64);
                    if mode == ExecutionMode::Parallel && workload != "storage_conflict_cold" {
                        assert_eq!(runtime.statistics().reused, 21 * txs.len() as u64);
                    } else if mode == ExecutionMode::Shadow && workload != "storage_conflict_cold" {
                        assert_eq!(runtime.statistics().shadow_matches, 21 * txs.len() as u64);
                    }
                }
                if mode == ExecutionMode::Parallel && chained {
                    assert_eq!(runtime.statistics().attempted, 0);
                }
                assert_eq!(runtime.statistics().shadow_mismatches, 0);
                if state_reads == reth_optimism_parallel::StateReads::Auto {
                    assert_eq!(runtime.statistics().broker_reads, 0);
                }
                samples.sort_by(f64::total_cmp);
                eprintln!(
                    "workload={workload} mode={mode} reads={state_reads} scheduler={scheduler} workers={workers} txs={count} snapshot_bytes={snapshot_bytes} snapshot_versions={snapshot_versions} provider_reads={provider_reads} median_ms={:.3} p95_ms={:.3} mean_ms={:.3} stats={:?}",
                    samples[10] * 1000.0,
                    samples[18] * 1000.0,
                    samples.iter().sum::<f64>() * 1000.0 / samples.len() as f64,
                    runtime.statistics()
                );
                let mean_ms: BTreeMap<_, _> =
                    timing.into_iter().map(|(key, value)| (key, value * 1000.0 / 20.0)).collect();
                eprintln!(
                    "timings workload={workload} mode={mode} scheduler={scheduler} workers={workers} mean_ms={mean_ms:?}"
                );
            }
        }
    }
}

#[test_case::test_case(true; "refill_uses_committed_state")]
#[test_case::test_case(false; "unpreviewed_selection_runs_canonically")]
fn rolling_executor_commits_while_a_later_transaction_is_running(refill: bool) {
    use std::sync::{Mutex, mpsc};
    let txs = [transfer(1, 0, CONTRACT), transfer(2, 0, CONTRACT), transfer(3, 0, CONTRACT)];
    let order = if refill {
        vec![txs[0].clone(), txs[1].clone(), txs[2].clone()]
    } else {
        vec![txs[0].clone(), txs[2].clone(), txs[1].clone()]
    };
    let reference = run(ParallelExecutionConfig::default(), &order, &[0], None, BENEFICIARY);
    let mut db = database(&[0]);
    let session = independent_session(&db);
    let (entered, started) = mpsc::channel();
    let (release, gate) = mpsc::channel();
    let gate = Mutex::new(gate);
    let started = Mutex::new(started);
    let release_on_commit = release.clone();
    db.set_state_hook(Some(Box::new(move |changes: revm::state::EvmState| {
        if !refill && changes.contains_key(&Address::repeat_byte(3)) {
            release_on_commit.send(()).unwrap();
        }
    })));
    let runtime = Arc::new(
        ParallelRuntime::new(ParallelExecutionConfig {
            mode: ExecutionMode::Parallel,
            scheduler: ExecutionScheduler::Rolling,
            workers: 2,
            max_in_flight: if refill { 2 } else { 3 },
            ..Default::default()
        })
        .unwrap(),
    );
    let factory = OpBlockExecutorFactory::new(
        OpAlloyReceiptBuilder::default(),
        OpChainHardforks::op_mainnet(),
        OpEvmFactory::<crate::OpTx, ObservedRefundPolicy<Policy>>::default(),
    )
    .with_parallel_runtime(runtime.clone());
    let evm = factory.evm_factory.create_evm(
        &mut db,
        EvmEnv::new(
            CfgEnv::new_with_spec(OpSpecId::JOVIAN),
            BlockEnv {
                timestamp: U256::from(JOVIAN_TIMESTAMP),
                number: U256::from(1),
                gas_limit: 10_000_000,
                basefee: 2,
                beneficiary: BENEFICIARY,
                ..Default::default()
            },
        ),
    );
    let mut executor = factory.create_executor_with_snapshot(
        evm,
        OpBlockExecutionCtx {
            post_exec_mode: PostExecMode::Produce,
            parallel_candidates: txs[..if refill { 3 } else { 2 }]
                .iter()
                .map(|tx| ParallelCandidate {
                    hash: tx.trie_hash(),
                    transaction: crate::OpTx::from_recovered_tx(tx.inner(), tx.signer()),
                })
                .collect(),
            ..Default::default()
        },
        Some(session),
    );
    let parallel = executor.parallel.as_mut().unwrap();
    let worker = parallel.worker.clone();
    parallel.worker = Arc::new(move |job, db| {
        match job.0.0.base.caller {
            caller if caller == Address::repeat_byte(1) => {
                started.lock().unwrap().recv_timeout(std::time::Duration::from_secs(10)).unwrap();
            }
            caller if caller == Address::repeat_byte(2) => {
                entered.send(()).unwrap();
                gate.lock().unwrap().recv_timeout(std::time::Duration::from_secs(10)).unwrap();
            }
            _ => {
                assert!(refill, "an unpreviewed selected transaction must run canonically");
                assert_eq!(
                    revm::Database::basic(db, Address::repeat_byte(1))?.unwrap().nonce,
                    1,
                    "replacement workers observe the committed prefix"
                );
                release.send(()).unwrap();
            }
        }
        worker(job, db)
    });
    for tx in &order {
        executor.execute_transaction(tx).unwrap();
    }
    assert_eq!(executor.receipts, reference.receipts);
    assert_eq!(executor.refund_snapshot(), reference.policy);
    assert_eq!(runtime.statistics().broker_reads, 0);
    assert_eq!(runtime.statistics().attempted, if refill { 3 } else { 2 });
    drop(executor);
    assert_eq!(state_root(&db), reference.state_root);
}

#[test_case::test_case(false; "phase_boundary")]
#[test_case::test_case(true; "executor_destruction")]
fn rolling_teardown_failure_disables_the_retained_session(destroy_executor: bool) {
    use std::sync::{Mutex, mpsc};
    struct Reader {
        parent: ParentReader,
        release: Arc<Mutex<mpsc::Receiver<()>>>,
    }
    impl revm::Database for Reader {
        type Error = reth_optimism_parallel::SpeculationError;
        fn basic(&mut self, address: Address) -> Result<Option<AccountInfo>, Self::Error> {
            self.parent.basic(address)
        }
        fn storage(&mut self, address: Address, slot: U256) -> Result<U256, Self::Error> {
            self.parent.storage(address, slot)
        }
        fn code_by_hash(&mut self, hash: B256) -> Result<Bytecode, Self::Error> {
            self.parent.code_by_hash(hash)
        }
        fn block_hash(&mut self, number: u64) -> Result<B256, Self::Error> {
            self.parent.block_hash(number)
        }
    }
    impl Drop for Reader {
        fn drop(&mut self) {
            self.release.lock().unwrap().recv_timeout(std::time::Duration::from_secs(10)).unwrap();
            panic!("reader teardown failed after commit");
        }
    }
    let txs = [transfer(1, 0, CONTRACT), transfer(2, 0, CONTRACT)];
    let mut db = database(&[0]);
    let parent = ParentReader(Arc::new(db.database.clone()), Arc::default());
    let (release, gate) = mpsc::channel();
    let gate = Arc::new(Mutex::new(gate));
    let session = Arc::new(reth_optimism_parallel::SnapshotSession::new(
        Arc::new(move || {
            Ok(Box::new(Reader { parent: parent.clone(), release: gate.clone() })
                as Box<dyn revm::Database<Error = reth_optimism_parallel::SpeculationError>>)
        }),
        64 * 1024 * 1024,
    ));
    let runtime = Arc::new(
        ParallelRuntime::new(ParallelExecutionConfig {
            mode: ExecutionMode::Parallel,
            scheduler: ExecutionScheduler::Rolling,
            workers: 1,
            max_in_flight: 2,
            ..Default::default()
        })
        .unwrap(),
    );
    let factory = OpBlockExecutorFactory::new(
        OpAlloyReceiptBuilder::default(),
        OpChainHardforks::op_mainnet(),
        OpEvmFactory::<crate::OpTx, ObservedRefundPolicy<Policy>>::default(),
    )
    .with_parallel_runtime(runtime);
    let evm = factory.evm_factory.create_evm(
        &mut db,
        EvmEnv::new(
            CfgEnv::new_with_spec(OpSpecId::JOVIAN),
            BlockEnv {
                timestamp: U256::from(JOVIAN_TIMESTAMP),
                number: U256::from(1),
                gas_limit: 10_000_000,
                basefee: 2,
                beneficiary: BENEFICIARY,
                ..Default::default()
            },
        ),
    );
    let mut executor = factory.create_executor_with_snapshot(
        evm,
        OpBlockExecutionCtx {
            post_exec_mode: PostExecMode::Produce,
            parallel_candidates: txs
                .iter()
                .map(|tx| ParallelCandidate {
                    hash: tx.trie_hash(),
                    transaction: crate::OpTx::from_recovered_tx(tx.inner(), tx.signer()),
                })
                .collect(),
            ..Default::default()
        },
        Some(session.clone()),
    );
    executor.execute_transaction(&txs[0]).unwrap();
    assert_eq!(executor.receipts.len(), 1);
    assert!(!session.is_disabled());
    release.send(()).unwrap();
    if !destroy_executor {
        crate::post_exec::PostExecExecutorExt::drain_parallel_work(&mut executor);
        assert!(session.is_disabled(), "draining must include reader teardown failure");
    }
    drop(executor);
    assert!(session.is_disabled(), "the next block/subblock must retain broker fallback");
    assert_eq!(db.cache.accounts[&Address::repeat_byte(1)].account_info().unwrap().nonce, 1);
}

#[test_case::test_case(ExecutionScheduler::Window; "window")]
#[test_case::test_case(ExecutionScheduler::Rolling; "rolling")]
fn evm_read_failure_disables_direct_reads_and_retries_canonically(scheduler: ExecutionScheduler) {
    struct Reader(ParentReader);
    impl revm::Database for Reader {
        type Error = reth_optimism_parallel::SpeculationError;
        fn basic(&mut self, address: Address) -> Result<Option<AccountInfo>, Self::Error> {
            self.0.basic(address)
        }
        fn storage(&mut self, address: Address, slot: U256) -> Result<U256, Self::Error> {
            self.0.storage(address, slot)
        }
        fn code_by_hash(&mut self, hash: B256) -> Result<Bytecode, Self::Error> {
            self.0.code_by_hash(hash)
        }
        fn block_hash(&mut self, _: u64) -> Result<B256, Self::Error> {
            Err(Self::Error::Source("parent hash read failed".into()))
        }
    }
    // BLOCKHASH propagates the provider error through the EVM's normal transaction error path.
    let code = [0x60, 0x00, 0x40, 0x50, 0x00];
    let txs: Vec<_> = (1..=4).map(|sender| transfer(sender, 0, CONTRACT)).collect();
    let reference = run(ParallelExecutionConfig::default(), &txs, &code, None, BENEFICIARY);
    let mut db = database(&code);
    let parent = ParentReader(Arc::new(db.database.clone()), Arc::default());
    let session = Arc::new(reth_optimism_parallel::SnapshotSession::new(
        Arc::new(move || {
            Ok(Box::new(Reader(parent.clone()))
                as Box<dyn revm::Database<Error = reth_optimism_parallel::SpeculationError>>)
        }),
        64 * 1024 * 1024,
    ));
    let runtime = Arc::new(
        ParallelRuntime::new(ParallelExecutionConfig {
            mode: ExecutionMode::Parallel,
            scheduler,
            workers: 1,
            max_in_flight: 2,
            ..Default::default()
        })
        .unwrap(),
    );
    let factory = OpBlockExecutorFactory::new(
        OpAlloyReceiptBuilder::default(),
        OpChainHardforks::op_mainnet(),
        OpEvmFactory::<crate::OpTx, ObservedRefundPolicy<Policy>>::default(),
    )
    .with_parallel_runtime(runtime.clone());
    let evm = factory.evm_factory.create_evm(
        &mut db,
        EvmEnv::new(
            CfgEnv::new_with_spec(OpSpecId::JOVIAN),
            BlockEnv {
                timestamp: U256::from(JOVIAN_TIMESTAMP),
                number: U256::from(1),
                gas_limit: 10_000_000,
                basefee: 2,
                beneficiary: BENEFICIARY,
                ..Default::default()
            },
        ),
    );
    let mut executor = factory.create_executor_with_snapshot(
        evm,
        OpBlockExecutionCtx {
            post_exec_mode: PostExecMode::Produce,
            parallel_candidates: txs
                .iter()
                .map(|tx| ParallelCandidate {
                    hash: tx.trie_hash(),
                    transaction: crate::OpTx::from_recovered_tx(tx.inner(), tx.signer()),
                })
                .collect(),
            ..Default::default()
        },
        Some(session.clone()),
    );
    executor.execute_transaction(&txs[0]).unwrap();
    assert!(session.is_disabled(), "EVM error wrapping must preserve failed source health");
    assert_eq!(runtime.statistics().reused, 0);
    let opens = runtime.statistics().provider_opens;
    assert!(opens > 0);
    for tx in &txs[1..] {
        executor.execute_transaction(tx).unwrap();
    }
    assert_eq!(executor.receipts, reference.receipts);
    assert_eq!(executor.refund_snapshot(), reference.policy);
    assert_eq!(runtime.statistics().provider_opens, opens, "failed sources stay disabled");
    assert!(runtime.statistics().broker_reads > 0, "later windows use the broker");
    drop(executor);
    assert_eq!(state_root(&db), reference.state_root);
    assert!(session.capture(&mut db).is_none(), "retained sessions preserve fallback");
}
