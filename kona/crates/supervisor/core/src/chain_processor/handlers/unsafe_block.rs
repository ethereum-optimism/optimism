use super::EventHandler;
use crate::{
    ChainProcessorError, LogIndexer, ProcessorState, chain_processor::Metrics,
    syncnode::BlockProvider,
};
use alloy_primitives::ChainId;
use async_trait::async_trait;
use derive_more::Constructor;
use kona_interop::InteropValidator;
use kona_protocol::BlockInfo;
use kona_supervisor_storage::LogStorage;
use std::sync::Arc;
use tracing::{error, trace};

/// Handler for unsafe blocks.
/// This handler processes unsafe blocks by syncing logs and initializing log storage.
#[derive(Debug, Constructor)]
pub struct UnsafeBlockHandler<P, W, V> {
    chain_id: ChainId,
    validator: Arc<V>,
    db_provider: Arc<W>,
    log_indexer: Arc<LogIndexer<P, W>>,
}

#[async_trait]
impl<P, W, V> EventHandler<BlockInfo> for UnsafeBlockHandler<P, W, V>
where
    P: BlockProvider + 'static,
    V: InteropValidator + 'static,
    W: LogStorage + 'static,
{
    async fn handle(
        &self,
        block: BlockInfo,
        state: &mut ProcessorState,
    ) -> Result<BlockInfo, ChainProcessorError> {
        trace!(
            target: "supervisor::chain_processor",
            chain_id = self.chain_id,
            block_number = block.number,
            "Processing unsafe block"
        );

        if state.is_invalidated() {
            trace!(
                target: "supervisor::chain_processor",
                chain_id = self.chain_id,
                block_number = block.number,
                "Invalidated block already set, skipping unsafe event processing"
            );
            return Ok(block);
        }

        let result = self.inner_handle(block).await;
        Metrics::record_block_processing(self.chain_id, Metrics::BLOCK_TYPE_LOCAL_UNSAFE, &result);

        result
    }
}

impl<P, W, V> UnsafeBlockHandler<P, W, V>
where
    P: BlockProvider + 'static,
    V: InteropValidator + 'static,
    W: LogStorage + 'static,
{
    async fn inner_handle(&self, block: BlockInfo) -> Result<BlockInfo, ChainProcessorError> {
        if self.validator.is_post_interop(self.chain_id, block.timestamp) {
            self.log_indexer.clone().sync_logs(block);
            return Ok(block);
        }

        if self.validator.is_interop_activation_block(self.chain_id, block) {
            trace!(
                target: "supervisor::chain_processor",
                chain_id = self.chain_id,
                block_number = block.number,
                "Initialising log storage for interop activation block"
            );

            self.db_provider.initialise_log_storage(block).inspect_err(|err| {
                error!(
                    target: "supervisor::chain_processor::db",
                    chain_id = self.chain_id,
                    %block,
                    %err,
                    "Failed to initialise log storage for interop activation block"
                );
            })?;
            return Ok(block);
        }

        Ok(block)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        ProcessorState,
        syncnode::{BlockProvider, ManagedNodeError},
    };
    use alloy_primitives::B256;
    use kona_interop::{DerivedRefPair, InteropValidationError};
    use kona_protocol::BlockInfo;
    use kona_supervisor_storage::{LogStorageReader, LogStorageWriter, StorageError};
    use kona_supervisor_types::{Log, Receipts};
    use mockall::mock;
    use std::sync::Arc;

    mock!(
        #[derive(Debug)]
        pub Node {}

        #[async_trait]
        impl BlockProvider for Node {
            async fn fetch_receipts(&self, _block_hash: B256) -> Result<Receipts, ManagedNodeError>;
            async fn block_by_number(&self, _number: u64) -> Result<BlockInfo, ManagedNodeError>;
        }
    );

    mock!(
        #[derive(Debug)]
        pub Db {}

        impl LogStorageWriter for Db {
            fn initialise_log_storage(
                &self,
                block: BlockInfo,
            ) -> Result<(), StorageError>;

            fn store_block_logs(
                &self,
                block: &BlockInfo,
                logs: Vec<Log>,
            ) -> Result<(), StorageError>;
        }

        impl LogStorageReader for Db {
            fn get_block(&self, block_number: u64) -> Result<BlockInfo, StorageError>;
            fn get_latest_block(&self) -> Result<BlockInfo, StorageError>;
            fn get_log(&self,block_number: u64,log_index: u32) -> Result<Log, StorageError>;
            fn get_logs(&self, block_number: u64) -> Result<Vec<Log>, StorageError>;
        }
    );

    mock! (
        #[derive(Debug)]
        pub Validator {}

        impl InteropValidator for Validator {
            fn validate_interop_timestamps(
                &self,
                initiating_chain_id: ChainId,
                initiating_timestamp: u64,
                executing_chain_id: ChainId,
                executing_timestamp: u64,
                timeout: Option<u64>,
            ) -> Result<(), InteropValidationError>;

            fn is_post_interop(&self, chain_id: ChainId, timestamp: u64) -> bool;

            fn is_interop_activation_block(&self, chain_id: ChainId, block: BlockInfo) -> bool;
        }
    );

    #[tokio::test]
    async fn test_handle_unsafe_event_skips_if_invalidated() {
        let mockdb = MockDb::new();
        let mockvalidator = MockValidator::new();
        let mocknode = MockNode::new();
        let mut state = ProcessorState::new();

        // Simulate invalidated state
        state.set_invalidated(DerivedRefPair {
            source: BlockInfo::new(B256::ZERO, 1, B256::ZERO, 0),
            derived: BlockInfo::new(B256::ZERO, 2, B256::ZERO, 0),
        });

        let writer = Arc::new(mockdb);
        let managed_node = Arc::new(mocknode);
        let log_indexer = Arc::new(LogIndexer::new(1, Some(managed_node), writer.clone()));

        let handler = UnsafeBlockHandler::new(1, Arc::new(mockvalidator), writer, log_indexer);

        let block = BlockInfo::new(B256::ZERO, 123, B256::ZERO, 10);
        let result = handler.handle(block, &mut state).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_handle_unsafe_event_pre_interop() {
        let mockdb = MockDb::new();
        let mut mockvalidator = MockValidator::new();
        let mocknode = MockNode::new();
        let mut state = ProcessorState::new();

        mockvalidator.expect_is_post_interop().returning(|_, _| false);
        mockvalidator.expect_is_interop_activation_block().returning(|_, _| false);

        let writer = Arc::new(mockdb);
        let managed_node = Arc::new(mocknode);
        // Create a mock log indexer
        let log_indexer = Arc::new(LogIndexer::new(1, Some(managed_node), writer.clone()));

        let handler = UnsafeBlockHandler::new(
            1, // chain_id
            Arc::new(mockvalidator),
            writer,
            log_indexer,
        );

        // Pre-interop block (timestamp < 1000)
        let block = BlockInfo::new(B256::ZERO, 123, B256::ZERO, 10);

        let result = handler.handle(block, &mut state).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_handle_unsafe_event_post_interop() {
        let mut mockdb = MockDb::new();
        let mut mockvalidator = MockValidator::new();
        let mut mocknode = MockNode::new();
        let mut state = ProcessorState::new();

        mockvalidator.expect_is_post_interop().returning(|_, _| true);

        // Send unsafe block event
        let block = BlockInfo::new(B256::ZERO, 123, B256::ZERO, 1003);

        mockdb.expect_store_block_logs().returning(move |_block, _log| Ok(()));
        mocknode.expect_fetch_receipts().returning(move |block_hash| {
            assert!(block_hash == block.hash);
            Ok(Receipts::default())
        });

        let writer = Arc::new(mockdb);
        let managed_node = Arc::new(mocknode);
        // Create a mock log indexer
        let log_indexer = Arc::new(LogIndexer::new(1, Some(managed_node), writer.clone()));

        let handler = UnsafeBlockHandler::new(
            1, // chain_id
            Arc::new(mockvalidator),
            writer,
            log_indexer,
        );

        let result = handler.handle(block, &mut state).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_handle_unsafe_event_interop_activation() {
        let mut mockdb = MockDb::new();
        let mut mockvalidator = MockValidator::new();
        let mocknode = MockNode::new();
        let mut state = ProcessorState::new();

        mockvalidator.expect_is_post_interop().returning(|_, _| false);
        mockvalidator.expect_is_interop_activation_block().returning(|_, _| true);

        // Block that triggers interop activation
        let block = BlockInfo::new(B256::ZERO, 123, B256::ZERO, 1001);

        mockdb.expect_initialise_log_storage().times(1).returning(move |b| {
            assert_eq!(b, block);
            Ok(())
        });

        let writer = Arc::new(mockdb);
        let managed_node = Arc::new(mocknode);
        // Create a mock log indexer
        let log_indexer = Arc::new(LogIndexer::new(1, Some(managed_node), writer.clone()));

        let handler = UnsafeBlockHandler::new(
            1, // chain_id
            Arc::new(mockvalidator),
            writer,
            log_indexer,
        );

        let result = handler.handle(block, &mut state).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_handle_unsafe_event_interop_activation_init_fails() {
        let mut mockdb = MockDb::new();
        let mut mockvalidator = MockValidator::new();
        let mocknode = MockNode::new();
        let mut state = ProcessorState::new();

        mockvalidator.expect_is_post_interop().returning(|_, _| false);
        mockvalidator.expect_is_interop_activation_block().returning(|_, _| true);

        let block = BlockInfo::new(B256::ZERO, 123, B256::ZERO, 1001);

        mockdb
            .expect_initialise_log_storage()
            .times(1)
            .returning(move |_b| Err(StorageError::ConflictError));

        let writer = Arc::new(mockdb);
        let managed_node = Arc::new(mocknode);
        let log_indexer = Arc::new(LogIndexer::new(1, Some(managed_node), writer.clone()));

        let handler = UnsafeBlockHandler::new(1, Arc::new(mockvalidator), writer, log_indexer);

        let result = handler.handle(block, &mut state).await;
        assert!(result.is_err());
    }
}
