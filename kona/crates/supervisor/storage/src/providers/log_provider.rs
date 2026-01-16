//! Reth's MDBX-backed abstraction of [`LogProvider`] for superchain state.
//!
//! This module provides the [`LogProvider`] struct, which uses the
//! [`reth-db`] abstraction of reth to store execution logs
//! and block metadata required by the Optimism supervisor.
//!
//! It supports:
//! - Writing full blocks of logs with metadata
//! - Retrieving block metadata by number
//! - Finding a block from a specific log (with hash/index match)
//! - Fetching logs per block using dup-sorted key layout
//!
//! Logs are stored in [`LogEntries`] under dup-sorted tables, with log index
//! used as the subkey. Block metadata is stored in [`BlockRefs`].

use crate::{
    error::{EntryNotFoundError, StorageError},
    models::{BlockRefs, LogEntries},
};
use alloy_eips::BlockNumHash;
use alloy_primitives::ChainId;
use kona_protocol::BlockInfo;
use kona_supervisor_types::Log;
use reth_db_api::{
    cursor::{DbCursorRO, DbDupCursorRO, DbDupCursorRW},
    transaction::{DbTx, DbTxMut},
};

use tracing::{debug, error, info, trace, warn};

const DEFAULT_LOG_INTERVAL: u64 = 100;

/// A log storage that wraps a transactional reference to the MDBX backend.
#[derive(Debug)]
pub(crate) struct LogProvider<'tx, TX> {
    tx: &'tx TX,
    chain_id: ChainId,
    #[doc(hidden)]
    observability_interval: u64,
}

impl<'tx, TX> LogProvider<'tx, TX> {
    pub(crate) const fn new(tx: &'tx TX, chain_id: ChainId) -> Self {
        Self::new_with_observability_interval(tx, chain_id, DEFAULT_LOG_INTERVAL)
    }

    pub(crate) const fn new_with_observability_interval(
        tx: &'tx TX,
        chain_id: ChainId,
        observability_interval: u64,
    ) -> Self {
        Self { tx, chain_id, observability_interval }
    }
}

impl<TX> LogProvider<'_, TX>
where
    TX: DbTxMut + DbTx,
{
    pub(crate) fn initialise(&self, activation_block: BlockInfo) -> Result<(), StorageError> {
        match self.get_block(0) {
            Ok(block) if block == activation_block => Ok(()),
            Ok(_) => Err(StorageError::ConflictError),
            Err(StorageError::EntryNotFound(_)) => {
                self.store_block_logs_internal(&activation_block, Vec::new())
            }

            Err(err) => Err(err),
        }
    }

    pub(crate) fn store_block_logs(
        &self,
        block: &BlockInfo,
        logs: Vec<Log>,
    ) -> Result<(), StorageError> {
        debug!(
            target: "supervisor::storage",
            chain_id = %self.chain_id,
            block_number = block.number,
            "Storing logs",
        );

        let latest_block = match self.get_latest_block() {
            Ok(block) => block,
            Err(StorageError::EntryNotFound(_)) => return Err(StorageError::DatabaseNotInitialised),
            Err(e) => return Err(e),
        };

        if latest_block.number >= block.number {
            // If the latest block is ahead of the incoming block, it means
            // the incoming block is old block, check if it is same as the stored block.
            let stored_block = self.get_block(block.number)?;
            if stored_block == *block {
                return Ok(());
            }
            warn!(
                target: "supervisor::storage",
                chain_id = %self.chain_id,
                %stored_block,
                incoming_block = %block,
                "Incoming log block is not consistent with the stored log block",
            );
            return Err(StorageError::ConflictError)
        }

        if !latest_block.is_parent_of(block) {
            warn!(
                target: "supervisor::storage",
                chain_id = %self.chain_id,
                %latest_block,
                incoming_block = %block,
                "Incoming block does not follow latest stored block"
            );
            return Err(StorageError::BlockOutOfOrder);
        }

        self.store_block_logs_internal(block, logs)
    }

    fn store_block_logs_internal(
        &self,
        block: &BlockInfo,
        logs: Vec<Log>,
    ) -> Result<(), StorageError> {
        self.tx.put::<BlockRefs>(block.number, (*block).into()).inspect_err(|err| {
            error!(
                target: "supervisor::storage",
                chain_id = %self.chain_id,
                block_number = block.number,
                %err,
                "Failed to insert block"
            );
        })?;

        let mut cursor = self.tx.cursor_dup_write::<LogEntries>().inspect_err(|err| {
            error!(
                target: "supervisor::storage",
                chain_id = %self.chain_id,
                %err,
                "Failed to get dup cursor"
            );
        })?;

        for log in logs {
            cursor.append_dup(block.number, log.into()).inspect_err(|err| {
                error!(
                    target: "supervisor::storage",
                    chain_id = %self.chain_id,
                    block_number = block.number,
                    %err,
                    "Failed to append logs"
                );
            })?;
        }
        Ok(())
    }

    /// Rewinds the log storage by deleting all blocks and logs from the given block onward.
    /// Fails if the given block exists with a mismatching hash (to prevent unsafe deletion).
    pub(crate) fn rewind_to(&self, block: &BlockNumHash) -> Result<(), StorageError> {
        info!(
            target: "supervisor::storage",
            chain_id = %self.chain_id,
            target_block_number = %block.number,
            target_block_hash = %block.hash,
            "Starting rewind of log storage"
        );

        // Get the latest block number from BlockRefs
        let latest_block = {
            let mut cursor = self.tx.cursor_read::<BlockRefs>()?;
            cursor.last()?.map(|(num, _)| num).unwrap_or(block.number)
        };

        // Check for future block
        if block.number > latest_block {
            error!(
                target: "supervisor::storage",
                chain_id = %self.chain_id,
                target_block_number = %block.number,
                latest_block,
                "Cannot rewind to future block"
            );
            return Err(StorageError::FutureData);
        }

        // total blocks to rewind down to and including tgt block
        let total_blocks = latest_block - block.number + 1;
        let mut processed_blocks = 0;

        // Delete all blocks and logs with number ≥ `block.number`
        {
            let mut cursor = self.tx.cursor_write::<BlockRefs>()?;
            let mut walker = cursor.walk(Some(block.number))?;

            trace!(
                target: "supervisor::storage",
                chain_id = %self.chain_id,
                target_block_number = %block.number,
                target_block_hash = %block.hash,
                latest_block,
                total_blocks,
                observability_interval = %self.observability_interval,
                "Rewinding log storage..."
            );

            while let Some(Ok((key, stored_block))) = walker.next() {
                if key == block.number && block.hash != stored_block.hash {
                    warn!(
                        target: "supervisor::storage",
                        chain_id = %self.chain_id,
                        %stored_block,
                        incoming_block = ?block,
                        "Requested block to rewind does not match stored block"
                    );
                    return Err(StorageError::ConflictError);
                }
                // remove the block
                walker.delete_current()?;

                // remove the logs of that block
                self.tx.delete::<LogEntries>(key, None)?;

                processed_blocks += 1;

                // Log progress periodically or on last block
                if processed_blocks % self.observability_interval == 0 ||
                    processed_blocks == total_blocks
                {
                    let percentage = if total_blocks > 0 {
                        (processed_blocks as f64 / total_blocks as f64 * 100.0).min(100.0)
                    } else {
                        100.0
                    };

                    info!(
                       target: "supervisor::storage",
                       chain_id = %self.chain_id,
                       block_number = %key,
                       percentage = %format!("{:.2}%", percentage),
                       processed_blocks,
                       total_blocks,
                       "Rewind progress"
                    );
                }
            }

            info!(
                target: "supervisor::storage",
                target_block_number = ?block.number,
                target_block_hash = %block.hash,
                chain_id = %self.chain_id,
                total_blocks,
                "Rewind completed successfully"
            );
        }

        Ok(())
    }
}

impl<TX> LogProvider<'_, TX>
where
    TX: DbTx,
{
    pub(crate) fn get_block(&self, block_number: u64) -> Result<BlockInfo, StorageError> {
        debug!(
            target: "supervisor::storage",
            chain_id = %self.chain_id,
            block_number,
            "Fetching block"
        );

        let block_option = self.tx.get::<BlockRefs>(block_number).inspect_err(|err| {
            error!(
                target: "supervisor::storage",
                chain_id = %self.chain_id,
                block_number,
                %err,
                "Failed to read block",
            );
        })?;

        let block = block_option.ok_or_else(|| {
            warn!(
                target: "supervisor::storage",
                chain_id = %self.chain_id,
                block_number,
                "Block not found"
            );
            EntryNotFoundError::DerivedBlockNotFound(block_number)
        })?;
        Ok(block.into())
    }

    pub(crate) fn get_latest_block(&self) -> Result<BlockInfo, StorageError> {
        debug!(target: "supervisor::storage", chain_id = %self.chain_id, "Fetching latest block");

        let mut cursor = self.tx.cursor_read::<BlockRefs>().inspect_err(|err| {
            error!(
                target: "supervisor::storage",
                chain_id = %self.chain_id,
                %err,
                "Failed to get cursor"
            );
        })?;

        let result = cursor.last().inspect_err(|err| {
            error!(
                target: "supervisor::storage",
                chain_id = %self.chain_id,
                %err,
                "Failed to seek to last block"
            );
        })?;

        let (_, block) = result.ok_or_else(|| {
            warn!(
                target: "supervisor::storage",
                chain_id = %self.chain_id,
                "No blocks found in storage"
            );
            StorageError::DatabaseNotInitialised
        })?;
        Ok(block.into())
    }

    pub(crate) fn get_log(&self, block_number: u64, log_index: u32) -> Result<Log, StorageError> {
        debug!(
            target: "supervisor::storage",
            chain_id = %self.chain_id,
            block_number,
            log_index,
            "Fetching block  by log"
        );

        let mut cursor = self.tx.cursor_dup_read::<LogEntries>().inspect_err(|err| {
            error!(
                target: "supervisor::storage",
                chain_id = %self.chain_id,
                %err,
                "Failed to get cursor for LogEntries"
            );
        })?;

        let result = cursor.seek_by_key_subkey(block_number, log_index).inspect_err(|err| {
            error!(
                target: "supervisor::storage",
                chain_id = %self.chain_id,
                block_number,
                log_index,
                %err,
                "Failed to read log entry"
            );
        })?;

        let log_entry = result.ok_or_else(|| {
            warn!(
                target: "supervisor::storage",
                chain_id = %self.chain_id,
                block_number,
                log_index,
                "Log not found"
            );
            EntryNotFoundError::LogNotFound { block_number, log_index }
        })?;

        Ok(Log::from(log_entry))
    }

    pub(crate) fn get_logs(&self, block_number: u64) -> Result<Vec<Log>, StorageError> {
        debug!(target: "supervisor::storage", chain_id = %self.chain_id, block_number, "Fetching logs");

        let mut cursor = self.tx.cursor_dup_read::<LogEntries>().inspect_err(|err| {
            error!(
                target: "supervisor::storage",
                chain_id = %self.chain_id,
                %err,
                "Failed to get dup cursor"
            );
        })?;

        let walker = cursor.walk_range(block_number..=block_number).inspect_err(|err| {
            error!(
                target: "supervisor::storage",
                chain_id = %self.chain_id,
                block_number,
                %err,
                "Failed to walk dup range",
            );
        })?;

        let mut logs = Vec::new();
        for row in walker {
            match row {
                Ok((_, entry)) => logs.push(entry.into()),
                Err(err) => {
                    error!(
                        target: "supervisor::storage",
                        chain_id = %self.chain_id,
                        block_number,
                        %err,
                        "Failed to read log entry",
                    );
                    return Err(StorageError::Database(err));
                }
            }
        }
        Ok(logs)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::models::Tables;
    use alloy_primitives::B256;
    use kona_cli::init_test_tracing;
    use kona_protocol::BlockInfo;
    use kona_supervisor_types::{ExecutingMessage, Log};
    use reth_db::{
        DatabaseEnv,
        mdbx::{DatabaseArguments, init_db_for},
    };
    use reth_db_api::Database;
    use tempfile::TempDir;

    static CHAIN_ID: ChainId = 1;

    fn genesis_block() -> BlockInfo {
        BlockInfo {
            hash: B256::from([0u8; 32]),
            number: 0,
            parent_hash: B256::ZERO,
            timestamp: 100,
        }
    }

    fn sample_block_info(block_number: u64, parent_hash: B256) -> BlockInfo {
        BlockInfo {
            number: block_number,
            hash: B256::from([0x11; 32]),
            parent_hash,
            timestamp: 123456,
        }
    }

    fn sample_log(log_index: u32, with_msg: bool) -> Log {
        Log {
            index: log_index,
            hash: B256::from([log_index as u8; 32]),
            executing_message: if with_msg {
                Some(ExecutingMessage {
                    chain_id: 10,
                    block_number: 999,
                    log_index: 7,
                    hash: B256::from([0x44; 32]),
                    timestamp: 88888,
                })
            } else {
                None
            },
        }
    }

    /// Sets up a new temp DB
    fn setup_db() -> DatabaseEnv {
        let temp_dir = TempDir::new().expect("Could not create temp dir");
        init_db_for::<_, Tables>(temp_dir.path(), DatabaseArguments::default())
            .expect("Failed to init database")
    }

    /// Helper to initialize database in a new transaction, committing if successful.
    fn initialize_db(db: &DatabaseEnv, block: &BlockInfo) -> Result<(), StorageError> {
        let tx = db.tx_mut().expect("Could not get mutable tx");
        let provider = LogProvider::new(&tx, CHAIN_ID);
        let res = provider.initialise(*block);
        if res.is_ok() {
            tx.commit().expect("Failed to commit transaction");
        } else {
            tx.abort();
        }
        res
    }

    /// Helper to insert a pair in a new transaction, committing if successful.
    fn insert_block_logs(
        db: &DatabaseEnv,
        block: &BlockInfo,
        logs: Vec<Log>,
    ) -> Result<(), StorageError> {
        let tx = db.tx_mut().expect("Could not get mutable tx");
        let provider = LogProvider::new(&tx, CHAIN_ID);
        let res = provider.store_block_logs(block, logs);
        if res.is_ok() {
            tx.commit().expect("Failed to commit transaction");
        }
        res
    }

    #[test]
    fn initialise_inserts_anchor_if_not_exists() {
        let db = setup_db();
        let genesis = genesis_block();

        // Should succeed and insert the anchor
        assert!(initialize_db(&db, &genesis).is_ok());

        // Check that the anchor is present
        let tx = db.tx().expect("Could not get tx");
        let provider = LogProvider::new(&tx, CHAIN_ID);
        let stored = provider.get_block(genesis.number).expect("should exist");
        assert_eq!(stored.hash, genesis.hash);
    }

    #[test]
    fn initialise_is_idempotent_if_anchor_matches() {
        let db = setup_db();
        let genesis = genesis_block();

        // First initialise
        assert!(initialize_db(&db, &genesis).is_ok());

        // Second initialise with the same anchor should succeed (idempotent)
        assert!(initialize_db(&db, &genesis).is_ok());
    }

    #[test]
    fn initialise_fails_if_anchor_mismatch() {
        let db = setup_db();

        // Initialize with the genesis block
        let genesis = genesis_block();
        assert!(initialize_db(&db, &genesis).is_ok());

        // Try to initialise with a different anchor (different hash)
        let mut wrong_genesis = genesis;
        wrong_genesis.hash = B256::from([42u8; 32]);

        let result = initialize_db(&db, &wrong_genesis);
        assert!(matches!(result, Err(StorageError::ConflictError)));
    }

    #[test]
    fn test_get_latest_block_empty() {
        let db = setup_db();

        let tx = db.tx().expect("Failed to start RO tx");
        let log_reader = LogProvider::new(&tx, CHAIN_ID);

        let result = log_reader.get_latest_block();
        assert!(matches!(result, Err(StorageError::DatabaseNotInitialised)));
    }

    #[test]
    fn test_storage_read_write_success() {
        let db = setup_db();

        // Initialize with genesis block
        let genesis = genesis_block();
        initialize_db(&db, &genesis).expect("Failed to initialize DB with genesis block");

        let block1 = sample_block_info(1, genesis.hash);
        let logs1 = vec![
            sample_log(0, false),
            sample_log(1, true),
            sample_log(3, false),
            sample_log(4, true),
        ];

        // Store logs for block1
        assert!(insert_block_logs(&db, &block1, logs1.clone()).is_ok());

        let block2 = sample_block_info(2, block1.hash);
        let logs2 = vec![sample_log(0, false), sample_log(1, true)];

        // Store logs for block2
        assert!(insert_block_logs(&db, &block2, logs2.clone()).is_ok());

        let block3 = sample_block_info(3, block2.hash);
        let logs3 = vec![sample_log(0, false), sample_log(1, true), sample_log(2, true)];

        // Store logs for block3
        assert!(insert_block_logs(&db, &block3, logs3).is_ok());

        let tx = db.tx().expect("Failed to start RO tx");
        let log_reader = LogProvider::new(&tx, CHAIN_ID);

        // get_block
        let block = log_reader.get_block(block2.number).expect("Failed to get block");
        assert_eq!(block, block2);

        // get_latest_block
        let block = log_reader.get_latest_block().expect("Failed to get latest block");
        assert_eq!(block, block3);

        // get log
        let log = log_reader.get_log(1, 1).expect("Failed to get block by log");
        assert_eq!(log, logs1[1]);

        // get_logs
        let logs = log_reader.get_logs(block2.number).expect("Failed to get logs");
        assert_eq!(logs.len(), 2);
        assert_eq!(logs[0], logs2[0]);
        assert_eq!(logs[1], logs2[1]);
    }

    #[test]
    fn test_not_found_error_and_empty_results() {
        let db = setup_db();

        let tx = db.tx().expect("Failed to start RO tx");
        let log_reader = LogProvider::new(&tx, CHAIN_ID);

        let result = log_reader.get_latest_block();
        assert!(matches!(result, Err(StorageError::DatabaseNotInitialised)));

        // Initialize with genesis block
        let genesis = genesis_block();
        initialize_db(&db, &genesis).expect("Failed to initialize DB with genesis block");

        assert!(
            insert_block_logs(&db, &sample_block_info(1, genesis.hash), vec![sample_log(0, true)])
                .is_ok()
        );

        let result = log_reader.get_block(2);
        assert!(matches!(result, Err(StorageError::EntryNotFound(_))));

        // should return empty logs but not an error
        let logs = log_reader.get_logs(2).expect("Should not return error");
        assert_eq!(logs.len(), 0);

        let result = log_reader.get_log(1, 1);
        assert!(matches!(result, Err(StorageError::EntryNotFound(_))));
    }

    #[test]
    fn test_block_append_failed_on_order_mismatch() {
        let db = setup_db();

        // Initialize with genesis block
        let genesis = genesis_block();
        initialize_db(&db, &genesis).expect("Failed to initialize DB with genesis block");

        let block1 = sample_block_info(1, genesis.hash);
        let logs1 = vec![sample_log(0, false)];

        let block2 = sample_block_info(3, genesis.hash);
        let logs2 = vec![sample_log(0, false), sample_log(1, true)];

        // Store logs
        assert!(insert_block_logs(&db, &block1, logs1).is_ok());

        let result = insert_block_logs(&db, &block2, logs2);
        assert!(matches!(result, Err(StorageError::BlockOutOfOrder)));
    }

    #[test]
    fn store_block_logs_skips_if_block_already_exists() {
        let db = setup_db();
        let genesis = genesis_block();
        initialize_db(&db, &genesis).expect("Failed to initialize DB with genesis block");

        let block1 = sample_block_info(1, genesis.hash);
        let logs1 = vec![sample_log(0, false)];

        // Store block1 for the first time
        assert!(insert_block_logs(&db, &block1, logs1.clone()).is_ok());

        // Try storing the same block again (should skip and succeed)
        assert!(insert_block_logs(&db, &block1, logs1.clone()).is_ok());

        // Try storing genesis block again (should skip and succeed)
        assert!(insert_block_logs(&db, &genesis, Vec::new()).is_ok());

        // Check that the logs are still present and correct
        let tx = db.tx().expect("Failed to start RO tx");
        let log_reader = LogProvider::new(&tx, CHAIN_ID);
        let logs = log_reader.get_logs(block1.number).expect("Should get logs");
        assert_eq!(logs, logs1);
    }

    #[test]
    fn store_block_logs_returns_conflict_if_block_exists_with_different_data() {
        let db = setup_db();
        let genesis = genesis_block();
        initialize_db(&db, &genesis).expect("Failed to initialize DB with genesis block");

        let block1 = sample_block_info(1, genesis.hash);
        let logs1 = vec![sample_log(0, false)];
        assert!(insert_block_logs(&db, &block1, logs1).is_ok());

        // Try storing block1 again with a different hash (simulate conflict)
        let mut block1_conflict = block1;
        block1_conflict.hash = B256::from([0x22; 32]);
        let logs1_conflict = vec![sample_log(0, false)];

        let result = insert_block_logs(&db, &block1_conflict, logs1_conflict);
        assert!(matches!(result, Err(StorageError::ConflictError)));

        // Try storing genesis block again with a different hash (simulate conflict)
        let mut genesis_conflict = genesis;
        genesis_conflict.hash = B256::from([0x33; 32]);
        let result = insert_block_logs(&db, &genesis_conflict, Vec::new());
        assert!(matches!(result, Err(StorageError::ConflictError)));
    }

    #[test]
    fn test_rewind_to() {
        init_test_tracing();

        let db = setup_db();
        let genesis = genesis_block();
        initialize_db(&db, &genesis).expect("Failed to initialize DB");

        // Add 5 blocks with logs
        let mut blocks = vec![genesis];
        for i in 1..=5 {
            let prev = &blocks[i - 1];
            let block = sample_block_info(i as u64, prev.hash);
            let logs = (0..3).map(|j| sample_log(j, j % 2 == 0)).collect();
            insert_block_logs(&db, &block, logs).expect("Failed to insert logs");
            blocks.push(block);
        }

        // Rewind to block 3, blocks 3, 4, 5 should be removed
        let tx = db.tx_mut().expect("Could not get mutable tx");
        let provider = LogProvider::new_with_observability_interval(&tx, CHAIN_ID, 1);
        provider.rewind_to(&blocks[3].id()).expect("Failed to rewind blocks");
        tx.commit().expect("Failed to commit rewind");

        let tx = db.tx().expect("Could not get RO tx");
        let provider = LogProvider::new_with_observability_interval(&tx, CHAIN_ID, 1);

        // Blocks 0,1,2 should still exist
        for i in 0..=2 {
            assert!(provider.get_block(i).is_ok(), "block {i} should exist after rewind");
        }

        // Logs for blocks 1,2 should exist
        for i in 1..=2 {
            let logs = provider.get_logs(i).expect("logs should exist");
            assert_eq!(logs.len(), 3, "block {i} should have 3 logs");
        }

        // Blocks 3,4,5 should be gone
        for i in 3..=5 {
            assert!(
                matches!(provider.get_block(i), Err(StorageError::EntryNotFound(_))),
                "block {i} should be removed"
            );

            let logs = provider.get_logs(i).expect("get_logs should not fail");
            assert!(logs.is_empty(), "logs for block {i} should be empty");
        }
    }

    #[test]
    fn test_rewind_to_conflict_hash() {
        let db = setup_db();
        let genesis = genesis_block();
        initialize_db(&db, &genesis).expect("Failed to initialize DB");

        // Insert block 1
        let block1 = sample_block_info(1, genesis.hash);
        insert_block_logs(&db, &block1, vec![sample_log(0, true)]).expect("insert block 1");

        // Create a conflicting block with the same number but different hash
        let mut conflicting_block1 = block1;
        conflicting_block1.hash = B256::from([0xAB; 32]); // different hash

        let tx = db.tx_mut().expect("Failed to get tx");
        let provider = LogProvider::new(&tx, CHAIN_ID);

        let result = provider.rewind_to(&conflicting_block1.id());
        assert!(
            matches!(result, Err(StorageError::ConflictError)),
            "Expected conflict error due to hash mismatch"
        );
    }
}
