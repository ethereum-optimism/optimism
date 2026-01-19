use super::EventHandler;
use crate::{
    ChainProcessorError, LogIndexer, ProcessorState,
    chain_processor::Metrics,
    syncnode::{BlockProvider, ManagedNodeCommand},
};
use alloy_primitives::ChainId;
use async_trait::async_trait;
use derive_more::Constructor;
use kona_interop::{DerivedRefPair, InteropValidator};
use kona_protocol::BlockInfo;
use kona_supervisor_storage::{DerivationStorage, LogStorage, StorageError, StorageRewinder};
use std::sync::Arc;
use tokio::sync::mpsc;
use tracing::{debug, error, info, trace, warn};

/// Handler for safe blocks.
#[derive(Debug, Constructor)]
pub struct SafeBlockHandler<P, W, V> {
    chain_id: ChainId,
    managed_node_sender: mpsc::Sender<ManagedNodeCommand>,
    db_provider: Arc<W>,
    validator: Arc<V>,
    log_indexer: Arc<LogIndexer<P, W>>,
}

#[async_trait]
impl<P, W, V> EventHandler<DerivedRefPair> for SafeBlockHandler<P, W, V>
where
    P: BlockProvider + 'static,
    V: InteropValidator + 'static,
    W: LogStorage + DerivationStorage + StorageRewinder + 'static,
{
    async fn handle(
        &self,
        derived_ref_pair: DerivedRefPair,
        state: &mut ProcessorState,
    ) -> Result<BlockInfo, ChainProcessorError> {
        trace!(
            target: "supervisor::chain_processor",
            chain_id = self.chain_id,
            %derived_ref_pair,
            "Processing local safe derived block pair"
        );

        if state.is_invalidated() {
            trace!(
                target: "supervisor::chain_processor",
                chain_id = self.chain_id,
                block_number = derived_ref_pair.derived.number,
                "Invalidated block already set, skipping safe event processing"
            );
            return Ok(derived_ref_pair.derived);
        }

        let result = self.inner_handle(derived_ref_pair).await;
        Metrics::record_block_processing(self.chain_id, Metrics::BLOCK_TYPE_LOCAL_SAFE, &result);

        result
    }
}

impl<P, W, V> SafeBlockHandler<P, W, V>
where
    P: BlockProvider + 'static,
    V: InteropValidator + 'static,
    W: LogStorage + DerivationStorage + StorageRewinder + 'static,
{
    async fn inner_handle(
        &self,
        derived_ref_pair: DerivedRefPair,
    ) -> Result<BlockInfo, ChainProcessorError> {
        if self.validator.is_post_interop(self.chain_id, derived_ref_pair.derived.timestamp) {
            self.process_safe_derived_block(derived_ref_pair).await?;
            return Ok(derived_ref_pair.derived);
        }

        if self.validator.is_interop_activation_block(self.chain_id, derived_ref_pair.derived) {
            trace!(
                target: "supervisor::chain_processor",
                chain_id = self.chain_id,
                block_number = derived_ref_pair.derived.number,
                "Initialising derivation storage for interop activation block"
            );

            self.db_provider.initialise_derivation_storage(derived_ref_pair).inspect_err(
                |err| {
                    error!(
                        target: "supervisor::chain_processor::db",
                        chain_id = self.chain_id,
                        %err,
                        "Failed to initialise derivation storage for interop activation block"
                    );
                },
            )?;
            return Ok(derived_ref_pair.derived);
        }

        Ok(derived_ref_pair.derived)
    }

    async fn process_safe_derived_block(
        &self,
        derived_ref_pair: DerivedRefPair,
    ) -> Result<(), ChainProcessorError> {
        trace!(
            target: "supervisor::chain_processor",
            chain_id = self.chain_id,
            block_number = derived_ref_pair.derived.number,
            "Processing safe derived block"
        );

        match self.db_provider.save_derived_block(derived_ref_pair) {
            Ok(_) => Ok(()),
            Err(StorageError::BlockOutOfOrder) => {
                debug!(
                    target: "supervisor::chain_processor::db",
                    chain_id = self.chain_id,
                    block_number = derived_ref_pair.derived.number,
                    "Block out of order detected, resetting managed node"
                );

                self.managed_node_sender.send(ManagedNodeCommand::Reset {}).await.map_err(
                    |err| {
                        warn!(
                            target: "supervisor::chain_processor::managed_node",
                            chain_id = self.chain_id,
                            %err,
                            "Failed to send reset command to managed node"
                        );
                        ChainProcessorError::ChannelSendFailed(err.to_string())
                    },
                )?;
                Ok(())
            }
            Err(StorageError::ReorgRequired) => {
                info!(
                    target: "supervisor::chain_processor",
                    chain = self.chain_id,
                    derived_block = %derived_ref_pair.derived,
                    "Local derivation conflict detected — rewinding"
                );

                self.rewind_log_storage(&derived_ref_pair.derived).await?;
                self.retry_with_resync_derived_block(derived_ref_pair).await?;
                Ok(())
            }
            Err(StorageError::FutureData) => {
                debug!(
                    target: "supervisor::chain_processor",
                    chain = self.chain_id,
                    derived_block = %derived_ref_pair.derived,
                    "Future data detected — retrying with resync"
                );

                self.retry_with_resync_derived_block(derived_ref_pair).await
            }
            Err(err) => {
                error!(
                    target: "supervisor::chain_processor",
                    chain_id = self.chain_id,
                    block_number = derived_ref_pair.derived.number,
                    %err,
                    "Failed to save derived block pair"
                );
                Err(err.into())
            }
        }
    }

    async fn rewind_log_storage(
        &self,
        derived_block: &BlockInfo,
    ) -> Result<(), ChainProcessorError> {
        trace!(
            target: "supervisor::chain_processor",
            chain_id = self.chain_id,
            block_number = derived_block.number,
            "Rewinding log storage for derived block"
        );

        let log_block = self.db_provider.get_block(derived_block.number).inspect_err(|err| {
            warn!(
                target: "supervisor::chain_processor::db",
                chain_id = self.chain_id,
                block_number = derived_block.number,
                %err,
                "Failed to get block for rewinding log storage"
            );
        })?;

        self.db_provider.rewind_log_storage(&log_block.id()).inspect_err(|err| {
            warn!(
                target: "supervisor::chain_processor::db",
                chain_id = self.chain_id,
                block_number = derived_block.number,
                %err,
                "Failed to rewind log storage for derived block"
            );
        })?;
        Ok(())
    }

    async fn retry_with_resync_derived_block(
        &self,
        derived_ref_pair: DerivedRefPair,
    ) -> Result<(), ChainProcessorError> {
        trace!(
            target: "supervisor::chain_processor",
            chain_id = self.chain_id,
            derived_block_number = derived_ref_pair.derived.number,
            "Retrying with resync of derived block"
        );

        self.log_indexer.process_and_store_logs(&derived_ref_pair.derived).await.inspect_err(
            |err| {
                error!(
                    target: "supervisor::chain_processor::log_indexer",
                    chain_id = self.chain_id,
                    block_number = derived_ref_pair.derived.number,
                    %err,
                    "Error resyncing logs for derived block"
                );
            },
        )?;

        self.db_provider.save_derived_block(derived_ref_pair).inspect_err(|err| {
            error!(
                target: "supervisor::chain_processor::db",
                chain_id = self.chain_id,
                block_number = derived_ref_pair.derived.number,
                %err,
                "Error saving derived block after resync"
            );
        })?;

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::syncnode::{
        BlockProvider, ManagedNodeController, ManagedNodeDataProvider, ManagedNodeError,
    };
    use alloy_primitives::B256;
    use alloy_rpc_types_eth::BlockNumHash;
    use async_trait::async_trait;
    use kona_interop::{DerivedRefPair, InteropValidationError};
    use kona_protocol::BlockInfo;
    use kona_supervisor_storage::{
        DerivationStorageReader, DerivationStorageWriter, HeadRefStorageWriter, LogStorageReader,
        LogStorageWriter, StorageError,
    };
    use kona_supervisor_types::{BlockSeal, Log, OutputV0, Receipts};
    use mockall::mock;

    mock!(
        #[derive(Debug)]
        pub Node {}

        #[async_trait]
        impl ManagedNodeDataProvider for Node {
            async fn output_v0_at_timestamp(
                &self,
                _timestamp: u64,
            ) -> Result<OutputV0, ManagedNodeError>;

            async fn pending_output_v0_at_timestamp(
                &self,
                _timestamp: u64,
            ) -> Result<OutputV0, ManagedNodeError>;

            async fn l2_block_ref_by_timestamp(
                &self,
                _timestamp: u64,
            ) -> Result<BlockInfo, ManagedNodeError>;
        }

        #[async_trait]
        impl BlockProvider for Node {
            async fn fetch_receipts(&self, _block_hash: B256) -> Result<Receipts, ManagedNodeError>;
            async fn block_by_number(&self, _number: u64) -> Result<BlockInfo, ManagedNodeError>;
        }

        #[async_trait]
        impl ManagedNodeController for Node {
            async fn update_finalized(
                &self,
                _finalized_block_id: BlockNumHash,
            ) -> Result<(), ManagedNodeError>;

            async fn update_cross_unsafe(
                &self,
                cross_unsafe_block_id: BlockNumHash,
            ) -> Result<(), ManagedNodeError>;

            async fn update_cross_safe(
                &self,
                source_block_id: BlockNumHash,
                derived_block_id: BlockNumHash,
            ) -> Result<(), ManagedNodeError>;

            async fn reset(&self) -> Result<(), ManagedNodeError>;

            async fn invalidate_block(&self, seal: BlockSeal) -> Result<(), ManagedNodeError>;
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

        impl DerivationStorageReader for Db {
            fn derived_to_source(&self, derived_block_id: BlockNumHash) -> Result<BlockInfo, StorageError>;
            fn latest_derived_block_at_source(&self, source_block_id: BlockNumHash) -> Result<BlockInfo, StorageError>;
            fn latest_derivation_state(&self) -> Result<DerivedRefPair, StorageError>;
            fn get_source_block(&self, source_block_number: u64) -> Result<BlockInfo, StorageError>;
            fn get_activation_block(&self) -> Result<BlockInfo, StorageError>;
        }

        impl DerivationStorageWriter for Db {
            fn initialise_derivation_storage(
                &self,
                incoming_pair: DerivedRefPair,
            ) -> Result<(), StorageError>;

            fn save_derived_block(
                &self,
                incoming_pair: DerivedRefPair,
            ) -> Result<(), StorageError>;

            fn save_source_block(
                &self,
                source: BlockInfo,
            ) -> Result<(), StorageError>;
        }

        impl HeadRefStorageWriter for Db {
            fn update_finalized_using_source(
                &self,
                block_info: BlockInfo,
            ) -> Result<BlockInfo, StorageError>;

            fn update_current_cross_unsafe(
                &self,
                block: &BlockInfo,
            ) -> Result<(), StorageError>;

            fn update_current_cross_safe(
                &self,
                block: &BlockInfo,
            ) -> Result<DerivedRefPair, StorageError>;
        }

        impl StorageRewinder for Db {
            fn rewind_log_storage(&self, to: &BlockNumHash) -> Result<(), StorageError>;
            fn rewind(&self, to: &BlockNumHash) -> Result<(), StorageError>;
            fn rewind_to_source(&self, to: &BlockNumHash) -> Result<Option<BlockInfo>, StorageError>;
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
    async fn test_handle_derived_event_skips_if_invalidated() {
        let mockdb = MockDb::new();
        let mockvalidator = MockValidator::new();
        let (tx, mut rx) = mpsc::channel(1);
        let mocknode = MockNode::new();
        let mut state = ProcessorState::new();

        // Simulate invalidated state
        state.set_invalidated(DerivedRefPair {
            source: BlockInfo {
                number: 1,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 0,
            },
            derived: BlockInfo {
                number: 2,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 0,
            },
        });

        let block_pair = DerivedRefPair {
            source: BlockInfo {
                number: 123,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 0,
            },
            derived: BlockInfo {
                number: 1234,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 1003,
            },
        };

        let writer = Arc::new(mockdb);
        let managed_node = Arc::new(mocknode);
        let log_indexer = Arc::new(LogIndexer::new(1, Some(managed_node.clone()), writer.clone()));

        let handler = SafeBlockHandler::new(1, tx, writer, Arc::new(mockvalidator), log_indexer);

        let result = handler.handle(block_pair, &mut state).await;
        assert!(result.is_ok());

        // Ensure no command was sent
        assert!(rx.try_recv().is_err());
    }

    #[tokio::test]
    async fn test_handle_derived_event_pre_interop() {
        let mockdb = MockDb::new();
        let mut mockvalidator = MockValidator::new();
        let (tx, mut rx) = mpsc::channel(1);
        let mocknode = MockNode::new();
        let mut state = ProcessorState::new();

        mockvalidator.expect_is_post_interop().returning(|_, _| false);
        mockvalidator.expect_is_interop_activation_block().returning(|_, _| false);

        let block_pair = DerivedRefPair {
            source: BlockInfo {
                number: 123,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 0,
            },
            derived: BlockInfo {
                number: 1234,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 999,
            },
        };

        let writer = Arc::new(mockdb);
        let managed_node = Arc::new(mocknode);
        // Create a mock log indexer
        let log_indexer = Arc::new(LogIndexer::new(1, Some(managed_node.clone()), writer.clone()));

        let handler = SafeBlockHandler::new(
            1, // chain_id
            tx,
            writer,
            Arc::new(mockvalidator),
            log_indexer,
        );

        let result = handler.handle(block_pair, &mut state).await;
        assert!(result.is_ok());

        // Ensure no command was sent
        assert!(rx.try_recv().is_err());
    }

    #[tokio::test]
    async fn test_handle_derived_event_post_interop() {
        let mut mockdb = MockDb::new();
        let mut mockvalidator = MockValidator::new();
        let (tx, mut rx) = mpsc::channel(1);
        let mocknode = MockNode::new();
        let mut state = ProcessorState::new();

        mockvalidator.expect_is_post_interop().returning(|_, _| true);

        let block_pair = DerivedRefPair {
            source: BlockInfo {
                number: 123,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 0,
            },
            derived: BlockInfo {
                number: 1234,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 1003,
            },
        };

        mockdb.expect_save_derived_block().returning(move |_pair: DerivedRefPair| {
            assert_eq!(_pair, block_pair);
            Ok(())
        });

        let writer = Arc::new(mockdb);
        let managed_node = Arc::new(mocknode);
        // Create a mock log indexer
        let log_indexer = Arc::new(LogIndexer::new(1, Some(managed_node.clone()), writer.clone()));

        let handler = SafeBlockHandler::new(
            1, // chain_id
            tx,
            writer,
            Arc::new(mockvalidator),
            log_indexer,
        );

        let result = handler.handle(block_pair, &mut state).await;
        assert!(result.is_ok());

        // Ensure no command was sent
        assert!(rx.try_recv().is_err());
    }

    #[tokio::test]
    async fn test_handle_derived_event_interop_activation() {
        let mut mockdb = MockDb::new();
        let mut mockvalidator = MockValidator::new();
        let (tx, mut rx) = mpsc::channel(1);
        let mocknode = MockNode::new();
        let mut state = ProcessorState::new();

        mockvalidator.expect_is_post_interop().returning(|_, _| false);
        mockvalidator.expect_is_interop_activation_block().returning(|_, _| true);

        let block_pair = DerivedRefPair {
            source: BlockInfo {
                number: 123,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 0,
            },
            derived: BlockInfo {
                number: 1234,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 1001,
            },
        };

        mockdb.expect_initialise_derivation_storage().returning(move |_pair: DerivedRefPair| {
            assert_eq!(_pair, block_pair);
            Ok(())
        });

        let writer = Arc::new(mockdb);
        let managed_node = Arc::new(mocknode);
        // Create a mock log indexer
        let log_indexer = Arc::new(LogIndexer::new(1, Some(managed_node.clone()), writer.clone()));

        let handler = SafeBlockHandler::new(
            1, // chain_id
            tx,
            writer,
            Arc::new(mockvalidator),
            log_indexer,
        );

        let result = handler.handle(block_pair, &mut state).await;
        assert!(result.is_ok());

        // Ensure no command was sent
        assert!(rx.try_recv().is_err());
    }

    #[tokio::test]
    async fn test_handle_derived_event_block_out_of_order_triggers_reset() {
        let mut mockdb = MockDb::new();
        let mut mockvalidator = MockValidator::new();
        let (tx, mut rx) = mpsc::channel(1);
        let mut mocknode = MockNode::new();
        let mut state = ProcessorState::new();

        mockvalidator.expect_is_post_interop().returning(|_, _| true);

        let block_pair = DerivedRefPair {
            source: BlockInfo {
                number: 123,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 0,
            },
            derived: BlockInfo {
                number: 1234,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 1003, // post-interop
            },
        };

        // Simulate BlockOutOfOrder error
        mockdb
            .expect_save_derived_block()
            .returning(move |_pair: DerivedRefPair| Err(StorageError::BlockOutOfOrder));

        // Expect reset to be called
        mocknode.expect_reset().returning(|| Ok(()));

        let writer = Arc::new(mockdb);
        let managed_node = Arc::new(mocknode);
        // Create a mock log indexer
        let log_indexer = Arc::new(LogIndexer::new(1, Some(managed_node.clone()), writer.clone()));

        let handler = SafeBlockHandler::new(
            1, // chain_id
            tx,
            writer,
            Arc::new(mockvalidator),
            log_indexer,
        );
        let result = handler.handle(block_pair, &mut state).await;
        assert!(result.is_ok());

        // Ensure reset command was sent
        if let Some(cmd) = rx.recv().await {
            assert!(matches!(cmd, ManagedNodeCommand::Reset {}));
        } else {
            panic!("Expected reset command to be sent");
        }
    }

    #[tokio::test]
    async fn test_handle_derived_event_block_out_of_order_triggers_reset_error() {
        let mut mockdb = MockDb::new();
        let mut mockvalidator = MockValidator::new();
        let (tx, rx) = mpsc::channel(1);
        let mocknode = MockNode::new();
        let mut state = ProcessorState::new();

        mockvalidator.expect_is_post_interop().returning(|_, _| true);

        let block_pair = DerivedRefPair {
            source: BlockInfo {
                number: 123,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 0,
            },
            derived: BlockInfo {
                number: 1234,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 1003, // post-interop
            },
        };

        // Simulate BlockOutOfOrder error
        mockdb
            .expect_save_derived_block()
            .returning(move |_pair: DerivedRefPair| Err(StorageError::BlockOutOfOrder));

        let writer = Arc::new(mockdb);
        let managed_node = Arc::new(mocknode);

        // Create a mock log indexer
        let log_indexer = Arc::new(LogIndexer::new(1, Some(managed_node), writer.clone()));

        drop(rx); // Simulate a send error by dropping the receiver

        let handler = SafeBlockHandler::new(
            1, // chain_id
            tx,
            writer,
            Arc::new(mockvalidator),
            log_indexer,
        );

        let result = handler.handle(block_pair, &mut state).await;
        assert!(result.is_err());
    }

    #[tokio::test]
    async fn test_handle_derived_event_block_triggers_reorg() {
        let mut mockdb = MockDb::new();
        let mut mockvalidator = MockValidator::new();
        let (tx, mut rx) = mpsc::channel(1);
        let mut mocknode = MockNode::new();
        let mut state = ProcessorState::new();

        mockvalidator.expect_is_post_interop().returning(|_, _| true);

        let block_pair = DerivedRefPair {
            source: BlockInfo {
                number: 123,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 0,
            },
            derived: BlockInfo {
                number: 1234,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 1003, // post-interop
            },
        };

        let mut seq = mockall::Sequence::new();
        // Simulate ReorgRequired error
        mockdb
            .expect_save_derived_block()
            .times(1)
            .in_sequence(&mut seq)
            .returning(move |_pair: DerivedRefPair| Err(StorageError::ReorgRequired));

        mockdb.expect_get_block().returning(move |num| {
            Ok(BlockInfo {
                number: num,
                hash: B256::random(), // different hash from safe derived block
                parent_hash: B256::ZERO,
                timestamp: 1003, // post-interop
            })
        });

        // Expect reorg on log storage
        mockdb.expect_rewind_log_storage().returning(|_block_id| Ok(()));
        mockdb.expect_store_block_logs().returning(|_block_id, _logs| Ok(()));
        mocknode.expect_fetch_receipts().returning(|_receipts| Ok(Receipts::default()));

        mockdb
            .expect_save_derived_block()
            .times(1)
            .in_sequence(&mut seq)
            .returning(move |_pair: DerivedRefPair| Ok(()));

        let writer = Arc::new(mockdb);
        let managed_node = Arc::new(mocknode);
        // Create a mock log indexer
        let log_indexer = Arc::new(LogIndexer::new(1, Some(managed_node.clone()), writer.clone()));

        let handler = SafeBlockHandler::new(
            1, // chain_id
            tx,
            writer,
            Arc::new(mockvalidator),
            log_indexer,
        );
        let result = handler.handle(block_pair, &mut state).await;
        assert!(result.is_ok());

        // Ensure no command was sent
        assert!(rx.try_recv().is_err());
    }

    #[tokio::test]
    async fn test_handle_derived_event_block_triggers_reorg_block_error() {
        let mut mockdb = MockDb::new();
        let mut mockvalidator = MockValidator::new();
        let (tx, mut rx) = mpsc::channel(1);
        let mocknode = MockNode::new();
        let mut state = ProcessorState::new();

        mockvalidator.expect_is_post_interop().returning(|_, _| true);

        let block_pair = DerivedRefPair {
            source: BlockInfo {
                number: 123,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 0,
            },
            derived: BlockInfo {
                number: 1234,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 1003, // post-interop
            },
        };

        let mut seq = mockall::Sequence::new();
        // Simulate ReorgRequired error
        mockdb
            .expect_save_derived_block()
            .times(1)
            .in_sequence(&mut seq)
            .returning(move |_pair: DerivedRefPair| Err(StorageError::ReorgRequired));

        mockdb.expect_get_block().returning(move |_| Err(StorageError::DatabaseNotInitialised));

        let writer = Arc::new(mockdb);
        let managed_node = Arc::new(mocknode);
        // Create a mock log indexer
        let log_indexer = Arc::new(LogIndexer::new(1, Some(managed_node.clone()), writer.clone()));

        let handler = SafeBlockHandler::new(
            1, // chain_id
            tx,
            writer,
            Arc::new(mockvalidator),
            log_indexer,
        );
        let result = handler.handle(block_pair, &mut state).await.unwrap_err();
        assert!(matches!(
            result,
            ChainProcessorError::StorageError(StorageError::DatabaseNotInitialised)
        ));

        // Ensure no command was sent
        assert!(rx.try_recv().is_err());
    }

    #[tokio::test]
    async fn test_handle_derived_event_block_triggers_reorg_rewind_error() {
        let mut mockdb = MockDb::new();
        let mut mockvalidator = MockValidator::new();
        let (tx, mut rx) = mpsc::channel(1);
        let mocknode = MockNode::new();
        let mut state = ProcessorState::new();

        mockvalidator.expect_is_post_interop().returning(|_, _| true);

        let block_pair = DerivedRefPair {
            source: BlockInfo {
                number: 123,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 0,
            },
            derived: BlockInfo {
                number: 1234,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 1003, // post-interop
            },
        };

        let mut seq = mockall::Sequence::new();
        // Simulate ReorgRequired error
        mockdb
            .expect_save_derived_block()
            .times(1)
            .in_sequence(&mut seq)
            .returning(move |_pair: DerivedRefPair| Err(StorageError::ReorgRequired));

        mockdb.expect_get_block().returning(move |num| {
            Ok(BlockInfo {
                number: num,
                hash: B256::random(), // different hash from safe derived block
                parent_hash: B256::ZERO,
                timestamp: 1003, // post-interop
            })
        });

        // Expect reorg on log storage
        mockdb
            .expect_rewind_log_storage()
            .returning(|_block_id| Err(StorageError::DatabaseNotInitialised));

        let writer = Arc::new(mockdb);
        let managed_node = Arc::new(mocknode);
        // Create a mock log indexer
        let log_indexer = Arc::new(LogIndexer::new(1, Some(managed_node.clone()), writer.clone()));

        let handler = SafeBlockHandler::new(
            1, // chain_id
            tx,
            writer,
            Arc::new(mockvalidator),
            log_indexer,
        );
        let result = handler.handle(block_pair, &mut state).await;
        assert!(matches!(
            result,
            Err(ChainProcessorError::StorageError(StorageError::DatabaseNotInitialised))
        ));

        // Ensure no command was sent
        assert!(rx.try_recv().is_err());
    }

    #[tokio::test]
    async fn test_handle_derived_event_block_triggers_resync() {
        let mut mockdb = MockDb::new();
        let mut mockvalidator = MockValidator::new();
        let (tx, mut rx) = mpsc::channel(1);
        let mut mocknode = MockNode::new();
        let mut state = ProcessorState::new();

        mockvalidator.expect_is_post_interop().returning(|_, _| true);

        let block_pair = DerivedRefPair {
            source: BlockInfo {
                number: 123,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 0,
            },
            derived: BlockInfo {
                number: 1234,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 1003, // post-interop
            },
        };

        let mut seq = mockall::Sequence::new();
        // Simulate ReorgRequired error
        mockdb
            .expect_save_derived_block()
            .times(1)
            .in_sequence(&mut seq)
            .returning(move |_pair: DerivedRefPair| Err(StorageError::FutureData));

        mockdb.expect_get_block().returning(move |num| {
            Ok(BlockInfo {
                number: num,
                hash: B256::random(), // different hash from safe derived block
                parent_hash: B256::ZERO,
                timestamp: 1003, // post-interop
            })
        });

        mockdb.expect_store_block_logs().returning(|_block_id, _logs| Ok(()));

        mocknode.expect_fetch_receipts().returning(|_receipts| Ok(Receipts::default()));

        mockdb
            .expect_save_derived_block()
            .times(1)
            .in_sequence(&mut seq)
            .returning(move |_pair: DerivedRefPair| Ok(()));

        let writer = Arc::new(mockdb);
        let managed_node = Arc::new(mocknode);
        // Create a mock log indexer
        let log_indexer = Arc::new(LogIndexer::new(1, Some(managed_node.clone()), writer.clone()));

        let handler = SafeBlockHandler::new(
            1, // chain_id
            tx,
            writer,
            Arc::new(mockvalidator),
            log_indexer,
        );
        let result = handler.handle(block_pair, &mut state).await;
        assert!(result.is_ok());

        // Ensure no command was sent
        assert!(rx.try_recv().is_err());
    }

    #[tokio::test]
    async fn test_handle_derived_event_other_error() {
        let mut mockdb = MockDb::new();
        let mut mockvalidator = MockValidator::new();
        let (tx, mut rx) = mpsc::channel(1);
        let mocknode = MockNode::new();
        let mut state = ProcessorState::new();

        mockvalidator.expect_is_post_interop().returning(|_, _| true);

        let block_pair = DerivedRefPair {
            source: BlockInfo {
                number: 123,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 0,
            },
            derived: BlockInfo {
                number: 1234,
                hash: B256::ZERO,
                parent_hash: B256::ZERO,
                timestamp: 1003, // post-interop
            },
        };

        // Simulate a different error
        mockdb
            .expect_save_derived_block()
            .returning(move |_pair: DerivedRefPair| Err(StorageError::DatabaseNotInitialised));

        let writer = Arc::new(mockdb);
        let managed_node = Arc::new(mocknode);
        // Create a mock log indexer
        let log_indexer = Arc::new(LogIndexer::new(1, Some(managed_node.clone()), writer.clone()));

        let handler = SafeBlockHandler::new(
            1, // chain_id
            tx,
            writer,
            Arc::new(mockvalidator),
            log_indexer,
        );
        let result = handler.handle(block_pair, &mut state).await;
        assert!(result.is_err());

        // Ensure no command was sent
        assert!(rx.try_recv().is_err());
    }
}
