//! Main database access structure and transaction contexts.

use crate::{
    Metrics, StorageRewinder,
    error::StorageError,
    providers::{DerivationProvider, LogProvider, SafetyHeadRefProvider},
    traits::{
        DerivationStorageReader, DerivationStorageWriter, HeadRefStorageReader,
        HeadRefStorageWriter, LogStorageReader, LogStorageWriter,
    },
};
use alloy_eips::eip1898::BlockNumHash;
use alloy_primitives::ChainId;
use kona_interop::DerivedRefPair;
use kona_protocol::BlockInfo;
use kona_supervisor_metrics::{MetricsReporter, observe_metrics_for_result};
use kona_supervisor_types::{Log, SuperHead};
use metrics::{Label, gauge};
use op_alloy_consensus::interop::SafetyLevel;
use reth_db::{
    DatabaseEnv,
    mdbx::{DatabaseArguments, init_db_for},
};
use reth_db_api::database::Database;
use std::path::Path;
use tracing::warn;

/// Manages the database environment for a single chain.
/// Provides transactional access to data via providers.
#[derive(Debug)]
pub struct ChainDb {
    chain_id: ChainId,
    metrics_enabled: Option<bool>,

    env: DatabaseEnv,
}

impl ChainDb {
    /// Creates or opens a database environment at the given path.
    pub fn new(chain_id: ChainId, path: &Path) -> Result<Self, StorageError> {
        let env = init_db_for::<_, crate::models::Tables>(path, DatabaseArguments::default())?;
        Ok(Self { chain_id, metrics_enabled: None, env })
    }

    /// Enables metrics on the database environment.
    pub fn with_metrics(mut self) -> Self {
        self.metrics_enabled = Some(true);
        crate::Metrics::init(self.chain_id);
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
                f(),
                "chain_id" => self.chain_id.to_string()
            )
        } else {
            f()
        }
    }
}

// todo: make sure all get method return DatabaseNotInitialised error if db is not initialised
impl DerivationStorageReader for ChainDb {
    fn derived_to_source(&self, derived_block_id: BlockNumHash) -> Result<BlockInfo, StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_DERIVED_TO_SOURCE, || {
            self.env.view(|tx| {
                DerivationProvider::new(tx, self.chain_id).derived_to_source(derived_block_id)
            })
        })?
    }

    fn latest_derived_block_at_source(
        &self,
        source_block_id: BlockNumHash,
    ) -> Result<BlockInfo, StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_LATEST_DERIVED_BLOCK_AT_SOURCE, || {
            self.env.view(|tx| {
                DerivationProvider::new(tx, self.chain_id)
                    .latest_derived_block_at_source(source_block_id)
            })
        })?
    }

    fn latest_derivation_state(&self) -> Result<DerivedRefPair, StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_LATEST_DERIVATION_STATE, || {
            self.env.view(|tx| DerivationProvider::new(tx, self.chain_id).latest_derivation_state())
        })?
    }

    fn get_source_block(&self, source_block_number: u64) -> Result<BlockInfo, StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_GET_SOURCE_BLOCK, || {
            self.env.view(|tx| {
                DerivationProvider::new(tx, self.chain_id).get_source_block(source_block_number)
            })
        })?
    }

    fn get_activation_block(&self) -> Result<BlockInfo, StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_GET_ACTIVATION_BLOCK, || {
            self.env.view(|tx| DerivationProvider::new(tx, self.chain_id).get_activation_block())
        })?
    }
}

impl DerivationStorageWriter for ChainDb {
    fn initialise_derivation_storage(
        &self,
        incoming_pair: DerivedRefPair,
    ) -> Result<(), StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_INITIALISE_DERIVATION_STORAGE, || {
            self.env.update(|ctx| {
                DerivationProvider::new(ctx, self.chain_id).initialise(incoming_pair)?;
                SafetyHeadRefProvider::new(ctx, self.chain_id)
                    .update_safety_head_ref(SafetyLevel::LocalSafe, &incoming_pair.derived)?;
                SafetyHeadRefProvider::new(ctx, self.chain_id)
                    .update_safety_head_ref(SafetyLevel::CrossSafe, &incoming_pair.derived)
            })
        })?
    }

    fn save_derived_block(&self, incoming_pair: DerivedRefPair) -> Result<(), StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_SAVE_DERIVED_BLOCK, || {
            self.env.update(|ctx| {
                DerivationProvider::new(ctx, self.chain_id).save_derived_block(incoming_pair)?;

                // Verify the consistency with log storage.
                // The check is intentionally deferred until after saving the derived block,
                // ensuring validation only triggers on the committed state to prevent false
                // positives.
                // Example: If the parent derived block doesn't exist, it should return error from
                // derivation provider, not from log provider.
                let derived_block = incoming_pair.derived;
                let block = LogProvider::new(ctx, self.chain_id)
                    .get_block(derived_block.number)
                    .map_err(|err| match err {
                        StorageError::EntryNotFound(_) => {
                            warn!(
                                target: "supervisor::storage",
                                incoming_block = %derived_block,
                                "Derived block not found in log storage: {derived_block:?}"
                            );
                            StorageError::FutureData
                        }
                        other => other, // propagate other errors as-is
                    })?;
                if block != derived_block {
                    warn!(
                        target: "supervisor::storage",
                        incoming_block = %derived_block,
                        stored_log_block = %block,
                        "Derived block does not match the stored log block"
                    );
                    return Err(StorageError::ReorgRequired);
                }

                SafetyHeadRefProvider::new(ctx, self.chain_id)
                    .update_safety_head_ref(SafetyLevel::LocalSafe, &incoming_pair.derived)
            })
        })?
    }

    fn save_source_block(&self, incoming_source: BlockInfo) -> Result<(), StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_SAVE_SOURCE_BLOCK, || {
            self.env.update(|ctx| {
                DerivationProvider::new(ctx, self.chain_id).save_source_block(incoming_source)
            })
        })?
    }
}

// todo: make sure all get method return DatabaseNotInitialised error if db is not initialised
impl LogStorageReader for ChainDb {
    fn get_latest_block(&self) -> Result<BlockInfo, StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_GET_LATEST_BLOCK, || {
            self.env.view(|tx| LogProvider::new(tx, self.chain_id).get_latest_block())
        })?
    }

    fn get_block(&self, block_number: u64) -> Result<BlockInfo, StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_GET_BLOCK, || {
            self.env.view(|tx| LogProvider::new(tx, self.chain_id).get_block(block_number))
        })?
    }

    fn get_log(&self, block_number: u64, log_index: u32) -> Result<Log, StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_GET_LOG, || {
            self.env.view(|tx| LogProvider::new(tx, self.chain_id).get_log(block_number, log_index))
        })?
    }

    fn get_logs(&self, block_number: u64) -> Result<Vec<Log>, StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_GET_LOGS, || {
            self.env.view(|tx| LogProvider::new(tx, self.chain_id).get_logs(block_number))
        })?
    }
}

impl LogStorageWriter for ChainDb {
    fn initialise_log_storage(&self, block: BlockInfo) -> Result<(), StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_INITIALISE_LOG_STORAGE, || {
            self.env.update(|ctx| {
                LogProvider::new(ctx, self.chain_id).initialise(block)?;
                SafetyHeadRefProvider::new(ctx, self.chain_id)
                    .update_safety_head_ref(SafetyLevel::LocalUnsafe, &block)?;
                SafetyHeadRefProvider::new(ctx, self.chain_id)
                    .update_safety_head_ref(SafetyLevel::CrossUnsafe, &block)
            })
        })?
    }

    fn store_block_logs(&self, block: &BlockInfo, logs: Vec<Log>) -> Result<(), StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_STORE_BLOCK_LOGS, || {
            self.env.update(|ctx| {
                LogProvider::new(ctx, self.chain_id).store_block_logs(block, logs)?;

                SafetyHeadRefProvider::new(ctx, self.chain_id)
                    .update_safety_head_ref(SafetyLevel::LocalUnsafe, block)
            })
        })?
    }
}

impl HeadRefStorageReader for ChainDb {
    fn get_safety_head_ref(&self, safety_level: SafetyLevel) -> Result<BlockInfo, StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_GET_SAFETY_HEAD_REF, || {
            self.env.view(|tx| {
                SafetyHeadRefProvider::new(tx, self.chain_id).get_safety_head_ref(safety_level)
            })
        })?
    }

    /// Fetches all safety heads and current L1 state
    fn get_super_head(&self) -> Result<SuperHead, StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_GET_SUPER_HEAD, || {
            self.env.view(|tx| {
                let sp = SafetyHeadRefProvider::new(tx, self.chain_id);
                let local_unsafe =
                    sp.get_safety_head_ref(SafetyLevel::LocalUnsafe).map_err(|err| {
                        if matches!(err, StorageError::FutureData) {
                            StorageError::DatabaseNotInitialised
                        } else {
                            err
                        }
                    })?;

                let cross_unsafe = match sp.get_safety_head_ref(SafetyLevel::CrossUnsafe) {
                    Ok(block) => Some(block),
                    Err(StorageError::FutureData) => None,
                    Err(err) => return Err(err),
                };

                let local_safe = match sp.get_safety_head_ref(SafetyLevel::LocalSafe) {
                    Ok(block) => Some(block),
                    Err(StorageError::FutureData) => None,
                    Err(err) => return Err(err),
                };

                let cross_safe = match sp.get_safety_head_ref(SafetyLevel::CrossSafe) {
                    Ok(block) => Some(block),
                    Err(StorageError::FutureData) => None,
                    Err(err) => return Err(err),
                };

                let finalized = match sp.get_safety_head_ref(SafetyLevel::Finalized) {
                    Ok(block) => Some(block),
                    Err(StorageError::FutureData) => None,
                    Err(err) => return Err(err),
                };

                let l1_source =
                    match DerivationProvider::new(tx, self.chain_id).latest_derivation_state() {
                        Ok(pair) => Some(pair.source),
                        Err(StorageError::DatabaseNotInitialised) => None,
                        Err(err) => return Err(err),
                    };

                Ok(SuperHead {
                    l1_source,
                    local_unsafe,
                    cross_unsafe,
                    local_safe,
                    cross_safe,
                    finalized,
                })
            })?
        })
    }
}

impl HeadRefStorageWriter for ChainDb {
    fn update_finalized_using_source(
        &self,
        finalized_source_block: BlockInfo,
    ) -> Result<BlockInfo, StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_UPDATE_FINALIZED_USING_SOURCE, || {
            self.env.update(|tx| {
                let sp = SafetyHeadRefProvider::new(tx, self.chain_id);
                let safe = sp.get_safety_head_ref(SafetyLevel::CrossSafe)?;

                let dp = DerivationProvider::new(tx, self.chain_id);
                let safe_block_pair = dp.get_derived_block_pair(safe.id())?;

                if finalized_source_block.number >= safe_block_pair.source.number {
                    // this could happen during initial sync
                    warn!(
                        target: "supervisor::storage",
                        chain_id = %self.chain_id,
                        l1_finalized_block_number = finalized_source_block.number,
                        safe_source_block_number = safe_block_pair.source.number,
                        "L1 finalized block is greater than safe block",
                    );
                    sp.update_safety_head_ref(SafetyLevel::Finalized, &safe)?;
                    return Ok(safe);
                }

                let latest_derived =
                    dp.latest_derived_block_at_source(finalized_source_block.id())?;
                sp.update_safety_head_ref(SafetyLevel::Finalized, &latest_derived)?;
                Ok(latest_derived)
            })
        })?
    }

    fn update_current_cross_unsafe(&self, block: &BlockInfo) -> Result<(), StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_UPDATE_CURRENT_CROSS_UNSAFE, || {
            self.env.update(|tx| {
                let lp = LogProvider::new(tx, self.chain_id);
                let sp = SafetyHeadRefProvider::new(tx, self.chain_id);

                // Check parent-child relationship with current CrossUnsafe head, if it exists.
                let parent = sp.get_safety_head_ref(SafetyLevel::CrossUnsafe)?;
                if !parent.is_parent_of(block) {
                    warn!(
                        target: "supervisor::storage",
                        chain_id = %self.chain_id,
                        incoming_block = %block,
                        latest_block = %parent,
                        "Incoming block is not the child of the current cross-unsafe head",
                    );
                    return Err(StorageError::ConflictError);
                }

                // Ensure the block exists in log storage and hasn't been pruned due to a re-org.
                let stored_block = lp.get_block(block.number)?;
                if stored_block.hash != block.hash {
                    warn!(
                        target: "supervisor::storage",
                        chain_id = %self.chain_id,
                        incoming_block_hash = %block.hash,
                        stored_block_hash = %stored_block.hash,
                        "Hash mismatch while updating CrossUnsafe head",
                    );
                    return Err(StorageError::ConflictError);
                }

                sp.update_safety_head_ref(SafetyLevel::CrossUnsafe, block)?;
                Ok(())
            })?
        })
    }

    fn update_current_cross_safe(&self, block: &BlockInfo) -> Result<DerivedRefPair, StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_UPDATE_CURRENT_CROSS_SAFE, || {
            self.env.update(|tx| {
                let dp = DerivationProvider::new(tx, self.chain_id);
                let sp = SafetyHeadRefProvider::new(tx, self.chain_id);

                // Check parent-child relationship with current CrossUnsafe head, if it exists.
                let parent = sp.get_safety_head_ref(SafetyLevel::CrossSafe)?;
                if !parent.is_parent_of(block) {
                    warn!(
                        target: "supervisor::storage",
                        chain_id = %self.chain_id,
                        incoming_block = %block,
                        latest_block = %parent,
                        "Incoming block is not the child of the current cross-safe head",
                    );
                    return Err(StorageError::ConflictError);
                }

                // Ensure the block exists in derivation storage and hasn't been pruned due to a
                // re-org.
                let derived_pair = dp.get_derived_block_pair(block.id())?;
                sp.update_safety_head_ref(SafetyLevel::CrossSafe, block)?;

                Ok(derived_pair.into())
            })?
        })
    }
}

impl StorageRewinder for ChainDb {
    fn rewind_log_storage(&self, to: &BlockNumHash) -> Result<(), StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_REWIND_LOG_STORAGE, || {
            self.env.update(|tx| {
                let lp = LogProvider::new(tx, self.chain_id);
                let hp = SafetyHeadRefProvider::new(tx, self.chain_id);

                // Ensure we don't rewind to or before the LocalSafe head.
                match hp.get_safety_head_ref(SafetyLevel::LocalSafe) {
                    Ok(local_safe) => {
                        // If the target block is less than or equal to the local safe head,
                        // we cannot rewind to it, as this would mean losing logs for the safe
                        // blocks. The check is inclusive since the rewind
                        // operation removes the target block as well.
                        if to.number <= local_safe.number {
                            return Err(StorageError::RewindBeyondLocalSafeHead {
                                to: to.number,
                                local_safe: local_safe.number,
                            });
                        }
                    }
                    Err(StorageError::FutureData) => {
                        // If LocalSafe is not set, we can rewind to any point.
                    }
                    Err(err) => return Err(err),
                }

                lp.rewind_to(to)?;

                // get the current latest block to update the safety head refs
                match lp.get_latest_block() {
                    Ok(latest_block) => {
                        hp.reset_safety_head_ref_if_ahead(SafetyLevel::LocalUnsafe, &latest_block)?;
                        hp.reset_safety_head_ref_if_ahead(SafetyLevel::CrossUnsafe, &latest_block)?;
                    }
                    Err(StorageError::DatabaseNotInitialised) => {
                        // If the database returns DatabaseNotInitialised, it means we have rewound
                        // past the activation block
                        hp.remove_safety_head_ref(SafetyLevel::LocalUnsafe)?;
                        hp.remove_safety_head_ref(SafetyLevel::CrossUnsafe)?;
                    }
                    Err(err) => return Err(err),
                };
                Ok(())
            })?
        })
    }

    fn rewind(&self, to: &BlockNumHash) -> Result<(), StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_REWIND, || {
            self.env.update(|tx| {
                let lp = LogProvider::new(tx, self.chain_id);
                let dp = DerivationProvider::new(tx, self.chain_id);
                let hp = SafetyHeadRefProvider::new(tx, self.chain_id);

                lp.rewind_to(to)?;
                dp.rewind_to(to)?;

                // get the current latest block to update the safety head refs
                match lp.get_latest_block() {
                    Ok(latest_block) => {
                        hp.reset_safety_head_ref_if_ahead(SafetyLevel::LocalUnsafe, &latest_block)?;
                        hp.reset_safety_head_ref_if_ahead(SafetyLevel::CrossUnsafe, &latest_block)?;
                        hp.reset_safety_head_ref_if_ahead(SafetyLevel::LocalSafe, &latest_block)?;
                        hp.reset_safety_head_ref_if_ahead(SafetyLevel::CrossSafe, &latest_block)?;
                        hp.reset_safety_head_ref_if_ahead(SafetyLevel::Finalized, &latest_block)?;
                    }
                    Err(StorageError::DatabaseNotInitialised) => {
                        // If the database returns DatabaseNotInitialised, it means we have rewound
                        // past the activation block
                        hp.remove_safety_head_ref(SafetyLevel::LocalUnsafe)?;
                        hp.remove_safety_head_ref(SafetyLevel::CrossUnsafe)?;
                        hp.remove_safety_head_ref(SafetyLevel::LocalSafe)?;
                        hp.remove_safety_head_ref(SafetyLevel::CrossSafe)?;
                        hp.remove_safety_head_ref(SafetyLevel::Finalized)?;
                    }
                    Err(err) => return Err(err),
                }
                Ok(())
            })?
        })
    }

    fn rewind_to_source(&self, to: &BlockNumHash) -> Result<Option<BlockInfo>, StorageError> {
        self.observe_call(Metrics::STORAGE_METHOD_REWIND_TO_SOURCE, || {
            self.env.update(|tx| {
                let lp = LogProvider::new(tx, self.chain_id);
                let dp = DerivationProvider::new(tx, self.chain_id);
                let hp = SafetyHeadRefProvider::new(tx, self.chain_id);

                let derived_target_block = dp.rewind_to_source(to)?;
                if let Some(rewind_target) = derived_target_block {
                    lp.rewind_to(&rewind_target.id())?;
                }

                // get the current latest block to update the safety head refs
                match lp.get_latest_block() {
                    Ok(latest_block) => {
                        hp.reset_safety_head_ref_if_ahead(SafetyLevel::LocalUnsafe, &latest_block)?;
                        hp.reset_safety_head_ref_if_ahead(SafetyLevel::CrossUnsafe, &latest_block)?;
                        hp.reset_safety_head_ref_if_ahead(SafetyLevel::LocalSafe, &latest_block)?;
                        hp.reset_safety_head_ref_if_ahead(SafetyLevel::CrossSafe, &latest_block)?;
                        hp.reset_safety_head_ref_if_ahead(SafetyLevel::Finalized, &latest_block)?;
                    }
                    Err(StorageError::DatabaseNotInitialised) => {
                        // If the database returns DatabaseNotInitialised, it means we have rewound
                        // past the activation block
                        hp.remove_safety_head_ref(SafetyLevel::LocalUnsafe)?;
                        hp.remove_safety_head_ref(SafetyLevel::CrossUnsafe)?;
                        hp.remove_safety_head_ref(SafetyLevel::LocalSafe)?;
                        hp.remove_safety_head_ref(SafetyLevel::CrossSafe)?;
                        hp.remove_safety_head_ref(SafetyLevel::Finalized)?;
                    }
                    Err(err) => return Err(err),
                }
                Ok(derived_target_block)
            })?
        })
    }
}

impl MetricsReporter for ChainDb {
    fn report_metrics(&self) {
        let mut metrics = Vec::new();

        let _ = self
            .env
            .view(|tx| {
                for table in crate::models::Tables::ALL.iter().map(crate::models::Tables::name) {
                    let table_db = tx.inner.open_db(Some(table))?;

                    let stats = tx.inner.db_stat(&table_db)?;

                    let page_size = stats.page_size() as usize;
                    let leaf_pages = stats.leaf_pages();
                    let branch_pages = stats.branch_pages();
                    let overflow_pages = stats.overflow_pages();
                    let num_pages = leaf_pages + branch_pages + overflow_pages;
                    let table_size = page_size * num_pages;
                    let entries = stats.entries();

                    metrics.push((
                        "kona_supervisor_storage.table_size",
                        table_size as f64,
                        vec![
                            Label::new("table", table),
                            Label::new("chain_id", self.chain_id.to_string()),
                        ],
                    ));
                    metrics.push((
                        "kona_supervisor_storage.table_pages",
                        leaf_pages as f64,
                        vec![
                            Label::new("table", table),
                            Label::new("type", "leaf"),
                            Label::new("chain_id", self.chain_id.to_string()),
                        ],
                    ));
                    metrics.push((
                        "kona_supervisor_storage.table_pages",
                        branch_pages as f64,
                        vec![
                            Label::new("table", table),
                            Label::new("type", "branch"),
                            Label::new("chain_id", self.chain_id.to_string()),
                        ],
                    ));
                    metrics.push((
                        "kona_supervisor_storage.table_pages",
                        overflow_pages as f64,
                        vec![
                            Label::new("table", table),
                            Label::new("type", "overflow"),
                            Label::new("chain_id", self.chain_id.to_string()),
                        ],
                    ));
                    metrics.push((
                        "kona_supervisor_storage.table_entries",
                        entries as f64,
                        vec![
                            Label::new("table", table),
                            Label::new("chain_id", self.chain_id.to_string()),
                        ],
                    ));
                }

                Ok::<(), eyre::Report>(())
            })
            .inspect_err(|err| {
                warn!(target: "supervisor::storage", %err, "Failed to collect database metrics");
            });

        for (name, value, labels) in metrics {
            gauge!(name, labels).set(value);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::B256;
    use kona_supervisor_types::Log;
    use tempfile::TempDir;

    #[test]
    fn test_create_and_open_db() {
        let tmp_dir = TempDir::new().expect("create temp dir");
        let db_path = tmp_dir.path().join("chaindb");
        let db = ChainDb::new(1, &db_path);
        assert!(db.is_ok(), "Should create or open database");
    }

    #[test]
    fn test_log_storage() {
        let tmp_dir = TempDir::new().expect("create temp dir");
        let db_path = tmp_dir.path().join("chaindb_logs");
        let db = ChainDb::new(1, &db_path).expect("create db");

        let anchor = DerivedRefPair {
            source: BlockInfo {
                hash: B256::from([0u8; 32]),
                number: 100,
                parent_hash: B256::from([1u8; 32]),
                timestamp: 0,
            },
            derived: BlockInfo {
                hash: B256::from([2u8; 32]),
                number: 0,
                parent_hash: B256::from([3u8; 32]),
                timestamp: 0,
            },
        };

        db.initialise_log_storage(anchor.derived).expect("initialise log storage");
        db.initialise_derivation_storage(anchor).expect("initialise derivation storage");

        let block = BlockInfo {
            hash: B256::from([4u8; 32]),
            number: 1,
            parent_hash: anchor.derived.hash,
            timestamp: 0,
        };
        let log1 = Log { index: 0, hash: B256::from([0u8; 32]), executing_message: None };
        let log2 = Log { index: 1, hash: B256::from([1u8; 32]), executing_message: None };
        let logs = vec![log1, log2];

        // Store logs
        db.store_block_logs(&block, logs.clone()).expect("store logs");

        // Retrieve logs
        let retrieved_logs = db.get_logs(block.number).expect("get logs");
        assert_eq!(retrieved_logs.len(), 2);
        assert_eq!(retrieved_logs, logs, "First log should match stored log");

        let latest_block = db.get_latest_block().expect("latest block");
        assert_eq!(latest_block, block, "Latest block should match stored block");

        let log = db.get_log(block.number, 1).expect("get block by log");
        assert_eq!(log, logs[1], "Block by log should match stored block");
    }

    #[test]
    fn test_super_head_empty() {
        let tmp_dir = TempDir::new().expect("create temp dir");
        let db_path = tmp_dir.path().join("chaindb_super_head_empty");
        let db = ChainDb::new(1, &db_path).expect("create db");

        // Get super head when no blocks are stored
        let err = db.get_super_head().unwrap_err();
        assert!(matches!(err, StorageError::DatabaseNotInitialised));
    }

    #[test]
    fn test_get_super_head_populated() {
        let tmp_dir = tempfile::TempDir::new().unwrap();
        let db_path = tmp_dir.path().join("chaindb");
        let db = ChainDb::new(1, &db_path).unwrap();

        // Prepare blocks
        let block = BlockInfo { number: 1, ..Default::default() };
        let derived_pair = DerivedRefPair { source: block, derived: block };

        // Initialise all heads
        db.initialise_log_storage(block).unwrap();
        db.initialise_derivation_storage(derived_pair).unwrap();

        let _ = db
            .env
            .update(|ctx| {
                let sp = SafetyHeadRefProvider::new(ctx, 1);
                sp.update_safety_head_ref(SafetyLevel::Finalized, &block)
            })
            .unwrap();

        // Should not error and all heads should be Some
        let super_head = db.get_super_head().unwrap();
        assert_eq!(super_head.local_unsafe, block);
        assert!(super_head.cross_unsafe.is_some());
        assert!(super_head.local_safe.is_some());
        assert!(super_head.cross_safe.is_some());
        assert!(super_head.finalized.is_some());
        assert!(super_head.l1_source.is_some());
    }

    #[test]
    fn test_get_super_head_with_some_missing_heads() {
        let tmp_dir = tempfile::TempDir::new().unwrap();
        let db_path = tmp_dir.path().join("chaindb");
        let db = ChainDb::new(1, &db_path).unwrap();

        // Only initialise log storage (not derivation storage)
        let block = BlockInfo { number: 1, ..Default::default() };
        db.initialise_log_storage(block).unwrap();

        let super_head = db.get_super_head().unwrap();
        assert_eq!(super_head.local_unsafe, block);
        // These will be None because derivation storage was not initialised
        assert!(super_head.local_safe.is_none());
        assert!(super_head.cross_safe.is_none());
        assert!(super_head.finalized.is_none());
        assert!(super_head.l1_source.is_none());
    }

    #[test]
    fn test_latest_derivation_state_empty() {
        let tmp_dir = TempDir::new().expect("create temp dir");
        let db_path = tmp_dir.path().join("chaindb_latest_derivation_empty");
        let db = ChainDb::new(1, &db_path).expect("create db");

        // Get latest derivation state when no blocks are stored
        let err = db.latest_derivation_state().unwrap_err();
        assert!(matches!(err, StorageError::DatabaseNotInitialised));
    }

    #[test]
    fn test_get_latest_block_empty() {
        let tmp_dir = TempDir::new().expect("create temp dir");
        let db_path = tmp_dir.path().join("chaindb_latest_block_empty");
        let db = ChainDb::new(1, &db_path).expect("create db");

        // Get latest block when no blocks are stored
        let err = db.get_latest_block().unwrap_err();
        assert!(matches!(err, StorageError::DatabaseNotInitialised));
    }

    #[test]
    fn test_derivation_storage() {
        let tmp_dir = TempDir::new().expect("create temp dir");
        let db_path = tmp_dir.path().join("chaindb_derivation");
        let db = ChainDb::new(1, &db_path).expect("create db");

        let anchor = DerivedRefPair {
            source: BlockInfo {
                hash: B256::from([0u8; 32]),
                number: 100,
                parent_hash: B256::from([1u8; 32]),
                timestamp: 0,
            },
            derived: BlockInfo {
                hash: B256::from([2u8; 32]),
                number: 0,
                parent_hash: B256::from([3u8; 32]),
                timestamp: 0,
            },
        };

        // Create dummy derived block pair
        let derived_pair = DerivedRefPair {
            source: BlockInfo {
                hash: B256::from([4u8; 32]),
                number: 101,
                parent_hash: anchor.source.hash,
                timestamp: 0,
            },
            derived: BlockInfo {
                hash: B256::from([6u8; 32]),
                number: 1,
                parent_hash: anchor.derived.hash,
                timestamp: 0,
            },
        };

        // Initialise the database with the anchor derived block pair
        db.initialise_log_storage(anchor.derived).expect("initialise log storage");
        db.initialise_derivation_storage(anchor).expect("initialise derivation storage");

        // Save derived block pair - should error BlockOutOfOrder error
        let err = db.save_derived_block(derived_pair).unwrap_err();
        assert!(matches!(err, StorageError::BlockOutOfOrder));

        db.store_block_logs(
            &BlockInfo {
                hash: B256::from([6u8; 32]),
                number: 1,
                parent_hash: anchor.derived.hash,
                timestamp: 0,
            },
            vec![],
        )
        .expect("storing logs failed");

        // Save derived block pair
        db.save_source_block(derived_pair.source).expect("save source block");
        db.save_derived_block(derived_pair).expect("save derived pair");

        // Retrieve latest derived block pair
        let latest_pair = db.latest_derivation_state().expect("get latest derived pair");
        assert_eq!(latest_pair, derived_pair, "Latest derived pair should match saved pair");

        // Retrieve derived to source mapping
        let derived_block_id =
            BlockNumHash::new(derived_pair.derived.number, derived_pair.derived.hash);
        let source_block = db.derived_to_source(derived_block_id).expect("get derived to source");
        assert_eq!(
            source_block, derived_pair.source,
            "Source block should match derived pair source"
        );

        // Retrieve latest derived block at source
        let source_block_id =
            BlockNumHash::new(derived_pair.source.number, derived_pair.source.hash);
        let latest_derived = db
            .latest_derived_block_at_source(source_block_id)
            .expect("get latest derived at source");
        assert_eq!(
            latest_derived, derived_pair.derived,
            "Latest derived block at source should match derived pair derived"
        );
    }

    #[test]
    fn test_update_current_cross_unsafe() {
        let tmp_dir = tempfile::TempDir::new().unwrap();
        let db_path = tmp_dir.path().join("chaindb");
        let db = ChainDb::new(1, &db_path).unwrap();

        let source = BlockInfo { number: 1, ..Default::default() };
        let block1 = BlockInfo {
            number: 10,
            hash: B256::random(),
            parent_hash: B256::random(),
            timestamp: 1,
        };
        let mut block2 = BlockInfo {
            number: 11,
            hash: B256::random(),
            parent_hash: B256::random(),
            timestamp: 1,
        };

        db.initialise_log_storage(block1).expect("initialise log storage");
        db.initialise_derivation_storage(DerivedRefPair { source, derived: block1 })
            .expect("initialise derivation storage");

        // should error as block2 must be child of block1
        let err = db.update_current_cross_unsafe(&block2).expect_err("should return an error");
        assert!(matches!(err, StorageError::ConflictError));

        // make block2 as child of block1
        block2.parent_hash = block1.hash;

        // block2 doesn't exist in log storage - should return not found error
        let err = db.update_current_cross_unsafe(&block2).expect_err("should return an error");
        assert!(matches!(err, StorageError::EntryNotFound(_)));

        db.store_block_logs(&block2, vec![]).unwrap();
        db.update_current_cross_unsafe(&block2).unwrap();

        let cross_unsafe_block = db.get_safety_head_ref(SafetyLevel::CrossUnsafe).unwrap();
        assert_eq!(cross_unsafe_block, block2);
    }

    #[test]
    fn test_update_current_cross_safe() {
        let tmp_dir = tempfile::TempDir::new().unwrap();
        let db_path = tmp_dir.path().join("chaindb");
        let db = ChainDb::new(1, &db_path).unwrap();

        let source = BlockInfo { number: 1, ..Default::default() };
        let block1 = BlockInfo {
            number: 10,
            hash: B256::random(),
            parent_hash: B256::random(),
            timestamp: 1,
        };
        let mut block2 = BlockInfo {
            number: 11,
            hash: B256::random(),
            parent_hash: B256::random(),
            timestamp: 1,
        };

        db.initialise_log_storage(block1).expect("initialise log storage");
        db.initialise_derivation_storage(DerivedRefPair { source, derived: block1 })
            .expect("initialise derivation storage");

        // should error as block2 must be child of block1
        let err = db.update_current_cross_safe(&block2).expect_err("should return an error");
        assert!(matches!(err, StorageError::ConflictError));

        // make block2 as child of block1
        block2.parent_hash = block1.hash;

        // block2 doesn't exist in derivation storage - should return not found error
        let err = db.update_current_cross_safe(&block2).expect_err("should return an error");
        assert!(matches!(err, StorageError::EntryNotFound(_)));

        db.store_block_logs(&block2, vec![]).unwrap();
        db.save_derived_block(DerivedRefPair { source, derived: block2 }).unwrap();

        let ref_pair = db.update_current_cross_safe(&block2).unwrap();
        assert_eq!(ref_pair.source, source);
        assert_eq!(ref_pair.derived, block2);

        let cross_safe_block = db.get_safety_head_ref(SafetyLevel::CrossSafe).unwrap();
        assert_eq!(cross_safe_block, block2);
    }

    #[test]
    fn test_source_block_storage() {
        let tmp_dir = TempDir::new().expect("create temp dir");
        let db_path = tmp_dir.path().join("chaindb_source_block");
        let db = ChainDb::new(1, &db_path).expect("create db");

        let source1 = BlockInfo {
            hash: B256::from([0u8; 32]),
            number: 100,
            parent_hash: B256::from([1u8; 32]),
            timestamp: 1234,
        };
        let source2 = BlockInfo {
            hash: B256::from([2u8; 32]),
            number: 101,
            parent_hash: source1.hash,
            timestamp: 5678,
        };
        let derived1 = BlockInfo {
            hash: B256::from([3u8; 32]),
            number: 1,
            parent_hash: source1.hash,
            timestamp: 9101,
        };

        db.initialise_log_storage(derived1).expect("initialise log storage");
        db.initialise_derivation_storage(DerivedRefPair { source: source1, derived: derived1 })
            .expect("initialise derivation storage");

        assert!(db.save_source_block(source2).is_ok());

        // Retrieve latest source block
        let latest = db.latest_derivation_state().expect("get latest source block");
        assert_eq!(latest.source, source2);
    }

    #[test]
    fn test_all_safe_derived() {
        let tmp_dir = TempDir::new().expect("create temp dir");
        let db_path = tmp_dir.path().join("chaindb_source_block");
        let db = ChainDb::new(1, &db_path).expect("create db");

        let anchor = DerivedRefPair {
            source: BlockInfo {
                hash: B256::from([0u8; 32]),
                number: 100,
                parent_hash: B256::from([1u8; 32]),
                timestamp: 1234,
            },
            derived: BlockInfo {
                hash: B256::from([1u8; 32]),
                number: 1,
                parent_hash: B256::from([2u8; 32]),
                timestamp: 1234,
            },
        };

        db.initialise_log_storage(anchor.derived).expect("initialise log storage");
        db.initialise_derivation_storage(anchor).expect("initialise derivation storage");

        let source1 = BlockInfo {
            hash: B256::from([2u8; 32]),
            number: 101,
            parent_hash: anchor.source.hash,
            timestamp: 1234,
        };
        let source2 = BlockInfo {
            hash: B256::from([3u8; 32]),
            number: 102,
            parent_hash: source1.hash,
            timestamp: 1234,
        };
        let derived1 = BlockInfo {
            hash: B256::from([4u8; 32]),
            number: 2,
            parent_hash: anchor.derived.hash,
            timestamp: 1234,
        };
        let derived2 = BlockInfo {
            hash: B256::from([5u8; 32]),
            number: 3,
            parent_hash: derived1.hash,
            timestamp: 1234,
        };
        let derived3 = BlockInfo {
            hash: B256::from([7u8; 32]),
            number: 4,
            parent_hash: derived2.hash,
            timestamp: 1234,
        };

        assert!(db.save_source_block(source1).is_ok());
        db.store_block_logs(&derived1, vec![]).expect("storing logs failed");
        db.store_block_logs(&derived2, vec![]).expect("storing logs failed");
        db.store_block_logs(&derived3, vec![]).expect("storing logs failed");

        assert!(
            db.save_derived_block(DerivedRefPair { source: source1, derived: derived1 }).is_ok()
        );

        assert!(db.save_source_block(source2).is_ok());
        assert!(
            db.save_derived_block(DerivedRefPair { source: source2, derived: derived2 }).is_ok()
        );
        assert!(
            db.save_derived_block(DerivedRefPair { source: source2, derived: derived3 }).is_ok()
        );

        let safe_derived = db.latest_derived_block_at_source(source1.id()).expect("should exist");
        assert_eq!(safe_derived, derived1);

        let safe_derived = db.latest_derived_block_at_source(source2.id()).expect("should exist");
        assert_eq!(safe_derived, derived3);

        let source = db.derived_to_source(derived2.id()).expect("should exist");
        assert_eq!(source, source2);

        let source = db.derived_to_source(derived3.id()).expect("should exist");
        assert_eq!(source, source2);

        let latest_derived_pair = db.latest_derivation_state().expect("should exist");
        assert_eq!(latest_derived_pair, DerivedRefPair { source: source2, derived: derived3 });
    }

    #[test]
    fn test_rewind_log_storage() {
        let tmp_dir = TempDir::new().expect("create temp dir");
        let db_path = tmp_dir.path().join("chaindb_rewind_log");
        let db = ChainDb::new(1, &db_path).expect("create db");

        let anchor = BlockInfo {
            hash: B256::from([2u8; 32]),
            number: 1,
            parent_hash: B256::from([3u8; 32]),
            timestamp: 0,
        };

        let next_block = BlockInfo {
            hash: B256::from([3u8; 32]),
            number: 2,
            parent_hash: anchor.hash,
            timestamp: 0,
        };

        db.initialise_log_storage(anchor).unwrap();
        db.store_block_logs(&next_block, vec![]).unwrap();

        // Add and promote next_block to CrossUnsafe and LocalUnsafe
        db.update_current_cross_unsafe(&next_block).unwrap();

        db.rewind_log_storage(&next_block.id()).expect("rewind log storage should succeed");

        // Should be rewound to anchor
        let local_unsafe =
            db.get_safety_head_ref(SafetyLevel::LocalUnsafe).expect("get safety head ref");
        let cross_unsafe =
            db.get_safety_head_ref(SafetyLevel::CrossUnsafe).expect("get safety head ref");

        assert_eq!(local_unsafe, anchor);
        assert_eq!(cross_unsafe, anchor);
    }

    #[test]
    fn test_rewind_log_storage_beyond_derivation_head_should_error() {
        let tmp_dir = tempfile::TempDir::new().unwrap();
        let db_path = tmp_dir.path().join("chaindb_rewind_beyond_derivation");
        let db = ChainDb::new(1, &db_path).unwrap();

        // Initialise anchor derived block and derivation storage
        let anchor = DerivedRefPair {
            source: BlockInfo {
                hash: B256::from([0u8; 32]),
                number: 100,
                parent_hash: B256::from([1u8; 32]),
                timestamp: 0,
            },
            derived: BlockInfo {
                hash: B256::from([2u8; 32]),
                number: 0,
                parent_hash: B256::from([3u8; 32]),
                timestamp: 0,
            },
        };

        db.initialise_log_storage(anchor.derived).unwrap();
        db.initialise_derivation_storage(anchor).unwrap();

        let block1 = BlockInfo {
            hash: B256::from([3u8; 32]),
            number: 1,
            parent_hash: anchor.derived.hash,
            timestamp: 0,
        };
        let source1 = BlockInfo {
            hash: B256::from([0u8; 32]),
            number: 100,
            parent_hash: B256::from([1u8; 32]),
            timestamp: 0,
        };

        let result = db.store_block_logs(&block1, Vec::new());
        assert!(result.is_ok(), "Should store block logs successfully");
        let result = db.save_source_block(source1);
        assert!(result.is_ok(), "Should save source block successfully");
        let result = db.save_derived_block(DerivedRefPair { source: source1, derived: block1 });
        assert!(result.is_ok(), "Should save derived block successfully");

        let block2 = BlockInfo {
            hash: B256::from([4u8; 32]),
            number: 2,
            parent_hash: block1.hash,
            timestamp: 0,
        };

        let result = db.store_block_logs(&block2, Vec::new());
        assert!(result.is_ok(), "Should store block logs successfully");

        // Attempt to rewind log storage beyond local safe head
        let err = db.rewind_log_storage(&anchor.derived.id()).unwrap_err();
        assert!(
            matches!(err, StorageError::RewindBeyondLocalSafeHead { to, local_safe } if to == anchor.derived.number && local_safe == block1.number),
            "Should not allow rewinding log storage beyond derivation head"
        );

        // Attempt to rewind log storage to the local safe head
        let result = db.rewind_log_storage(&block1.id()).unwrap_err();
        assert!(
            matches!(result, StorageError::RewindBeyondLocalSafeHead { to, local_safe } if to == block1.number && local_safe == block1.number),
            "Should not allow rewinding log storage to the local safe head"
        );
    }

    #[test]
    fn test_rewind_log_comprehensive() {
        let tmp_dir = tempfile::TempDir::new().unwrap();
        let db_path = tmp_dir.path().join("chaindb_rewind_beyond_derivation");
        let db = ChainDb::new(1, &db_path).unwrap();

        // Initialise anchor derived block and derivation storage
        let block0 = BlockInfo {
            hash: B256::from([2u8; 32]),
            number: 0,
            parent_hash: B256::ZERO,
            timestamp: 0,
        };

        let result = db.initialise_log_storage(block0);
        assert!(result.is_ok(), "Should initialise log storage successfully");

        let block1 = BlockInfo {
            hash: B256::from([3u8; 32]),
            number: 1,
            parent_hash: block0.hash,
            timestamp: 0,
        };

        let result = db.store_block_logs(&block1, Vec::new());
        assert!(result.is_ok(), "Should store block logs successfully");

        let block2 = BlockInfo {
            hash: B256::from([4u8; 32]),
            number: 2,
            parent_hash: block1.hash,
            timestamp: 0,
        };

        let result = db.store_block_logs(&block2, Vec::new());
        assert!(result.is_ok(), "Should store block logs successfully");

        db.update_current_cross_unsafe(&block1).expect("update cross unsafe");

        let result = db.rewind_log_storage(&block2.id());
        assert!(result.is_ok(), "Should rewind log storage successfully");

        let local_unsafe =
            db.get_safety_head_ref(SafetyLevel::LocalUnsafe).expect("get safety head ref");
        let cross_unsafe =
            db.get_safety_head_ref(SafetyLevel::CrossUnsafe).expect("get safety head ref");

        assert_eq!(local_unsafe, block1);
        assert_eq!(cross_unsafe, block1);

        let result = db.rewind_log_storage(&block1.id());
        assert!(result.is_ok(), "Should rewind log storage successfully");

        let local_unsafe =
            db.get_safety_head_ref(SafetyLevel::LocalUnsafe).expect("get safety head ref");
        let cross_unsafe =
            db.get_safety_head_ref(SafetyLevel::CrossUnsafe).expect("get safety head ref");

        assert_eq!(local_unsafe, block0);
        assert_eq!(cross_unsafe, block0);
    }

    #[test]
    fn test_rewind_log_storage_to_activation_block() {
        let tmp_dir = tempfile::TempDir::new().unwrap();
        let db_path = tmp_dir.path().join("chaindb_rewind_beyond_derivation");
        let db = ChainDb::new(1, &db_path).unwrap();

        // Initialise anchor derived block and derivation storage
        let block0 = BlockInfo {
            hash: B256::from([2u8; 32]),
            number: 0,
            parent_hash: B256::ZERO,
            timestamp: 0,
        };

        let result = db.initialise_log_storage(block0);
        assert!(result.is_ok(), "Should initialise log storage successfully");

        let block1 = BlockInfo {
            hash: B256::from([3u8; 32]),
            number: 1,
            parent_hash: block0.hash,
            timestamp: 0,
        };

        let result = db.store_block_logs(&block1, Vec::new());
        assert!(result.is_ok(), "Should store block logs successfully");

        let block2 = BlockInfo {
            hash: B256::from([4u8; 32]),
            number: 2,
            parent_hash: block1.hash,
            timestamp: 0,
        };

        let result = db.store_block_logs(&block2, Vec::new());
        assert!(result.is_ok(), "Should store block logs successfully");

        let result = db.rewind_log_storage(&block0.id());
        assert!(result.is_ok(), "Should rewind log storage successfully");
    }

    #[test]
    fn test_rewind_comprehensive() {
        let tmp_dir = TempDir::new().expect("create temp dir");
        let db_path = tmp_dir.path().join("chaindb_rewind_all");
        let db = ChainDb::new(1, &db_path).expect("create db");

        let anchor = DerivedRefPair {
            source: BlockInfo {
                hash: B256::from([0u8; 32]),
                number: 100,
                parent_hash: B256::from([1u8; 32]),
                timestamp: 0,
            },
            derived: BlockInfo {
                hash: B256::from([2u8; 32]),
                number: 1,
                parent_hash: B256::from([3u8; 32]),
                timestamp: 0,
            },
        };

        let pair1 = DerivedRefPair {
            source: BlockInfo {
                hash: B256::from([3u8; 32]),
                number: 101,
                parent_hash: anchor.source.hash,
                timestamp: 0,
            },
            derived: BlockInfo {
                hash: B256::from([4u8; 32]),
                number: 2,
                parent_hash: anchor.derived.hash,
                timestamp: 1,
            },
        };

        let pair2 = DerivedRefPair {
            source: BlockInfo {
                hash: B256::from([4u8; 32]),
                number: 102,
                parent_hash: pair1.source.hash,
                timestamp: 1,
            },
            derived: BlockInfo {
                hash: B256::from([5u8; 32]),
                number: 3,
                parent_hash: pair1.derived.hash,
                timestamp: 2,
            },
        };

        let unsafe_block = BlockInfo {
            hash: B256::from([5u8; 32]),
            number: 3,
            parent_hash: pair1.derived.hash,
            timestamp: 2,
        };

        db.initialise_log_storage(anchor.derived).expect("initialise log storage");
        db.initialise_derivation_storage(anchor).expect("initialise derivation storage");

        db.store_block_logs(&pair1.derived, vec![]).expect("store logs");
        db.store_block_logs(&unsafe_block, vec![]).expect("store logs");

        db.save_source_block(pair1.source).expect("save source block");
        db.save_derived_block(pair1).expect("save derived block");

        db.save_source_block(pair2.source).expect("save source block");
        db.save_derived_block(pair2).expect("save derived block");

        db.update_current_cross_unsafe(&pair1.derived).expect("update cross unsafe");
        db.update_current_cross_safe(&pair1.derived).expect("update cross safe");

        db.update_current_cross_unsafe(&pair2.derived).expect("update cross unsafe");
        db.update_current_cross_safe(&pair2.derived).expect("update cross safe");

        db.update_finalized_using_source(anchor.source).expect("update finalized using source");

        db.rewind(&pair2.derived.id()).expect("rewind should succeed");

        // Everything should be rewound to pair1.derived
        let local_unsafe = db.get_safety_head_ref(SafetyLevel::LocalUnsafe).unwrap();
        let cross_unsafe = db.get_safety_head_ref(SafetyLevel::CrossUnsafe).unwrap();
        let local_safe = db.get_safety_head_ref(SafetyLevel::LocalSafe).unwrap();
        let cross_safe = db.get_safety_head_ref(SafetyLevel::CrossSafe).unwrap();
        let latest_pair = db.latest_derivation_state().unwrap();
        let log_block = db.get_latest_block().unwrap();
        let finalized = db.get_safety_head_ref(SafetyLevel::Finalized).unwrap();

        assert_eq!(local_unsafe, pair1.derived);
        assert_eq!(cross_unsafe, pair1.derived);
        assert_eq!(local_safe, pair1.derived);
        assert_eq!(cross_safe, pair1.derived);
        assert_eq!(latest_pair, pair1);
        assert_eq!(log_block, pair1.derived);
        assert_eq!(finalized, anchor.derived);

        db.update_finalized_using_source(pair1.source).expect("update finalized using source");
        db.rewind(&pair1.derived.id()).expect("rewind should succeed");

        // Everything should be rewound to anchor.derived
        let local_unsafe = db.get_safety_head_ref(SafetyLevel::LocalUnsafe).unwrap();
        let cross_unsafe = db.get_safety_head_ref(SafetyLevel::CrossUnsafe).unwrap();
        let local_safe = db.get_safety_head_ref(SafetyLevel::LocalSafe).unwrap();
        let cross_safe = db.get_safety_head_ref(SafetyLevel::CrossSafe).unwrap();
        let latest_pair = db.latest_derivation_state().unwrap();
        let log_block = db.get_latest_block().unwrap();
        let finalized = db.get_safety_head_ref(SafetyLevel::Finalized).unwrap();

        assert_eq!(local_unsafe, anchor.derived);
        assert_eq!(cross_unsafe, anchor.derived);
        assert_eq!(local_safe, anchor.derived);
        assert_eq!(cross_safe, anchor.derived);
        assert_eq!(latest_pair, anchor);
        assert_eq!(log_block, anchor.derived);
        assert_eq!(finalized, anchor.derived);
    }

    #[test]
    fn test_rewind_to_activation_block() {
        let tmp_dir = TempDir::new().expect("create temp dir");
        let db_path = tmp_dir.path().join("chaindb_rewind_all");
        let db = ChainDb::new(1, &db_path).expect("create db");

        let pair0 = DerivedRefPair {
            source: BlockInfo {
                hash: B256::from([0u8; 32]),
                number: 100,
                parent_hash: B256::from([1u8; 32]),
                timestamp: 0,
            },
            derived: BlockInfo {
                hash: B256::from([2u8; 32]),
                number: 1,
                parent_hash: B256::from([3u8; 32]),
                timestamp: 0,
            },
        };

        let pair1 = DerivedRefPair {
            source: BlockInfo {
                hash: B256::from([3u8; 32]),
                number: 101,
                parent_hash: pair0.source.hash,
                timestamp: 0,
            },
            derived: BlockInfo {
                hash: B256::from([4u8; 32]),
                number: 2,
                parent_hash: pair0.derived.hash,
                timestamp: 1,
            },
        };

        let unsafe_block = BlockInfo {
            hash: B256::from([5u8; 32]),
            number: 3,
            parent_hash: pair1.derived.hash,
            timestamp: 2,
        };

        db.initialise_log_storage(pair0.derived).expect("initialise log storage");
        db.initialise_derivation_storage(pair0).expect("initialise derivation storage");

        db.store_block_logs(&pair1.derived, vec![]).expect("store logs");
        db.store_block_logs(&unsafe_block, vec![]).expect("store logs");

        db.save_source_block(pair1.source).expect("save source block");
        db.save_derived_block(pair1).expect("save derived block");

        db.update_current_cross_unsafe(&pair1.derived).expect("update cross unsafe");

        db.rewind(&pair0.derived.id()).expect("rewind should succeed");

        // Everything should return error
        let local_unsafe = db.get_safety_head_ref(SafetyLevel::LocalUnsafe);
        assert!(matches!(local_unsafe, Err(StorageError::FutureData)));

        let cross_unsafe = db.get_safety_head_ref(SafetyLevel::CrossUnsafe);
        assert!(matches!(cross_unsafe, Err(StorageError::FutureData)));

        let local_safe = db.get_safety_head_ref(SafetyLevel::LocalSafe);
        assert!(matches!(local_safe, Err(StorageError::FutureData)));

        let cross_safe = db.get_safety_head_ref(SafetyLevel::CrossSafe);
        assert!(matches!(cross_safe, Err(StorageError::FutureData)));

        let latest_derivation_state = db.latest_derivation_state();
        assert!(matches!(latest_derivation_state, Err(StorageError::DatabaseNotInitialised)));

        let latest_log_block = db.get_latest_block();
        assert!(matches!(latest_log_block, Err(StorageError::DatabaseNotInitialised)));
    }

    #[test]
    fn test_rewind_to_source_updates_logs_and_heads() {
        let tmp_dir = TempDir::new().expect("create temp dir");
        let db_path = tmp_dir.path().join("chaindb_rewind_to_source");
        let db = ChainDb::new(1, &db_path).expect("create db");

        // Anchor (activation)
        let anchor = DerivedRefPair {
            source: BlockInfo {
                hash: B256::from([0u8; 32]),
                number: 100,
                parent_hash: B256::from([1u8; 32]),
                timestamp: 0,
            },
            derived: BlockInfo {
                hash: B256::from([2u8; 32]),
                number: 0,
                parent_hash: B256::from([3u8; 32]),
                timestamp: 0,
            },
        };

        // Initialise DB with anchor
        db.initialise_log_storage(anchor.derived).expect("initialise log storage");
        db.initialise_derivation_storage(anchor).expect("initialise derivation storage");

        // Build two source entries and several derived blocks
        let source1 = BlockInfo {
            hash: B256::from([3u8; 32]),
            number: 101,
            parent_hash: anchor.source.hash,
            timestamp: 0,
        };
        let source2 = BlockInfo {
            hash: B256::from([4u8; 32]),
            number: 102,
            parent_hash: source1.hash,
            timestamp: 0,
        };

        // Derived blocks chained off the anchor/previous derived blocks
        let derived1 = BlockInfo {
            hash: B256::from([10u8; 32]),
            number: 1,
            parent_hash: anchor.derived.hash,
            timestamp: 0,
        };
        let derived2 = BlockInfo {
            hash: B256::from([11u8; 32]),
            number: 2,
            parent_hash: derived1.hash,
            timestamp: 0,
        };
        let derived3 = BlockInfo {
            hash: B256::from([12u8; 32]),
            number: 3,
            parent_hash: derived2.hash,
            timestamp: 0,
        };
        let derived4 = BlockInfo {
            hash: B256::from([13u8; 32]),
            number: 4,
            parent_hash: derived3.hash,
            timestamp: 0,
        };
        let derived5 = BlockInfo {
            hash: B256::from([14u8; 32]),
            number: 5,
            parent_hash: derived4.hash,
            timestamp: 0,
        };

        // Insert sources and derived blocks into storage (logs + derivation)
        assert!(db.save_source_block(source1).is_ok());
        db.store_block_logs(&derived1, vec![]).expect("store logs derived1");
        db.save_derived_block(DerivedRefPair { source: source1, derived: derived1 })
            .expect("save derived1");

        db.store_block_logs(&derived2, vec![]).expect("store logs derived2");
        db.save_derived_block(DerivedRefPair { source: source1, derived: derived2 })
            .expect("save derived2");

        db.store_block_logs(&derived3, vec![]).expect("store logs derived3");
        db.save_derived_block(DerivedRefPair { source: source1, derived: derived3 })
            .expect("save derived3");

        assert!(db.save_source_block(source2).is_ok());
        db.store_block_logs(&derived4, vec![]).expect("store logs derived4");
        db.save_derived_block(DerivedRefPair { source: source2, derived: derived4 })
            .expect("save derived4");

        db.store_block_logs(&derived5, vec![]).expect("store logs derived5");
        db.save_derived_block(DerivedRefPair { source: source2, derived: derived5 })
            .expect("save derived5");

        // Advance safety heads to be ahead of anchor so that rewind will need to reset them.
        db.update_current_cross_unsafe(&derived1).expect("update cross unsafe");
        db.update_current_cross_unsafe(&derived2).expect("update cross unsafe");
        db.update_current_cross_unsafe(&derived3).expect("update cross unsafe");
        db.update_current_cross_unsafe(&derived4).expect("update cross unsafe");

        db.update_current_cross_safe(&derived1).expect("update cross safe");
        db.update_current_cross_safe(&derived2).expect("update cross safe");

        // Now rewind to source1: expected derived rewind target is derived1 (first derived for
        // source1)
        let res = db.rewind_to_source(&source1.id()).expect("rewind_to_source should succeed");
        assert!(res.is_some(), "expected a derived rewind target");
        let rewind_target = res.unwrap();
        assert_eq!(rewind_target, derived1);

        // After rewind, logs should be rewound to before derived1 -> latest block == anchor.derived
        let latest_log = db.get_latest_block().expect("latest block after rewind");
        assert_eq!(latest_log, anchor.derived);

        // All safety heads that were ahead should be reset to the new latest (anchor.derived)
        let local_unsafe = db.get_safety_head_ref(SafetyLevel::LocalUnsafe).expect("local unsafe");
        let cross_unsafe = db.get_safety_head_ref(SafetyLevel::CrossUnsafe).expect("cross unsafe");
        let local_safe = db.get_safety_head_ref(SafetyLevel::LocalSafe).expect("local safe");
        let cross_safe = db.get_safety_head_ref(SafetyLevel::CrossSafe).expect("cross safe");

        assert_eq!(local_unsafe, anchor.derived);
        assert_eq!(cross_unsafe, anchor.derived);
        assert_eq!(local_safe, anchor.derived);
        assert_eq!(cross_safe, anchor.derived);
    }

    #[test]
    fn test_rewind_to_source_with_empty_source_returns_none() {
        let tmp_dir = TempDir::new().expect("create temp dir");
        let db_path = tmp_dir.path().join("chaindb_rewind_to_source_empty");
        let db = ChainDb::new(1, &db_path).expect("create db");

        // Anchor (activation)
        let anchor = DerivedRefPair {
            source: BlockInfo {
                hash: B256::from([0u8; 32]),
                number: 100,
                parent_hash: B256::from([1u8; 32]),
                timestamp: 0,
            },
            derived: BlockInfo {
                hash: B256::from([2u8; 32]),
                number: 0,
                parent_hash: B256::from([3u8; 32]),
                timestamp: 0,
            },
        };

        // Initialise DB with anchor
        db.initialise_log_storage(anchor.derived).expect("initialise log storage");
        db.initialise_derivation_storage(anchor).expect("initialise derivation storage");

        // Insert a source block that has no derived entries
        let source = BlockInfo {
            hash: B256::from([3u8; 32]),
            number: 101,
            parent_hash: anchor.source.hash,
            timestamp: 0,
        };
        db.save_source_block(source).expect("save source block");

        // Rewind to the source with empty derived list -> should return None
        let res = db.rewind_to_source(&source.id()).expect("rewind_to_source should succeed");
        assert!(res.is_none(), "Expected None when source has no derived blocks");

        // Ensure latest log and derivation state remain at the anchor
        let latest_log = db.get_latest_block().expect("latest block after noop rewind");
        assert_eq!(latest_log, anchor.derived);

        let latest_pair = db.latest_derivation_state().expect("latest derivation state");
        assert_eq!(latest_pair, anchor);
    }
}
