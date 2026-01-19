use super::EventHandler;
use crate::{ChainProcessorError, ProcessorState, syncnode::ManagedNodeCommand};
use alloy_primitives::ChainId;
use async_trait::async_trait;
use derive_more::Constructor;
use kona_protocol::BlockInfo;
use kona_supervisor_storage::{DerivationStorageWriter, StorageError};
use std::sync::Arc;
use tokio::sync::mpsc;
use tracing::{debug, error, trace, warn};

/// Handler for origin updates in the chain.
#[derive(Debug, Constructor)]
pub struct OriginHandler<W> {
    chain_id: ChainId,
    managed_node_sender: mpsc::Sender<ManagedNodeCommand>,
    db_provider: Arc<W>,
}

#[async_trait]
impl<W> EventHandler<BlockInfo> for OriginHandler<W>
where
    W: DerivationStorageWriter + Send + Sync + 'static,
{
    async fn handle(
        &self,
        origin: BlockInfo,
        state: &mut ProcessorState,
    ) -> Result<BlockInfo, ChainProcessorError> {
        trace!(
            target: "supervisor::chain_processor",
            chain_id = self.chain_id,
            %origin,
            "Processing derivation origin update"
        );

        if state.is_invalidated() {
            trace!(
                target: "supervisor::chain_processor",
                chain_id = self.chain_id,
                %origin,
                "Invalidated block set, skipping derivation origin update"
            );
            return Ok(origin);
        }

        match self.db_provider.save_source_block(origin) {
            Ok(_) => Ok(origin),
            Err(StorageError::BlockOutOfOrder) => {
                debug!(
                    target: "supervisor::chain_processor",
                    chain_id = self.chain_id,
                    %origin,
                    "Block out of order detected, resetting managed node"
                );

                self.managed_node_sender.send(ManagedNodeCommand::Reset {}).await.map_err(
                    |err| {
                        warn!(
                            target: "supervisor::chain_processor::managed_node",
                            chain_id = self.chain_id,
                            %origin,
                            %err,
                            "Failed to send reset command to managed node"
                        );
                        ChainProcessorError::ChannelSendFailed(err.to_string())
                    },
                )?;
                Ok(origin)
            }
            Err(err) => {
                error!(
                    target: "supervisor::chain_processor",
                    chain_id = self.chain_id,
                    %origin,
                    %err,
                    "Failed to save source block during derivation origin update"
                );
                Err(err.into())
            }
        }
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
    use kona_interop::DerivedRefPair;
    use kona_protocol::BlockInfo;
    use kona_supervisor_storage::{DerivationStorageWriter, StorageError};
    use kona_supervisor_types::{BlockSeal, OutputV0, Receipts};
    use mockall::mock;

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
    );

    #[tokio::test]
    async fn test_handle_derivation_origin_update_triggers() {
        let mut mockdb = MockDb::new();
        let (tx, mut rx) = mpsc::channel(1);
        let mut state = ProcessorState::new();

        let origin =
            BlockInfo { number: 42, hash: B256::ZERO, parent_hash: B256::ZERO, timestamp: 123456 };

        let origin_clone = origin;
        mockdb.expect_save_source_block().returning(move |block_info: BlockInfo| {
            assert_eq!(block_info, origin_clone);
            Ok(())
        });

        let writer = Arc::new(mockdb);

        let handler = OriginHandler::new(
            1, // chain_id
            tx, writer,
        );

        let result = handler.handle(origin, &mut state).await;
        assert!(result.is_ok());

        // Ensure no command was sent
        assert!(rx.try_recv().is_err());
    }

    #[tokio::test]
    async fn test_handle_derivation_origin_update_block_out_of_order_triggers_reset() {
        let mut mockdb = MockDb::new();
        let (tx, mut rx) = mpsc::channel(1);
        let mut state = ProcessorState::new();

        let origin =
            BlockInfo { number: 42, hash: B256::ZERO, parent_hash: B256::ZERO, timestamp: 123456 };

        mockdb.expect_save_source_block().returning(|_| Err(StorageError::BlockOutOfOrder));

        let writer = Arc::new(mockdb);

        let handler = OriginHandler::new(1, tx, writer);

        let result = handler.handle(origin, &mut state).await;
        assert!(result.is_ok());

        // The handler should send the reset command
        if let Some(ManagedNodeCommand::Reset {}) = rx.recv().await {
            // Command received successfully
        } else {
            panic!("Expected Reset command");
        }
    }

    #[tokio::test]
    async fn test_handle_derivation_origin_update_reset_fails() {
        let mut mockdb = MockDb::new();
        let (tx, rx) = mpsc::channel(1);
        let mut state = ProcessorState::new();

        let origin =
            BlockInfo { number: 42, hash: B256::ZERO, parent_hash: B256::ZERO, timestamp: 123456 };

        mockdb.expect_save_source_block().returning(|_| Err(StorageError::BlockOutOfOrder));

        let writer = Arc::new(mockdb);

        drop(rx); // Simulate a send error by dropping the receiver

        let handler = OriginHandler::new(1, tx, writer);

        let result = handler.handle(origin, &mut state).await;
        assert!(result.is_err());
    }

    #[tokio::test]
    async fn test_handle_derivation_origin_update_other_storage_error() {
        let mut mockdb = MockDb::new();
        let (tx, mut rx) = mpsc::channel(1);
        let mut state = ProcessorState::new();

        let origin =
            BlockInfo { number: 42, hash: B256::ZERO, parent_hash: B256::ZERO, timestamp: 123456 };

        mockdb.expect_save_source_block().returning(|_| Err(StorageError::DatabaseNotInitialised));

        let writer = Arc::new(mockdb);

        let handler = OriginHandler::new(1, tx, writer);

        let result = handler.handle(origin, &mut state).await;
        assert!(result.is_err());

        // Ensure no command was sent
        assert!(rx.try_recv().is_err());
    }
}
