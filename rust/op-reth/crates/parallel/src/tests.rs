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
