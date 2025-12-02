use std::{
    collections::HashMap,
    path::PathBuf,
    sync::{Arc, RwLock},
};

use crate::{
    CrossChainSafetyProvider, FinalizedL1Storage, HeadRefStorageReader, HeadRefStorageWriter,
    LogStorageReader, Metrics, chaindb::ChainDb, error::StorageError,
};
use alloy_primitives::ChainId;
use kona_interop::DerivedRefPair;
use kona_protocol::BlockInfo;
use kona_supervisor_metrics::{MetricsReporter, observe_metrics_for_result};
use kona_supervisor_types::Log;
use op_alloy_consensus::interop::SafetyLevel;
use tracing::error;

/// Factory for managing multiple chain databases.
/// This struct allows for the creation and retrieval of `ChainDb` instances
/// based on chain IDs, ensuring that each chain has its own database instance.
#[derive(Debug)]
pub struct ChainDbFactory {
    db_path: PathBuf,
    metrics_enabled: Option<bool>,

    dbs: RwLock<HashMap<ChainId, Arc<ChainDb>>>,
    /// Finalized L1 block reference, used for tracking the finalized L1 block.
    /// In-memory only, not persisted.
    finalized_l1: RwLock<Option<BlockInfo>>,
}

impl ChainDbFactory {
    /// Create a new, empty factory.
    pub fn new(db_path: PathBuf) -> Self {
        Self {
            db_path,
            metrics_enabled: None,
            dbs: RwLock::new(HashMap::new()),
            finalized_l1: RwLock::new(None),
        }
    }

    /// Enables metrics on the database environment.
    pub const fn with_metrics(mut self) -> Self {
        self.metrics_enabled = Some(true);
        self
    }

    fn observe_call<T, E, F: FnOnce() -> Result<T, E>>(
        &self,
        name: &'static str,
        f: F,
    ) -> Result<T, E> {
        if self.metrics_enabled.unwrap_or(false) {
            observe_metrics_for_result!(
                Metrics::STORAGE_REQUESTS_SUCCESS_TOTAL,
                Metrics::STORAGE_REQUESTS_ERROR_TOTAL,
                Metrics::STORAGE_REQUEST_DURATION_SECONDS,
                name,
                f()
            )
        } else {
            f()
        }
    }

    /// Get or create a [`ChainDb`] for the given chain id.
    ///
    /// If the database does not exist, it will be created at the path `self.db_path/<chain_id>`.
    pub fn get_or_create_db(&self, chain_id: ChainId) -> Result<Arc<ChainDb>, StorageError> {
        {
            // Try to get it without locking for write
            let dbs = self.dbs.read().map_err(|err| {
                error!(target: "supervisor::storage", %err, "Failed to acquire read lock on databases");
                StorageError::LockPoisoned
            })?;
            if let Some(db) = dbs.get(&chain_id) {
                return Ok(db.clone());
            }
        }

        // Not found, create and insert
        let mut dbs = self.dbs.write().map_err(|err| {
            error!(target: "supervisor::storage", %err, "Failed to acquire write lock on databases");
            StorageError::LockPoisoned
        })?;
        // Double-check in case another thread inserted
        if let Some(db) = dbs.get(&chain_id) {
            return Ok(db.clone());
        }

        let chain_db_path = self.db_path.join(chain_id.to_string());
        let mut chain_db = ChainDb::new(chain_id, chain_db_path.as_path())?;
        if self.metrics_enabled.unwrap_or(false) {
            chain_db = chain_db.with_metrics();
        }
        let db = Arc::new(chain_db);
        dbs.insert(chain_id, db.clone());
        Ok(db)
    }

    /// Get a [`ChainDb`] for the given chain id, returning an error if it doesn't exist.
    ///
    /// # Returns
    /// * `Ok(Arc<ChainDb>)` if the database exists.
    /// * `Err(StorageError)` if the database does not exist.
    pub fn get_db(&self, chain_id: ChainId) -> Result<Arc<ChainDb>, StorageError> {
        let dbs = self.dbs.read().map_err(|_| StorageError::LockPoisoned)?;
        dbs.get(&chain_id).cloned().ok_or_else(|| StorageError::DatabaseNotInitialised)
    }
}

impl MetricsReporter for ChainDbFactory {
    fn report_metrics(&self) {
        let metrics_enabled = self.metrics_enabled.unwrap_or(false);
        if metrics_enabled {
            let dbs: Vec<Arc<ChainDb>> = {
                match self.dbs.read() {
                    Ok(dbs_guard) => dbs_guard.values().cloned().collect(),
                    Err(_) => {
                        error!(target: "supervisor::storage", "Failed to acquire read lock for metrics reporting");
                        return;
                    }
                }
            };
            for db in dbs {
                db.report_metrics();
            }
        }
    }
}

impl FinalizedL1Storage for ChainDbFactory {
    fn get_finalized_l1(&self) -> Result<BlockInfo, StorageError> {
        self.observe_call(
            Metrics::STORAGE_METHOD_GET_FINALIZED_L1,
            || {
                let guard = self.finalized_l1.read().map_err(|err| {
                    error!(target: "supervisor::storage", %err, "Failed to acquire read lock on finalized_l1");
                    StorageError::LockPoisoned
                })?;
                guard.as_ref().cloned().ok_or(StorageError::FutureData)
            }
        )
    }

    fn update_finalized_l1(&self, block: BlockInfo) -> Result<(), StorageError> {
        self.observe_call(
            Metrics::STORAGE_METHOD_UPDATE_FINALIZED_L1,
            || {
                let mut guard = self
                    .finalized_l1
                    .write()
                    .map_err(|err| {
                        error!(target: "supervisor::storage", %err, "Failed to acquire write lock on finalized_l1");
                        StorageError::LockPoisoned
                    })?;

                // Check if the new block number is greater than the current finalized block
                if let Some(ref current) = *guard {
                    if block.number <= current.number {
                        error!(target: "supervisor::storage",
                            current_block_number = current.number,
                            new_block_number = block.number,
                            "New finalized block number is not greater than current finalized block number",
                        );
                        return Err(StorageError::BlockOutOfOrder);
                    }
                }
                *guard = Some(block);
                Ok(())
            }
        )
    }
}

impl CrossChainSafetyProvider for ChainDbFactory {
    fn get_block(&self, chain_id: ChainId, block_number: u64) -> Result<BlockInfo, StorageError> {
        self.get_db(chain_id)?.get_block(block_number)
    }

    fn get_log(
        &self,
        chain_id: ChainId,
        block_number: u64,
        log_index: u32,
    ) -> Result<Log, StorageError> {
        self.get_db(chain_id)?.get_log(block_number, log_index)
    }

    fn get_block_logs(
        &self,
        chain_id: ChainId,
        block_number: u64,
    ) -> Result<Vec<Log>, StorageError> {
        self.get_db(chain_id)?.get_logs(block_number)
    }

    fn get_safety_head_ref(
        &self,
        chain_id: ChainId,
        level: SafetyLevel,
    ) -> Result<BlockInfo, StorageError> {
        self.get_db(chain_id)?.get_safety_head_ref(level)
    }

    fn update_current_cross_unsafe(
        &self,
        chain_id: ChainId,
        block: &BlockInfo,
    ) -> Result<(), StorageError> {
        self.get_db(chain_id)?.update_current_cross_unsafe(block)
    }

    fn update_current_cross_safe(
        &self,
        chain_id: ChainId,
        block: &BlockInfo,
    ) -> Result<DerivedRefPair, StorageError> {
        self.get_db(chain_id)?.update_current_cross_safe(block)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::TempDir;

    fn temp_factory() -> (TempDir, ChainDbFactory) {
        let tmp = TempDir::new().expect("create temp dir");
        let factory = ChainDbFactory::new(tmp.path().to_path_buf());
        (tmp, factory)
    }

    #[test]
    fn test_get_or_create_db_creates_and_returns_db() {
        let (_tmp, factory) = temp_factory();
        let db = factory.get_or_create_db(1).expect("should create db");
        assert!(Arc::strong_count(&db) >= 1);
    }

    #[test]
    fn test_get_or_create_db_returns_same_instance() {
        let (_tmp, factory) = temp_factory();
        let db1 = factory.get_or_create_db(42).unwrap();
        let db2 = factory.get_or_create_db(42).unwrap();
        assert!(Arc::ptr_eq(&db1, &db2));
    }

    #[test]
    fn test_get_db_returns_error_if_not_exists() {
        let (_tmp, factory) = temp_factory();
        let err = factory.get_db(999).unwrap_err();
        assert!(matches!(err, StorageError::DatabaseNotInitialised));
    }

    #[test]
    fn test_get_db_returns_existing_db() {
        let (_tmp, factory) = temp_factory();
        let db = factory.get_or_create_db(7).unwrap();
        let db2 = factory.get_db(7).unwrap();
        assert!(Arc::ptr_eq(&db, &db2));
    }

    #[test]
    fn test_db_path_is_unique_per_chain() {
        let (tmp, factory) = temp_factory();
        let db1 = factory.get_or_create_db(1).unwrap();
        let db2 = factory.get_or_create_db(2).unwrap();
        assert!(!Arc::ptr_eq(&db1, &db2));

        assert!(tmp.path().join("1").exists());
        assert!(tmp.path().join("2").exists());
    }

    #[test]
    fn test_get_finalized_l1_returns_error_when_none() {
        let (_tmp, factory) = temp_factory();
        let err = factory.get_finalized_l1().unwrap_err();
        assert!(matches!(err, StorageError::FutureData));
    }

    #[test]
    fn test_update_and_get_finalized_l1_success() {
        let (_tmp, factory) = temp_factory();
        let block1 = BlockInfo { number: 100, ..Default::default() };
        let block2 = BlockInfo { number: 200, ..Default::default() };

        // Set first finalized block
        factory.update_finalized_l1(block1).unwrap();
        assert_eq!(factory.get_finalized_l1().unwrap(), block1);

        // Update with higher block number
        factory.update_finalized_l1(block2).unwrap();
        assert_eq!(factory.get_finalized_l1().unwrap(), block2);
    }

    #[test]
    fn test_update_finalized_l1_with_lower_block_number_errors() {
        let (_tmp, factory) = temp_factory();
        let block1 = BlockInfo { number: 100, ..Default::default() };
        let block2 = BlockInfo { number: 50, ..Default::default() };

        factory.update_finalized_l1(block1).unwrap();
        let err = factory.update_finalized_l1(block2).unwrap_err();
        assert!(matches!(err, StorageError::BlockOutOfOrder));
    }

    #[test]
    fn test_update_finalized_l1_with_same_block_number_errors() {
        let (_tmp, factory) = temp_factory();
        let block1 = BlockInfo { number: 100, ..Default::default() };
        let block2 = BlockInfo { number: 100, ..Default::default() };

        factory.update_finalized_l1(block1).unwrap();
        let err = factory.update_finalized_l1(block2).unwrap_err();
        assert!(matches!(err, StorageError::BlockOutOfOrder));
    }
}
