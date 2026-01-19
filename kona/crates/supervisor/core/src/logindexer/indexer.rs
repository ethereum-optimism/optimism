use crate::{
    logindexer::{log_to_log_hash, payload_hash_to_log_hash},
    syncnode::{BlockProvider, ManagedNodeError},
};
use alloy_primitives::ChainId;
use kona_interop::parse_log_to_executing_message;
use kona_protocol::BlockInfo;
use kona_supervisor_storage::{LogStorageReader, LogStorageWriter, StorageError};
use kona_supervisor_types::{ExecutingMessage, Log};
use std::sync::Arc;
use thiserror::Error;
use tokio::sync::Mutex;
use tracing::{debug, error};

/// The [`LogIndexer`] is responsible for processing L2 receipts, extracting [`ExecutingMessage`]s,
/// and persisting them to the state manager.
#[derive(Debug)]
pub struct LogIndexer<P, S> {
    /// The chain ID of the rollup.
    chain_id: ChainId,
    /// Component that provides receipts for a given block hash.
    block_provider: Mutex<Option<Arc<P>>>,
    /// Component that persists parsed log entries to storage.
    log_storage: Arc<S>,
    /// Protects concurrent catch-up
    is_catch_up_running: Mutex<bool>,
}

impl<P, S> LogIndexer<P, S>
where
    P: BlockProvider + 'static,
    S: LogStorageWriter + LogStorageReader + 'static,
{
    /// Creates a new [`LogIndexer`] with the given receipt provider and state manager.
    ///
    /// # Arguments
    /// - `block_provider`: Shared reference to a component capable of fetching block ref and
    ///   receipts.
    /// - `log_storage`: Shared reference to the storage layer for persisting parsed logs.
    pub fn new(chain_id: ChainId, block_provider: Option<Arc<P>>, log_storage: Arc<S>) -> Self {
        Self {
            chain_id,
            block_provider: Mutex::new(block_provider),
            log_storage,
            is_catch_up_running: Mutex::new(false),
        }
    }

    /// Sets the block provider
    pub async fn set_block_provider(&self, block_provider: Arc<P>) {
        let mut guard = self.block_provider.lock().await;
        *guard = Some(block_provider);
    }

    /// Asynchronously initiates a background task to catch up and index logs
    /// starting from the latest successfully indexed block up to the specified block.
    ///
    /// If a catch-up job is already running, this call is ignored.
    ///
    /// # Arguments
    /// - `block`: The target block to sync logs up to (inclusive).
    pub fn sync_logs(self: Arc<Self>, block: BlockInfo) {
        tokio::spawn(async move {
            let mut running = self.is_catch_up_running.lock().await;

            if *running {
                debug!(target: "supervisor::log_indexer", chain_id = %self.chain_id, "Catch-up running log index");
                return;
            }

            *running = true;
            drop(running); // release the lock while the job runs

            if let Err(err) = self.index_log_upto(&block).await {
                error!(
                    target: "supervisor::log_indexer",
                    chain_id = %self.chain_id,
                    %err,
                    "Log indexer catch-up failed"
                );
            }

            let mut running = self.is_catch_up_running.lock().await;
            *running = false;
        });
    }

    /// Performs log indexing sequentially from the latest indexed block up to the given target
    /// block.
    async fn index_log_upto(&self, block: &BlockInfo) -> Result<(), LogIndexerError> {
        let mut current_number = self.log_storage.get_latest_block()?.number + 1;

        while current_number < block.number {
            let provider = {
                let guard = self.block_provider.lock().await;
                guard.as_ref().ok_or(LogIndexerError::NoBlockProvider)?.clone()
            };

            let current_block = provider.block_by_number(current_number).await?;
            self.process_and_store_logs(&current_block).await?;
            current_number += 1;
        }
        self.process_and_store_logs(block).await?;

        Ok(())
    }

    /// Processes and stores the logs of a given block in into the state manager.
    ///
    /// This function:
    /// - Fetches all receipts for the given block from the specified chain.
    /// - Iterates through all logs in all receipts.
    /// - For each log, computes a hash from the log and optionally parses an [`ExecutingMessage`].
    /// - Records each [`Log`] including the message if found.
    /// - Saves all log entries atomically using the [`LogStorageWriter`].
    ///
    /// # Arguments
    /// - `block`: Metadata about the block being processed.
    pub async fn process_and_store_logs(&self, block: &BlockInfo) -> Result<(), LogIndexerError> {
        let provider = {
            let guard = self.block_provider.lock().await;
            guard.as_ref().ok_or(LogIndexerError::NoBlockProvider)?.clone()
        };

        let receipts = provider.fetch_receipts(block.hash).await?;
        let mut log_entries = Vec::with_capacity(receipts.len());
        let mut log_index: u32 = 0;

        for receipt in receipts {
            for log in receipt.logs() {
                let log_hash = log_to_log_hash(log);

                let executing_message = parse_log_to_executing_message(log).map(|msg| {
                    let payload_hash =
                        payload_hash_to_log_hash(msg.payloadHash, msg.identifier.origin);
                    ExecutingMessage {
                        chain_id: msg.identifier.chainId.try_into().unwrap(),
                        block_number: msg.identifier.blockNumber.try_into().unwrap(),
                        log_index: msg.identifier.logIndex.try_into().unwrap(),
                        timestamp: msg.identifier.timestamp.try_into().unwrap(),
                        hash: payload_hash,
                    }
                });

                log_entries.push(Log { index: log_index, hash: log_hash, executing_message });

                log_index += 1;
            }
        }

        log_entries.shrink_to_fit();

        self.log_storage.store_block_logs(block, log_entries)?;
        Ok(())
    }
}

/// Error type for the [`LogIndexer`].
#[derive(Error, Debug, PartialEq, Eq)]
pub enum LogIndexerError {
    /// No block provider set when attempting to index logs.
    #[error("no block provider set")]
    NoBlockProvider,

    /// Failed to write processed logs for a block to the state manager.
    #[error(transparent)]
    StateWrite(#[from] StorageError),

    /// Failed to fetch logs for a block from the state manager.   
    #[error(transparent)]
    FetchReceipt(#[from] ManagedNodeError),
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::syncnode::{AuthenticationError, ClientError};
    use alloy_primitives::{Address, B256, Bytes};
    use async_trait::async_trait;
    use kona_interop::{ExecutingMessageBuilder, InteropProvider, SuperchainBuilder};
    use kona_protocol::BlockInfo;
    use kona_supervisor_storage::StorageError;
    use kona_supervisor_types::{Log, Receipts};
    use mockall::mock;
    use op_alloy_consensus::{OpReceiptEnvelope, OpTxType};
    use std::sync::Arc;
    mock! (
        #[derive(Debug)]
        pub BlockProvider {}

        #[async_trait]
        impl BlockProvider for BlockProvider {
            async fn fetch_receipts(&self, block_hash: B256) -> Result<Receipts, ManagedNodeError>;
            async fn block_by_number(&self, number: u64) -> Result<BlockInfo, ManagedNodeError>;
        }
    );

    mock!(
         #[derive(Debug)]
        pub Db {}

        impl LogStorageWriter for Db {
            fn initialise_log_storage(&self, _block: BlockInfo) -> Result<(), StorageError>;
            fn store_block_logs(&self, block: &BlockInfo, logs: Vec<Log>) -> Result<(), StorageError>;
        }

        impl LogStorageReader for Db {
            fn get_block(&self, block_number: u64) -> Result<BlockInfo, StorageError>;
            fn get_latest_block(&self) -> Result<BlockInfo, StorageError>;
            fn get_log(&self,block_number: u64,log_index: u32) -> Result<Log, StorageError>;
            fn get_logs(&self, block_number: u64) -> Result<Vec<Log>, StorageError>;
        }
    );

    fn hash_for_number(n: u64) -> B256 {
        let mut bytes = [0u8; 32];
        bytes[24..].copy_from_slice(&n.to_be_bytes());
        B256::from(bytes)
    }

    async fn build_receipts() -> Receipts {
        let mut builder = SuperchainBuilder::new();
        builder
            .chain(10)
            .with_timestamp(123456)
            .add_initiating_message(Bytes::from_static(b"init-msg"))
            .add_executing_message(
                ExecutingMessageBuilder::default()
                    .with_message_hash(B256::repeat_byte(0xaa))
                    .with_origin_address(Address::ZERO)
                    .with_origin_log_index(0)
                    .with_origin_block_number(1)
                    .with_origin_chain_id(10)
                    .with_origin_timestamp(123456),
            );
        let (headers, _, mock_provider) = builder.build();
        let block = headers.get(&10).unwrap();

        mock_provider.receipts_by_hash(10, block.hash()).await.unwrap()
    }

    #[tokio::test]
    async fn test_process_and_store_logs_success() {
        let receipts = build_receipts().await;
        let block_hash = B256::random();
        let block_info =
            BlockInfo { number: 1, hash: block_hash, timestamp: 123456789, ..Default::default() };

        let mut mock_provider = MockBlockProvider::new();
        mock_provider
            .expect_fetch_receipts()
            .withf(move |hash| *hash == block_hash)
            .returning(move |_| Ok(receipts.clone()));

        mock_provider.expect_block_by_number().returning(|_| Ok(BlockInfo::default())); // Not used here

        let mut mock_db = MockDb::new();
        mock_db
            .expect_store_block_logs()
            .withf(|block, logs| block.number == 1 && logs.len() == 2)
            .returning(|_, _| Ok(()));

        let log_indexer = LogIndexer::new(1, Some(Arc::new(mock_provider)), Arc::new(mock_db));

        let result = log_indexer.process_and_store_logs(&block_info).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_process_and_store_logs_with_empty_logs() {
        let block_hash = B256::random();
        let block_info =
            BlockInfo { number: 2, hash: block_hash, timestamp: 111111111, ..Default::default() };

        let empty_log_receipt =
            OpReceiptEnvelope::from_parts(true, 21000, vec![], OpTxType::Eip1559, None, None);
        let receipts = vec![empty_log_receipt];

        let mut mock_provider = MockBlockProvider::new();
        mock_provider
            .expect_fetch_receipts()
            .withf(move |hash| *hash == block_hash)
            .returning(move |_| Ok(receipts.clone()));

        mock_provider.expect_block_by_number().returning(|_| Ok(BlockInfo::default())); // Not used

        let mut mock_db = MockDb::new();
        mock_db
            .expect_store_block_logs()
            .withf(|block, logs| block.number == 2 && logs.is_empty())
            .returning(|_, _| Ok(()));

        let log_indexer = LogIndexer::new(1, Some(Arc::new(mock_provider)), Arc::new(mock_db));

        let result = log_indexer.process_and_store_logs(&block_info).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_process_and_store_logs_receipt_fetch_fails() {
        let block_hash = B256::random();
        let block_info =
            BlockInfo { number: 3, hash: block_hash, timestamp: 123456, ..Default::default() };

        let mut mock_provider = MockBlockProvider::new();
        mock_provider.expect_fetch_receipts().withf(move |hash| *hash == block_hash).returning(
            |_| {
                Err(ManagedNodeError::ClientError(ClientError::Authentication(
                    AuthenticationError::InvalidHeader,
                )))
            },
        );

        mock_provider.expect_block_by_number().returning(|_| Ok(BlockInfo::default())); // Not used

        let mock_db = MockDb::new(); // No call expected

        let log_indexer = LogIndexer::new(1, Some(Arc::new(mock_provider)), Arc::new(mock_db));

        let result = log_indexer.process_and_store_logs(&block_info).await;
        assert!(result.is_err());
    }

    #[tokio::test]
    async fn test_sync_logs_stores_all_blocks_in_range() {
        let target_block = BlockInfo {
            number: 5,
            hash: B256::random(),
            timestamp: 123456789,
            ..Default::default()
        };

        // BlockProvider mock
        let mut mock_provider = MockBlockProvider::new();
        mock_provider.expect_block_by_number().withf(|n| *n >= 1 && *n <= 5).returning(|n| {
            Ok(BlockInfo {
                number: n,
                hash: hash_for_number(n),
                timestamp: 0,
                ..Default::default()
            })
        });

        mock_provider.expect_fetch_receipts().times(5).returning(move |_| {
            Ok(vec![]) // Empty receipts
        });

        // Db mock
        let mut mock_db = MockDb::new();
        mock_db
            .expect_get_latest_block()
            .returning(|| Ok(BlockInfo { number: 0, ..Default::default() }));

        mock_db.expect_store_block_logs().times(5).returning(move |_, _| Ok(()));

        let indexer =
            Arc::new(LogIndexer::new(1, Some(Arc::new(mock_provider)), Arc::new(mock_db)));

        indexer.clone().sync_logs(target_block);

        // Let the background task complete
        tokio::time::sleep(tokio::time::Duration::from_millis(300)).await;
    }
}
