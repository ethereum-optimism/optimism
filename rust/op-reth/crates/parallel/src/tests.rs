use super::*;
use revm::{
    database::InMemoryDB,
    primitives::{Address, B256, U256},
    state::{AccountInfo, Bytecode},
};
use std::{cell::Cell, rc::Rc, sync::Barrier};

/// Models reth's thread-affine read transaction. Only the coordinator may access it.
struct ThreadBound {
    database: InMemoryDB,
    reads: Rc<Cell<usize>>,
    thread: std::thread::ThreadId,
}

impl Database for ThreadBound {
    type Error = core::convert::Infallible;
    fn basic(&mut self, address: Address) -> Result<Option<AccountInfo>, Self::Error> {
        assert_eq!(std::thread::current().id(), self.thread);
        self.reads.set(self.reads.get() + 1);
        self.database.basic(address)
    }
    fn storage(&mut self, address: Address, slot: U256) -> Result<U256, Self::Error> {
        self.database.storage(address, slot)
    }
    fn code_by_hash(&mut self, hash: B256) -> Result<Bytecode, Self::Error> {
        self.database.code_by_hash(hash)
    }
    fn block_hash(&mut self, number: u64) -> Result<B256, Self::Error> {
        self.database.block_hash(number)
    }
}

#[test]
fn workers_run_concurrently_and_reads_stay_on_coordinator() {
    let runtime =
        ParallelRuntime::new(ParallelExecutionConfig { workers: 4, ..Default::default() }).unwrap();
    let barrier = Arc::new(Barrier::new(4));
    let reads = Rc::new(Cell::new(0));
    let mut db = ThreadBound {
        database: InMemoryDB::default(),
        reads: reads.clone(),
        thread: std::thread::current().id(),
    };
    let output = runtime.execute(
        &mut db,
        vec![0, 1, 2, 3],
        Arc::new(move |index, db| {
            barrier.wait();
            // Local cache hits must still remain part of the dependency set.
            assert_eq!(db.basic(Address::ZERO)?, None);
            assert_eq!(db.basic(Address::ZERO)?, None);
            for _ in 0..(3 - index) {
                std::thread::yield_now();
            }
            Ok(index)
        }),
    );
    assert_eq!(reads.get(), 4);
    for (index, output) in output.into_iter().enumerate() {
        let (value, dependencies) = output.unwrap();
        assert_eq!(value, index);
        assert!(dependencies.validate(&mut db).unwrap());
        db.database
            .insert_account_info(Address::ZERO, AccountInfo::default().with_balance(U256::from(1)));
        assert!(!dependencies.validate(&mut db).unwrap());
        db.database.cache.accounts.remove(&Address::ZERO);
    }
}

#[test]
fn storage_account_lifecycle_code_and_parent_dependencies() {
    let runtime =
        ParallelRuntime::new(ParallelExecutionConfig { workers: 2, ..Default::default() }).unwrap();
    let mut db = InMemoryDB::default();
    let address = Address::repeat_byte(1);
    let code = Bytecode::new_raw(revm::primitives::bytes!("60005400"));
    let hash = code.hash_slow();
    db.insert_account_info(address, AccountInfo::default().with_code(code));
    db.insert_account_storage(address, U256::ZERO, U256::from(7)).unwrap();
    let (_, reads) = runtime
        .execute(
            &mut db,
            vec![()],
            Arc::new(move |(), db| {
                db.basic(address)?;
                db.storage(address, U256::ZERO)?;
                db.code_by_hash(hash)?;
                db.block_hash(12)?;
                Ok(())
            }),
        )
        .pop()
        .unwrap()
        .unwrap();
    assert!(reads.validate(&mut db).unwrap());
    db.insert_account_storage(address, U256::from(1), U256::from(99)).unwrap();
    assert!(reads.validate(&mut db).unwrap(), "an unrelated slot is not a conflict");
    db.insert_account_storage(address, U256::ZERO, U256::from(8)).unwrap();
    assert!(!reads.validate(&mut db).unwrap());
    db.insert_account_storage(address, U256::ZERO, U256::from(7)).unwrap();
    db.cache.block_hashes.insert(U256::from(12), B256::repeat_byte(9));
    assert!(!reads.validate(&mut db).unwrap(), "parent changes invalidate hash reads");
    db.cache.block_hashes.clear();
    db.cache.accounts.remove(&address);
    assert!(!reads.validate(&mut db).unwrap(), "deletion invalidates account and storage reads");
}

#[test]
fn memory_limits_panics_and_provider_errors_are_speculative_failures() {
    let runtime = ParallelRuntime::new(ParallelExecutionConfig {
        workers: 2,
        max_read_bytes: 1,
        ..Default::default()
    })
    .unwrap();
    let mut db = InMemoryDB::default();
    let results = runtime.execute(
        &mut db,
        vec![false, true],
        Arc::new(|panic, db| {
            assert!(!panic, "deliberate worker panic");
            let _ = db.basic(Address::ZERO); // Even ignored observer errors must abort the attempt.
            Ok(())
        }),
    );
    assert!(matches!(results[0], Err(SpeculationError::Limit)));
    assert!(matches!(results[1], Err(SpeculationError::Worker(_))));
    // The coordinator is reusable after a worker fails.
    assert!(runtime.execute(&mut db, vec![()], Arc::new(|(), _| Ok(())))[0].is_ok());
    assert!(
        ParallelRuntime::new(ParallelExecutionConfig { max_in_flight: 0, ..Default::default() })
            .is_err()
    );
}

#[test]
fn cancellation_discards_completed_results_and_new_parent_is_independent() {
    let runtime =
        ParallelRuntime::new(ParallelExecutionConfig { workers: 2, ..Default::default() }).unwrap();
    let generation = ExecutionGeneration::default();
    let cancel = generation.clone();
    let mut db = InMemoryDB::default();
    let results = runtime.execute_in_generation(
        &mut db,
        vec![0, 1, 2],
        Arc::new(move |index, db| {
            db.basic(Address::ZERO)?;
            if index == 1 {
                cancel.cancel();
            }
            Ok(index)
        }),
        &generation,
    );
    assert!(results.iter().all(|result| matches!(result, Err(SpeculationError::Cancelled))));
    let next_parent = ExecutionGeneration::default();
    assert_ne!(next_parent.id(), generation.id());
    assert!(
        runtime.execute_in_generation(&mut db, vec![()], Arc::new(|(), _| Ok(())), &next_parent)[0]
            .is_ok()
    );
}

struct LocalReader {
    database: InMemoryDB,
    thread: std::thread::ThreadId,
    // A !Send, !Sync field proves readers need neither trait.
    reads: Rc<Cell<usize>>,
    dropped: Arc<AtomicU64>,
    panic_read: bool,
    panic_drop: bool,
}
impl Drop for LocalReader {
    fn drop(&mut self) {
        assert_eq!(self.thread, std::thread::current().id());
        self.dropped.fetch_add(1, Ordering::Relaxed);
        assert!(!self.panic_drop, "test reader teardown panic");
    }
}
impl Database for LocalReader {
    type Error = SpeculationError;
    fn basic(&mut self, address: Address) -> Result<Option<AccountInfo>, Self::Error> {
        assert_eq!(self.thread, std::thread::current().id());
        assert!(!self.panic_read, "test reader read panic");
        self.reads.set(self.reads.get() + 1);
        Ok(self.database.basic(address).unwrap())
    }
    fn storage(&mut self, address: Address, slot: U256) -> Result<U256, Self::Error> {
        assert_eq!(self.thread, std::thread::current().id());
        Ok(self.database.storage(address, slot).unwrap())
    }
    fn code_by_hash(&mut self, hash: B256) -> Result<Bytecode, Self::Error> {
        Ok(self.database.code_by_hash(hash).unwrap())
    }
    fn block_hash(&mut self, number: u64) -> Result<B256, Self::Error> {
        Ok(self.database.block_hash(number).unwrap())
    }
}

#[test]
fn independent_readers_are_worker_local_reused_and_drained() {
    let runtime =
        ParallelRuntime::new(ParallelExecutionConfig { workers: 4, ..Default::default() }).unwrap();
    let dropped = Arc::new(AtomicU64::new(0));
    let counter = dropped.clone();
    let coordinator = std::thread::current().id();
    let session = SnapshotSession::new(
        Arc::new(move || {
            assert_ne!(std::thread::current().id(), coordinator);
            Ok(Box::new(LocalReader {
                database: InMemoryDB::default(),
                thread: std::thread::current().id(),
                reads: Rc::default(),
                dropped: counter.clone(),
                panic_read: false,
                panic_drop: false,
            }) as Box<dyn Database<Error = SpeculationError>>)
        }),
        65536,
    );
    let mut state = revm::database::State::builder().with_database(InMemoryDB::default()).build();
    let snapshot = session.capture(&mut state);
    let output = runtime.execute_with_reads(
        &mut state,
        (0..16).collect(),
        Arc::new(|index, db| {
            db.basic(Address::repeat_byte(index))?;
            db.storage(Address::repeat_byte(index), U256::ZERO)?;
            db.block_hash(1)?;
            Ok(index)
        }),
        &ExecutionGeneration::default(),
        snapshot,
    );
    assert!(output.iter().all(Result::is_ok));
    let statistics = runtime.statistics();
    assert_eq!(statistics.provider_opens, 4);
    assert_eq!(dropped.load(Ordering::Relaxed), 4, "return includes reader teardown");
    assert_eq!(statistics.broker_reads, 0);
    assert_eq!(statistics.direct_reads, 32);
    assert!(state.cache.accounts.is_empty(), "no read messages populated coordinator cache");
    for result in output {
        assert!(result.unwrap().1.validate(&mut state).unwrap());
    }
    assert_eq!(
        state.cache.accounts.len(),
        16,
        "validation still warms canonical state before commit"
    );
}

#[test]
fn independent_reader_panics_errors_and_cancellation_discard_attempts() {
    for failure in 0..4 {
        let runtime =
            ParallelRuntime::new(ParallelExecutionConfig { workers: 1, ..Default::default() })
                .unwrap();
        let session = SnapshotSession::new(
            Arc::new(move || {
                assert_ne!(failure, 0, "test reader initialization panic");
                if failure == 1 {
                    return Err(SpeculationError::Source("test unavailable base".into()));
                }
                Ok(Box::new(LocalReader {
                    database: InMemoryDB::default(),
                    thread: std::thread::current().id(),
                    reads: Rc::default(),
                    dropped: Arc::default(),
                    panic_read: failure == 2,
                    panic_drop: failure == 3,
                }) as Box<dyn Database<Error = SpeculationError>>)
            }),
            65536,
        );
        let mut state =
            revm::database::State::builder().with_database(InMemoryDB::default()).build();
        let reads = session.capture(&mut state);
        let output = runtime.execute_with_reads(
            &mut state,
            vec![(), ()],
            Arc::new(|(), db| {
                db.basic(Address::ZERO)?;
                Ok(())
            }),
            &ExecutionGeneration::default(),
            reads,
        );
        assert!(output.iter().all(|result| matches!(result, Err(SpeculationError::Source(_)))));
        assert_eq!(runtime.statistics().broker_reads, 0);
        session.disable("test_failure");
        assert!(session.capture(&mut state).is_none());
    }
    for cancel_session in [false, true] {
        let runtime =
            ParallelRuntime::new(ParallelExecutionConfig { workers: 1, ..Default::default() })
                .unwrap();
        let session = SnapshotSession::new(
            Arc::new(|| {
                Ok(Box::new(revm::database::EmptyDBTyped::default())
                    as Box<dyn Database<Error = SpeculationError>>)
            }),
            65536,
        );
        let mut state =
            revm::database::State::builder().with_database(InMemoryDB::default()).build();
        let reads = session.capture(&mut state);
        let generation = ExecutionGeneration::default();
        let cancel = if cancel_session { session.generation().clone() } else { generation.clone() };
        let output = runtime.execute_with_reads(
            &mut state,
            vec![(), ()],
            Arc::new(move |(), db| {
                db.basic(Address::ZERO)?;
                cancel.cancel();
                assert!(matches!(db.basic(Address::ZERO), Err(SpeculationError::Cancelled)));
                assert!(matches!(db.block_hash(42), Err(SpeculationError::Cancelled)));
                Ok(())
            }),
            &generation,
            reads,
        );
        assert!(output.iter().all(|result| matches!(result, Err(SpeculationError::Cancelled))));
    }
}

fn direct_test_session(balance: u64, dropped: Arc<AtomicU64>, panic_drop: bool) -> SnapshotSession {
    let coordinator = std::thread::current().id();
    SnapshotSession::new(
        Arc::new(move || {
            assert_ne!(std::thread::current().id(), coordinator);
            let mut database = InMemoryDB::default();
            database.insert_account_info(
                Address::repeat_byte(10),
                AccountInfo::default().with_balance(U256::from(balance)),
            );
            Ok(Box::new(LocalReader {
                database,
                thread: std::thread::current().id(),
                reads: Rc::default(),
                dropped: dropped.clone(),
                panic_read: false,
                panic_drop,
            }) as Box<dyn Database<Error = SpeculationError>>)
        }),
        65536,
    )
}

fn commit_test_balance(state: &mut revm::database::State<InMemoryDB>, balance: u64) {
    let address = Address::repeat_byte(10);
    let mut account = revm::state::Account::from(state.basic(address).unwrap().unwrap());
    account.info.balance = U256::from(balance);
    account.mark_touch();
    revm::DatabaseCommit::commit(state, std::iter::once((address, account)).collect());
}

#[test]
fn rolling_commits_and_refills_while_a_later_reader_is_still_running() {
    let runtime = ParallelRuntime::new(ParallelExecutionConfig {
        workers: 2,
        max_in_flight: 2,
        scheduler: ExecutionScheduler::Rolling,
        ..Default::default()
    })
    .unwrap();
    let dropped = Arc::new(AtomicU64::new(0));
    let session = direct_test_session(1, dropped.clone(), false);
    let mut base = InMemoryDB::default();
    base.insert_account_info(
        Address::repeat_byte(10),
        AccountInfo::default().with_balance(U256::from(1)),
    );
    let mut canonical = revm::database::State::builder().with_database(base).build();
    let initial = session.capture(&mut canonical).unwrap();
    let (entered, started) = mpsc::channel();
    let (release, gate) = mpsc::channel();
    let gate = Mutex::new(gate);
    let worker = Arc::new(move |index: u8, db: &mut SpeculativeDatabase| {
        let balance = db.basic(Address::repeat_byte(10))?.unwrap().balance;
        if index == 1 {
            entered.send(()).unwrap();
            gate.lock().unwrap().recv_timeout(std::time::Duration::from_secs(10)).unwrap();
        }
        Ok(balance)
    });
    let pipeline =
        ExecutionPipeline::new(&runtime, worker, ExecutionGeneration::default(), &initial);
    pipeline
        .submit_batch(vec![
            (B256::repeat_byte(0), 0, initial.clone()),
            (B256::repeat_byte(1), 1, initial.clone()),
        ])
        .unwrap();
    started.recv_timeout(std::time::Duration::from_secs(10)).unwrap();
    let (value, dependencies) = pipeline.take(B256::repeat_byte(0)).unwrap().unwrap();
    assert_eq!(value, U256::from(1));
    assert!(dependencies.validate(&mut canonical).unwrap());
    assert_eq!(pipeline.outstanding(), 2, "prepared output still occupies its slot");
    assert!(matches!(
        pipeline.submit(B256::repeat_byte(2), 2, initial),
        Err(SpeculationError::Limit)
    ));
    commit_test_balance(&mut canonical, 9);
    pipeline.retire(B256::repeat_byte(0));
    let fresh = session.capture(&mut canonical).unwrap();
    pipeline.submit(B256::repeat_byte(2), 2, fresh).unwrap();
    // The tail remains gated. Its replacement sees the newly committed state immediately.
    let (value, reads) = pipeline.take(B256::repeat_byte(2)).unwrap().unwrap();
    assert_eq!(value, U256::from(9));
    assert!(reads.validate(&mut canonical).unwrap());
    pipeline.retire(B256::repeat_byte(2));
    release.send(()).unwrap();
    let (old, reads) = pipeline.take(B256::repeat_byte(1)).unwrap().unwrap();
    assert_eq!(old, U256::from(1));
    assert!(!reads.validate(&mut canonical).unwrap(), "stale tail requires canonical retry");
    pipeline.cancel_and_drain();
    assert_eq!(pipeline.outstanding(), 0);
    assert_eq!(runtime.statistics().broker_reads, 0);
    assert_eq!(runtime.statistics().provider_opens, dropped.load(Ordering::Relaxed));
}

#[test]
fn rolling_late_teardown_failure_preserves_an_already_committed_prefix() {
    let runtime =
        ParallelRuntime::new(ParallelExecutionConfig { workers: 1, ..Default::default() }).unwrap();
    let session = direct_test_session(1, Arc::default(), true);
    let mut base = InMemoryDB::default();
    base.insert_account_info(
        Address::repeat_byte(10),
        AccountInfo::default().with_balance(U256::from(1)),
    );
    let mut canonical = revm::database::State::builder().with_database(base).build();
    let reads = session.capture(&mut canonical).unwrap();
    let (entered, started) = mpsc::channel();
    let (release, gate) = mpsc::channel();
    let gate = Mutex::new(gate);
    let worker = Arc::new(move |index: u8, db: &mut SpeculativeDatabase| {
        let result = db.basic(Address::repeat_byte(10))?;
        if index == 1 {
            entered.send(()).unwrap();
            gate.lock().unwrap().recv_timeout(std::time::Duration::from_secs(10)).unwrap();
        }
        Ok(result)
    });
    let pipeline = ExecutionPipeline::new(&runtime, worker, ExecutionGeneration::default(), &reads);
    pipeline
        .submit_batch(vec![(B256::ZERO, 0, reads.clone()), (B256::repeat_byte(1), 1, reads)])
        .unwrap();
    started.recv_timeout(std::time::Duration::from_secs(10)).unwrap();
    let (_, dependencies) = pipeline.take(B256::ZERO).unwrap().unwrap();
    assert!(dependencies.validate(&mut canonical).unwrap());
    commit_test_balance(&mut canonical, 9);
    pipeline.retire(B256::ZERO);
    release.send(()).unwrap();
    pipeline.cancel_and_drain();
    assert!(pipeline.source_failed());
    assert_eq!(canonical.basic(Address::repeat_byte(10)).unwrap().unwrap().balance, U256::from(9));
    assert!(matches!(pipeline.take(B256::repeat_byte(1)), Some(Err(SpeculationError::Source(_)))));
    assert_eq!(runtime.statistics().broker_reads, 0);
}

#[test]
fn rolling_retired_work_keeps_its_slot_until_execution_stops() {
    let runtime = ParallelRuntime::new(ParallelExecutionConfig {
        workers: 1,
        max_in_flight: 1,
        ..Default::default()
    })
    .unwrap();
    let session = direct_test_session(1, Arc::default(), false);
    let mut canonical =
        revm::database::State::builder().with_database(InMemoryDB::default()).build();
    let reads = session.capture(&mut canonical).unwrap();
    let (entered, started) = mpsc::channel();
    let (release, gate) = mpsc::channel();
    let gate = Mutex::new(gate);
    let worker = Arc::new(move |(): (), db: &mut SpeculativeDatabase| {
        db.basic(Address::repeat_byte(10))?;
        entered.send(()).unwrap();
        gate.lock().unwrap().recv_timeout(std::time::Duration::from_secs(10)).unwrap();
        assert!(
            matches!(db.basic(Address::repeat_byte(10)), Err(SpeculationError::Cancelled)),
            "cached reads check attempt cancellation"
        );
        Ok(())
    });
    let pipeline = ExecutionPipeline::new(&runtime, worker, ExecutionGeneration::default(), &reads);
    pipeline.submit(B256::ZERO, (), reads.clone()).unwrap();
    started.recv_timeout(std::time::Duration::from_secs(10)).unwrap();
    pipeline.retire(B256::ZERO);
    assert_eq!(pipeline.outstanding(), 1);
    assert!(matches!(
        pipeline.submit(B256::repeat_byte(1), (), reads),
        Err(SpeculationError::Limit)
    ));
    assert!(pipeline.take(B256::ZERO).is_none());
    release.send(()).unwrap();
    pipeline.cancel_and_drain();
    assert_eq!(pipeline.outstanding(), 0);
}

#[test]
fn rolling_yields_to_competing_operations_and_honors_the_shared_worker_budget() {
    let runtime =
        ParallelRuntime::new(ParallelExecutionConfig { workers: 1, ..Default::default() }).unwrap();
    let session = direct_test_session(1, Arc::default(), false);
    let mut canonical =
        revm::database::State::builder().with_database(InMemoryDB::default()).build();
    let reads = session.capture(&mut canonical).unwrap();
    let order = Arc::new(Mutex::new(Vec::new()));
    let record = order.clone();
    let (entered, started) = mpsc::channel();
    let (release, gate) = mpsc::channel();
    let gate = Mutex::new(gate);
    let worker = Arc::new(move |index: u8, _: &mut SpeculativeDatabase| {
        record.lock().unwrap().push(index);
        if index == 0 {
            entered.send(()).unwrap();
            gate.lock().unwrap().recv_timeout(std::time::Duration::from_secs(10)).unwrap();
        }
        Ok(index)
    });
    let first =
        ExecutionPipeline::new(&runtime, worker.clone(), ExecutionGeneration::default(), &reads);
    first.submit_batch((0..3).map(|i| (B256::repeat_byte(i), i, reads.clone())).collect()).unwrap();
    started.recv_timeout(std::time::Duration::from_secs(10)).unwrap();
    let second = ExecutionPipeline::new(&runtime, worker, ExecutionGeneration::default(), &reads);
    second.submit(B256::repeat_byte(9), 9, reads).unwrap();
    assert_eq!(
        *order.lock().unwrap(),
        vec![0],
        "the other operation cannot exceed the shared permit budget"
    );
    release.send(()).unwrap();
    assert_eq!(second.take(B256::repeat_byte(9)).unwrap().unwrap().0, 9);
    for i in 0..3 {
        first.take(B256::repeat_byte(i)).unwrap().unwrap();
    }
    first.cancel_and_drain();
    second.cancel_and_drain();
    assert_eq!(*order.lock().unwrap(), vec![0, 9, 1, 2]);
}

#[test]
fn rolling_initialization_read_and_execution_failures_are_non_authoritative() {
    for failure in 0..4 {
        let runtime =
            ParallelRuntime::new(ParallelExecutionConfig { workers: 2, ..Default::default() })
                .unwrap();
        let session = SnapshotSession::new(
            Arc::new(move || {
                assert_ne!(failure, 0, "initialization panic");
                if failure == 1 {
                    return Err(SpeculationError::Source("unavailable".into()));
                }
                Ok(Box::new(LocalReader {
                    database: InMemoryDB::default(),
                    thread: std::thread::current().id(),
                    reads: Rc::default(),
                    dropped: Arc::default(),
                    panic_read: failure == 2,
                    panic_drop: false,
                }) as Box<dyn Database<Error = SpeculationError>>)
            }),
            65536,
        );
        let mut canonical =
            revm::database::State::builder().with_database(InMemoryDB::default()).build();
        let reads = session.capture(&mut canonical).unwrap();
        let pipeline = ExecutionPipeline::new(
            &runtime,
            Arc::new(move |(): (), db: &mut SpeculativeDatabase| {
                assert_ne!(failure, 3, "execution panic");
                let _ = db.basic(Address::ZERO);
                Ok(())
            }),
            ExecutionGeneration::default(),
            &reads,
        );
        pipeline.submit(B256::ZERO, (), reads).unwrap();
        assert!(matches!(pipeline.take(B256::ZERO), Some(Err(SpeculationError::Source(_)))));
        pipeline.cancel_and_drain();
        assert_eq!(pipeline.outstanding(), 0);
        assert_eq!(runtime.statistics().broker_reads, 0);
    }
}

#[test]
fn rolling_dropped_spawned_tasks_still_complete_their_drain() {
    let runtime = ParallelRuntime::with_spawner(
        ParallelExecutionConfig { workers: 1, ..Default::default() },
        1,
        drop,
    )
    .unwrap();
    let session = direct_test_session(1, Arc::default(), false);
    let mut canonical =
        revm::database::State::builder().with_database(InMemoryDB::default()).build();
    let reads = session.capture(&mut canonical).unwrap();
    let pipeline = ExecutionPipeline::new(
        &runtime,
        Arc::new(|(): (), _: &mut SpeculativeDatabase| Ok(())),
        ExecutionGeneration::default(),
        &reads,
    );
    pipeline.submit(B256::ZERO, (), reads).unwrap();
    assert!(matches!(pipeline.take(B256::ZERO), Some(Err(SpeculationError::Source(_)))));
    pipeline.cancel_and_drain();
    assert_eq!(pipeline.outstanding(), 0);
    assert_eq!(runtime.statistics().provider_opens, 0);
}

#[test]
fn provider_errors_survive_worker_wrapping_and_cancellation() {
    struct Reader {
        generation: ExecutionGeneration,
        panic: bool,
    }
    impl Database for Reader {
        type Error = SpeculationError;
        fn basic(&mut self, _: Address) -> Result<Option<AccountInfo>, Self::Error> {
            // A running read can fail after its attempt is retired or its build is cancelled.
            self.generation.cancel();
            assert!(!self.panic, "provider panicked during cancellation");
            Err(SpeculationError::Source("provider failed during cancellation".into()))
        }
        fn storage(&mut self, _: Address, _: U256) -> Result<U256, Self::Error> {
            unreachable!()
        }
        fn code_by_hash(&mut self, _: B256) -> Result<Bytecode, Self::Error> {
            unreachable!()
        }
        fn block_hash(&mut self, _: u64) -> Result<B256, Self::Error> {
            unreachable!()
        }
    }
    for (rolling, panic) in [(false, false), (true, false), (false, true), (true, true)] {
        let runtime =
            ParallelRuntime::new(ParallelExecutionConfig { workers: 1, ..Default::default() })
                .unwrap();
        let generation = ExecutionGeneration::default();
        let cancel = generation.clone();
        let session = SnapshotSession::new(
            Arc::new(move || {
                Ok(Box::new(Reader { generation: cancel.clone(), panic })
                    as Box<dyn Database<Error = SpeculationError>>)
            }),
            65536,
        );
        let mut canonical =
            revm::database::State::builder().with_database(InMemoryDB::default()).build();
        let reads = session.capture(&mut canonical).unwrap();
        let execute: Arc<SpeculativeWorker<(), ()>> = Arc::new(|(), db| {
            db.basic(Address::ZERO).map_err(|error| SpeculationError::Worker(error.to_string()))?;
            Ok(())
        });
        if rolling {
            let pipeline = ExecutionPipeline::new(&runtime, execute, generation, &reads);
            pipeline.submit(B256::ZERO, (), reads).unwrap();
            assert!(pipeline.take(B256::ZERO).is_none_or(|result| result.is_err()));
            pipeline.cancel_and_drain();
            assert!(pipeline.source_failed(), "cancellation must not hide failed source health");
        } else {
            let output = runtime.execute_with_reads(
                &mut canonical,
                vec![()],
                execute,
                &generation,
                Some(reads),
            );
            assert!(matches!(output[0], Err(SpeculationError::Source(_))));
        }
        assert_eq!(runtime.statistics().broker_reads, 0);
    }
}

#[test]
fn rolling_cancels_queued_operations_without_waiting_for_another_build() {
    let runtime =
        ParallelRuntime::new(ParallelExecutionConfig { workers: 1, ..Default::default() }).unwrap();
    let session = direct_test_session(1, Arc::default(), false);
    let mut canonical =
        revm::database::State::builder().with_database(InMemoryDB::default()).build();
    let reads = session.capture(&mut canonical).unwrap();
    let (entered, started) = mpsc::channel();
    let (release, gate) = mpsc::channel();
    let gate = Mutex::new(gate);
    let first = ExecutionPipeline::new(
        &runtime,
        Arc::new(move |(): (), _: &mut SpeculativeDatabase| {
            entered.send(()).unwrap();
            gate.lock().unwrap().recv_timeout(std::time::Duration::from_secs(10)).unwrap();
            Ok(())
        }),
        ExecutionGeneration::default(),
        &reads,
    );
    first.submit(B256::ZERO, (), reads.clone()).unwrap();
    started.recv_timeout(std::time::Duration::from_secs(10)).unwrap();
    let second = ExecutionPipeline::new(
        &runtime,
        Arc::new(|(): (), _: &mut SpeculativeDatabase| -> Result<(), SpeculationError> {
            panic!("cancelled job ran")
        }),
        ExecutionGeneration::default(),
        &reads,
    );
    second.submit(B256::repeat_byte(1), (), reads).unwrap();
    second.cancel_and_drain();
    assert_eq!(second.outstanding(), 0);
    assert!(!second.source_failed());
    release.send(()).unwrap();
    first.take(B256::ZERO).unwrap().unwrap();
    first.cancel_and_drain();
}
