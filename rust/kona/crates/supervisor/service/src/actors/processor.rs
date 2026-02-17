use async_trait::async_trait;
use kona_interop::InteropValidator;
use kona_supervisor_core::{ChainProcessor, event::ChainEvent, syncnode::BlockProvider};
use kona_supervisor_storage::{
    DerivationStorage, HeadRefStorageWriter, LogStorage, StorageRewinder,
};
use thiserror::Error;
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;
use tracing::info;

use crate::SupervisorActor;

/// Represents an actor that processes chain events using the [`ChainProcessor`].
/// It listens for [`ChainEvent`]s and handles them accordingly.
#[derive(Debug)]
pub struct ChainProcessorActor<P, W, V> {
    chain_processor: ChainProcessor<P, W, V>,
    cancel_token: CancellationToken,
    event_rx: mpsc::Receiver<ChainEvent>,
}

impl<P, W, V> ChainProcessorActor<P, W, V>
where
    P: BlockProvider + 'static,
    V: InteropValidator + 'static,
    W: LogStorage + DerivationStorage + HeadRefStorageWriter + StorageRewinder + 'static,
{
    /// Creates a new [`ChainProcessorActor`].
    pub const fn new(
        chain_processor: ChainProcessor<P, W, V>,
        cancel_token: CancellationToken,
        event_rx: mpsc::Receiver<ChainEvent>,
    ) -> Self {
        Self { chain_processor, cancel_token, event_rx }
    }
}

#[async_trait]
impl<P, W, V> SupervisorActor for ChainProcessorActor<P, W, V>
where
    P: BlockProvider + 'static,
    V: InteropValidator + 'static,
    W: LogStorage + DerivationStorage + HeadRefStorageWriter + StorageRewinder + 'static,
{
    type InboundEvent = ChainEvent;
    type Error = ChainProcessorActorError;

    async fn start(mut self) -> Result<(), Self::Error> {
        info!(
            target: "supervisor::chain_processor_actor",
            "Starting ChainProcessorActor"
        );

        loop {
            tokio::select! {
                maybe_event = self.event_rx.recv() => {
                    if let Some(event) = maybe_event {
                        self.chain_processor.handle_event(event).await;
                    } else {
                        info!(
                            target: "supervisor::chain_processor_actor",
                            "Chain event receiver closed, stopping ChainProcessorActor"
                        );
                        return Err(ChainProcessorActorError::ReceiverClosed);
                    }
                }
                _ = self.cancel_token.cancelled() => {
                    info!(
                        target: "supervisor::chain_processor_actor",
                        "ChainProcessorActor cancellation requested, stopping..."
                    );
                    break;
                }
            }
        }

        Ok(())
    }
}

#[derive(Debug, Error)]
pub enum ChainProcessorActorError {
    /// Error when the chain event receiver is closed.
    #[error("Chain event receiver closed")]
    ReceiverClosed,
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::SupervisorActor;
    use alloy_eips::BlockNumHash;
    use alloy_primitives::{B256, ChainId};
    use kona_interop::{DerivedRefPair, InteropValidationError};
    use kona_protocol::BlockInfo;
    use kona_supervisor_core::{
        LogIndexer,
        syncnode::{BlockProvider, ManagedNodeCommand, ManagedNodeDataProvider, ManagedNodeError},
    };
    use kona_supervisor_storage::{
        DerivationStorageReader, DerivationStorageWriter, HeadRefStorageWriter, LogStorageReader,
        LogStorageWriter, StorageError, StorageRewinder,
    };
    use kona_supervisor_types::{Log, OutputV0, Receipts};
    use mockall::{mock, predicate::*};
    use std::sync::Arc;
    use tokio::sync::mpsc;
    use tokio_util::sync::CancellationToken;

    mock!(
        #[derive(Debug)]
        pub Node {}

        #[async_trait]
        impl BlockProvider for Node {
            async fn fetch_receipts(&self, _block_hash: B256) -> Result<Receipts, ManagedNodeError>;
            async fn block_by_number(&self, _number: u64) -> Result<BlockInfo, ManagedNodeError>;
        }

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

    mock!(
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
    async fn test_actor_handles_event() {
        let mock_node = MockNode::new();
        let mock_db = MockDb::new();
        let validator = MockValidator::new();
        let (mn_sender, mut mn_receiver) = mpsc::channel(1);

        let db = Arc::new(mock_db);
        let log_indexer = LogIndexer::new(1, Some(Arc::new(mock_node)), db.clone());

        let processor =
            ChainProcessor::new(Arc::new(validator), 1, Arc::new(log_indexer), db, mn_sender);

        let cancel_token = CancellationToken::new();
        let (tx, rx) = mpsc::channel(1);

        let actor = ChainProcessorActor::new(processor, cancel_token.clone(), rx);

        // Send an event
        let block = BlockInfo {
            number: 1,
            hash: B256::from([0; 32]),
            timestamp: 1000,
            ..Default::default()
        };
        tx.send(ChainEvent::CrossUnsafeUpdate { block }).await.unwrap();

        // Cancel after a short delay to exit the loop
        let cancel = cancel_token.clone();
        tokio::spawn(async move {
            tokio::time::sleep(std::time::Duration::from_millis(50)).await;
            cancel.cancel();
        });

        let result = actor.start().await;
        assert!(result.is_ok());

        if let Some(ManagedNodeCommand::UpdateCrossUnsafe { block_id }) = mn_receiver.recv().await {
            assert_eq!(block_id, block.id());
        } else {
            panic!("Expected UpdateCrossUnsafe command");
        }
    }

    #[tokio::test]
    async fn test_actor_receiver_closed() {
        let mock_node = MockNode::new();
        let mock_db = MockDb::new();
        let validator = MockValidator::new();
        let (mn_sender, _mn_receiver) = mpsc::channel(1);

        let db = Arc::new(mock_db);
        let log_indexer = LogIndexer::new(1, Some(Arc::new(mock_node)), db.clone());

        let processor =
            ChainProcessor::new(Arc::new(validator), 1, Arc::new(log_indexer), db, mn_sender);

        let cancel_token = CancellationToken::new();
        let (tx, rx) = mpsc::channel::<ChainEvent>(1); // No sender, so channel is closed
        drop(tx);

        let actor = ChainProcessorActor::new(processor, cancel_token, rx);

        let result = actor.start().await;
        assert!(matches!(result, Err(ChainProcessorActorError::ReceiverClosed)));
    }

    #[tokio::test]
    async fn test_actor_cancellation() {
        let mock_node = MockNode::new();
        let mock_db = MockDb::new();
        let validator = MockValidator::new();
        let (mn_sender, _mn_receiver) = mpsc::channel(1);

        let db = Arc::new(mock_db);
        let log_indexer = LogIndexer::new(1, Some(Arc::new(mock_node)), db.clone());

        let processor =
            ChainProcessor::new(Arc::new(validator), 1, Arc::new(log_indexer), db, mn_sender);

        let cancel_token = CancellationToken::new();
        let (_tx, rx) = mpsc::channel::<ChainEvent>(1);

        let actor = ChainProcessorActor::new(processor, cancel_token.clone(), rx);

        // Cancel immediately
        cancel_token.cancel();

        let result = actor.start().await;
        assert!(result.is_ok());
    }
}
